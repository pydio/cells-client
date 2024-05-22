package rest

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gosuri/uiprogress"

	"github.com/pydio/cells-client/v4/common"
	"github.com/pydio/cells-sdk-go/v5/models"
)

var (
	DryRun   bool
	PoolSize = 3
)

// CrawlNode enables processing the scp command step by step.
type CrawlNode struct {
	IsLocal bool

	IsDir       bool
	FullPath    string
	RelPath     string
	MTime       time.Time
	Size        int64
	NewFileName string

	os.FileInfo
	models.TreeNode

	// fixme
	s3Client   *s3.Client
	bucketName string
}

func NewCrawler(ctx context.Context, basePath string, isLocal bool) (*CrawlNode, error) {
	if isLocal {
		// We expect a clean absolute path to an existing file or folder on the local machine at this point
		fileInfo, e := os.Stat(basePath)
		if e != nil {
			return nil, e
		}
		return NewLocalNode(basePath, fileInfo), nil
	} else {
		n, b := StatNode(ctx, basePath)
		if !b {
			return nil, fmt.Errorf("no node found at %s", basePath)
		}
		return NewRemoteNode(n), nil
	}
}

// NewLocalNode creates the base node for crawling in case of an upload.
func NewLocalNode(absPath string, i os.FileInfo) *CrawlNode {
	n := &CrawlNode{
		IsLocal:  true,
		IsDir:    i.IsDir(),
		FullPath: absPath,
		RelPath:  filepath.Base(absPath),
		MTime:    i.ModTime(),
		Size:     i.Size(),
	}
	n.FileInfo = i
	return n
}

// NewRemoteNode creates the base node for crawling in case of a download.
func NewRemoteNode(t *models.TreeNode) *CrawlNode {
	n := &CrawlNode{
		IsDir:    t.Type != nil && *t.Type == models.TreeNodeTypeCOLLECTION,
		FullPath: strings.Trim(t.Path, "/"),
	}
	n.Size, _ = strconv.ParseInt(t.Size, 10, 64)
	unixTime, _ := strconv.ParseInt(t.MTime, 10, 32)
	n.MTime = time.Unix(unixTime, 0)
	n.TreeNode = *t
	return n
}

func NewTarget(ctx context.Context, target string, source *CrawlNode, rename bool) (*CrawlNode, error) {
	c := &CrawlNode{
		IsLocal:  !source.IsLocal,
		IsDir:    source.IsDir,
		FullPath: target,
		RelPath:  "",
	}

	// We kind of cache the client in the crawler as a quick and dirty way to ensure it is unique.
	s3Client, bucketName, e := GetS3Client(ctx)
	if e != nil {
		return nil, e
	}
	c.s3Client = s3Client
	c.bucketName = bucketName

	//// For dirs, add source directory name, if we are not in the rename case:
	//// in such case, target is already the full target path.
	//if source.IsDir && !rename {
	//	c.FullPath = c.Join(c.FullPath, source.Base())
	//}

	// TODO handle rename corner case for files:
	//if rename && !source.IsDir {
	//	// We must compute NewFileName first because it relies on the FullPath that is then impacted
	//	c.NewFileName = c.Base()
	//	c.FullPath = c.Dir()
	//}

	return c, nil
}

// Walk prepares the list of single upload/download nodes that we process in a second time.
func (c *CrawlNode) Walk(ctx context.Context, givenRelPath ...string) (toCreateNodes []*CrawlNode, e error) {
	relPath := ""

	if len(givenRelPath) == 0 {
		c.RelPath = c.Base()
		toCreateNodes = append(toCreateNodes, c)
		if !c.IsDir { // Source is a single file
			return
		} else {
			relPath = c.RelPath
		}
	} else {
		relPath = givenRelPath[0]
	}

	//if c.IsDir && !c.IsLocal {
	//	c.RelPath = c.Base()
	//	toCreateNodes = append(toCreateNodes, c)
	//}

	if c.IsLocal {
		rootPath := filepath.Join(c.FullPath)
		parentPath := filepath.Dir(rootPath)
		e = filepath.Walk(rootPath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Skip hidden file TODO make this OS independent
			if strings.HasPrefix(filepath.Base(p), ".") {
				return nil
			}
			n := NewLocalNode(p, info)
			n.RelPath = strings.TrimPrefix(p, parentPath+"/")
			toCreateNodes = append(toCreateNodes, n)
			return nil
		})
	} else {
		nn, er := GetAllBulkMeta(ctx, path.Join(c.FullPath, "*"))
		if er != nil {
			e = er
			return
		}
		for _, n := range nn {
			remote := NewRemoteNode(n)
			remote.RelPath = path.Join(relPath, filepath.Base(n.Path))
			toCreateNodes = append(toCreateNodes, remote)
			if *n.Type == models.TreeNodeTypeCOLLECTION {
				cc, er := remote.Walk(ctx, remote.RelPath)
				if er != nil {
					e = er
					return
				}
				toCreateNodes = append(toCreateNodes, cc...)
			}
		}
	}
	return
}

// MkdirAll prepares a recursive scp by first creating all necessary folders under the target root folder.
func (c *CrawlNode) MkdirAll(ctx context.Context, dd []*CrawlNode, pool *BarsPool) error {

	var createParent bool
	var mm []*models.TreeNode
	// Manage current folder
	if c.IsLocal {
		if _, e := os.Stat(c.FullPath); e != nil {
			// Create base folder if necessary
			if DryRun {
				fmt.Println("MkDir: \t", c.FullPath)
			} else if e1 := os.MkdirAll(c.FullPath, 0755); e1 != nil {
				return e1
			}
		}
	} else { //  Remote
		if tn, b := StatNode(ctx, c.FullPath); !b { // Also create remote parent if required
			mm = append(mm, &models.TreeNode{Path: c.FullPath, Type: models.NewTreeNodeType(models.TreeNodeTypeCOLLECTION)})
			createParent = true
		} else if *tn.Type != models.TreeNodeTypeCOLLECTION { // Sanity check
			// Target root is not a folder: failing fast
			return fmt.Errorf("%s exists on the server and is not a folder, cannot upload there", c.FullPath)
		}
	}
	// Manage descendants: local folders are created and remote are gathered in the mm array
	for _, d := range dd {
		if !d.IsDir {
			continue
		}
		if d.RelPath == "" && createParent {
			//continue
		}
		newFolder := c.Join(c.FullPath, d.RelPath)
		if DryRun {
			fmt.Println("MkDir: \t", newFolder)
			continue
		}
		if c.IsLocal {
			if e := os.MkdirAll(newFolder, 0755); e != nil {
				return e
			} else if pool != nil {
				pool.Done()
			}
		} else {
			mm = append(mm, &models.TreeNode{Path: newFolder, Type: models.NewTreeNodeType(models.TreeNodeTypeCOLLECTION)})
		}
	}
	if !DryRun && len(mm) > 0 {
		if !c.IsLocal {
			return createRemoteFolders(ctx, mm, pool)
		} else if pool == nil {
			fmt.Printf("... Created %d folders under %s\n", len(mm), c.FullPath)
		}
	}
	return nil
}

// CopyAll performs the real parallel transfers of file, after they have been prepared during the Walk step.
func (c *CrawlNode) CopyAll(ctx context.Context, dd []*CrawlNode, pool *BarsPool) (errs []error) {

	idx := -1
	buf := make(chan struct{}, PoolSize)
	wg := &sync.WaitGroup{}
	for _, d := range dd {
		if d.IsDir {
			continue
		}
		buf <- struct{}{}
		idx++
		barSize := d.Size
		emptyFile := false
		if barSize == 0 {
			emptyFile = true
			barSize = 1
		}
		wg.Add(1)
		var bar *uiprogress.Bar
		if pool != nil {
			bar = pool.Get(idx, int(barSize), d.Base())
		}
		go func(src *CrawlNode, barId int) {
			defer func() {
				wg.Done()
				if pool != nil {
					pool.Done()
				}
				<-buf
			}()
			if !c.IsLocal {
				if e := c.upload(ctx, src, bar); e != nil {
					contextualizedErr := fmt.Errorf("could not upload '%s' at '%s': %s", src.RelPath, c.FullPath, e.Error())
					errs = append(errs, contextualizedErr)
				}
				if emptyFile && bar != nil {
					_ = bar.Set(1)
				}
			} else {
				if e := c.download(ctx, src, bar); e != nil {
					contextualizedErr := fmt.Errorf("could not dowload '%s' to '%s': %s", src.FullPath, c.FullPath, e.Error())
					errs = append(errs, contextualizedErr)
				}
				if emptyFile && bar != nil {
					_ = bar.Set(1)
				}
			}
		}(d, idx)
	}
	wg.Wait()
	if pool != nil {
		pool.Stop()
	} else {
		fmt.Printf("... Transfer has terminated successfully\n")
	}
	return
}

func (c *CrawlNode) upload(ctx context.Context, src *CrawlNode, bar *uiprogress.Bar) error {
	file, e := os.Open(src.FullPath)
	if e != nil {
		return e
	}
	stats, e := file.Stat()
	if e != nil {
		fmt.Printf("[Error] could not stat file at %s, cause: %s", src.FullPath, e.Error())
		return e
	}

	var content io.ReadSeeker
	var errChan chan error
	if bar != nil {
		wrapper := &ReaderWithProgress{
			Reader: file,
			Seeker: file,
			bar:    bar,
			total:  int(stats.Size()),
			double: true,
		}
		var done chan struct{}
		errChan, done = wrapper.CreateErrorChan()
		defer close(done)
		wrapper.double = false
		content = wrapper
	} else {
		content = file
	}

	bName := src.RelPath
	// TODO re-handle new name
	//bName := filepath.Base(src.RelPath)
	//if c.NewFileName != "" {
	//	bName = c.NewFileName
	//}
	fullPath := c.Join(c.FullPath, bName)

	//// TODO Handle corner case when trying to upload a file and *folder* with same name already exists at target path
	//if tn, b := StatNode(ctx, fullPath); b && *tn.Type == models.TreeNodeTypeCOLLECTION {
	//	// target root is not a folder, fail fast.
	//	return fmt.Errorf("cannot upload *file* to %s, a *folder* with same name already exists at the target path", fullPath)
	//}

	var upErr error
	if stats.Size() <= common.UploadSwitchMultipart*(1024*1024) {
		if _, e = PutFile(ctx, c.s3Client, c.bucketName, fullPath, file, false); e != nil {
			upErr = fmt.Errorf("could not upload single part file %s: %s", fullPath, e.Error())
		}
		if bar == nil { // TODO this must be a debug level msg
			fmt.Printf("%s: OK\n", fullPath)
		}
	} else {
		upErr = s3Upload(ctx, c.s3Client, c.bucketName, fullPath, content, stats.Size(), bar == nil, errChan)
	}

	return upErr
}

func (c *CrawlNode) download(ctx context.Context, src *CrawlNode, bar *uiprogress.Bar) error {
	reader, length, e := GetFile(ctx, src.FullPath)
	if e != nil {
		return e
	}

	var content io.Reader
	if bar != nil {
		content = &ReaderWithProgress{
			Reader: reader,
			bar:    bar,
			total:  length,
		}
	} else {
		content = reader
	}
	targetName := src.RelPath
	//if c.NewFileName != "" {
	//	// TODO check if NewFileName is a Base Name or really a rel path at it is implied here
	//	targetName = c.NewFileName
	//}
	localTargetPath := c.Join(c.FullPath, targetName)
	writer, e := os.OpenFile(localTargetPath, os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer func(writer *os.File) {
		err := writer.Close()
		if err != nil && bar == nil { // Only in no progress mode. TODO rather use a logger
			fmt.Printf(
				"[Warning] could not close writer after creating %s: %s\n",
				localTargetPath,
				err.Error(),
			)
		}
	}(writer)
	_, e = io.Copy(writer, content)
	return e
}

func (c *CrawlNode) Join(p ...string) string {
	if os.PathSeparator != '/' {
		for i, pa := range p {
			if c.IsLocal {
				p[i] = strings.ReplaceAll(pa, "/", string(os.PathSeparator))
			} else {
				p[i] = strings.ReplaceAll(pa, string(os.PathSeparator), "/")
			}
		}
	}
	if c.IsLocal {
		return filepath.Join(p...)
	} else {
		return path.Join(p...)
	}
}

func (c *CrawlNode) Base() string {
	if c.IsLocal {
		return filepath.Base(c.FullPath)
	} else {
		return path.Base(c.FullPath)
	}
}

func (c *CrawlNode) Dir() string {
	if c.IsLocal {
		return filepath.Dir(c.FullPath)
	} else {
		return path.Dir(c.FullPath)
	}
}

type ReaderWithProgress struct {
	io.Reader
	io.Seeker
	bar   *uiprogress.Bar
	total int
	read  int

	double bool
	first  bool

	errChan chan error
}

func (r *ReaderWithProgress) CreateErrorChan() (chan error, chan struct{}) {
	done := make(chan struct{}, 1)
	r.errChan = make(chan error)
	go func() {
		for {
			select {
			case e := <-r.errChan:
				r.sendErr(e)
			case <-done:
				close(r.errChan)
				return
			}
		}
	}()
	return r.errChan, done
}

func (r *ReaderWithProgress) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if err == nil {
		if r.double {
			r.read += n / 2
		} else {
			r.read += n
		}
		r.bar.Set(r.read)
	} else if err == io.EOF {
		if r.double && !r.first {
			r.first = true
			r.bar.Set(r.total / 2)
		} else {
			r.bar.Set(r.total)
		}
	}
	return
}

func (r *ReaderWithProgress) Seek(offset int64, whence int) (int64, error) {
	if r.double && r.first {
		r.read = r.total/2 + int(offset)/2
	} else {
		r.read = int(offset)
	}
	r.bar.Set(r.read)
	return r.Seeker.Seek(offset, whence)
}

func (r *ReaderWithProgress) sendErr(err error) {
	r.bar.AppendFunc(func(b *uiprogress.Bar) string {
		return err.Error()
	})
}

//func (c *CrawlNode) uploadVerbose(ctx context.Context, src *CrawlNode) error {
//	file, e := os.Open(src.FullPath)
//	if e != nil {
//		return e
//	}
//	stats, e := file.Stat()
//	if e != nil {
//		fmt.Printf("[Error] could not stat file at %s, cause: %s", src.FullPath, e.Error())
//		return e
//	}
//	bName := src.RelPath
//	if c.NewFileName != "" {
//		bName = c.NewFileName
//	}
//
//	fullPath := c.Join(c.FullPath, bName)
//	// Handle corner case when trying to upload a file and *folder* with same name already exists at target path
//	if tn, b := StatNode(ctx, fullPath); b && *tn.Type == models.TreeNodeTypeCOLLECTION {
//		// target root is not a folder, fail fast.
//		return fmt.Errorf("cannot upload file to %s, a folder with same name already exists at target path", fullPath)
//	}
//	if stats.Size() <= common.UploadSwitchMultipart*(1024*1024) {
//		if _, err := PutFile(ctx, fullPath, file, false); err != nil {
//			return err
//		}
//	} else if err := uploadManager(ctx, stats, fullPath, file, true); err != nil {
//		return err
//	}
//	return nil
//}

//// CopyAllVerbose performs the real transfer of files in parallel.
//// It relies on the list that has been prepared during the Walk step,
//// uses no progress bar and rather adds more logs.
//func (c *CrawlNode) CopyAllVerbose(ctx context.Context, dd []*CrawlNode) (errs []error) {
//	idx := -1
//	buf := make(chan struct{}, PoolSize)
//	wg := &sync.WaitGroup{}
//	for _, d := range dd {
//		if d.IsDir {
//			continue
//		}
//		buf <- struct{}{}
//		idx++
//		wg.Add(1)
//		go func(src *CrawlNode) {
//			defer func() {
//				wg.Done()
//				<-buf
//			}()
//			if !c.IsLocal {
//				if e := c.uploadVerbose(ctx, src); e != nil {
//					errs = append(errs, e)
//				}
//			} else {
//				if e := c.downloadVerbose(ctx, src); e != nil {
//					errs = append(errs, e)
//				}
//			}
//		}(d)
//	}
//	wg.Wait()
//	return
//}
//
//func (c *CrawlNode) downloadVerbose(ctx context.Context, src *CrawlNode) error {
//	reader, length, e := GetFile(ctx, src.FullPath)
//	if e != nil {
//		return e
//	}
//	bName := src.RelPath
//	if c.NewFileName != "" {
//		bName = c.NewFileName
//	}
//	downloadToLocation := c.Join(c.FullPath, bName)
//	writer, e := os.OpenFile(downloadToLocation, os.O_CREATE|os.O_WRONLY, 0644)
//	if e != nil {
//		return e
//	}
//	defer writer.Close()
//	written, e := io.Copy(writer, reader)
//	if length != int(written) {
//		fmt.Printf("[Warning] written size (%d) differs from expected length (%d) for %s\n", written, length, downloadToLocation)
//	}
//	return e
//}
