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

	"github.com/pydio/cells-client/v4/common"

	"github.com/gosuri/uiprogress"

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
}

func NewCrawler(ctx context.Context, target string, isSrcLocal bool) (*CrawlNode, error) {
	if isSrcLocal {
		target, _ = filepath.Abs(target)
		i, e := os.Stat(target)
		if e != nil {
			return nil, e
		}
		return NewLocalNode(target, i), nil
	} else {
		n, b := StatNode(ctx, target)
		if !b {
			return nil, fmt.Errorf("no node found at %s", target)
		}
		return NewRemoteNode(n), nil
	}
}

// NewLocalNode creates the base node for crawling in case of an upload.
func NewLocalNode(fullPath string, i os.FileInfo) *CrawlNode {
	fullPath = strings.TrimRight(fullPath, string(os.PathSeparator))
	n := &CrawlNode{
		IsLocal:  true,
		IsDir:    i.IsDir(),
		FullPath: fullPath,
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

func NewTarget(target string, source *CrawlNode, rename bool) *CrawlNode {
	c := &CrawlNode{
		IsLocal:  !source.IsLocal,
		IsDir:    source.IsDir,
		FullPath: target,
		RelPath:  "",
	}
	// For dirs, add source directory name, if we are not in the rename case:
	// in such case, target is already the full target path.
	if source.IsDir && !rename {
		c.FullPath = c.Join(c.FullPath, source.Base())
	}

	// Manage rename corner case for files:
	// we remove last part of the path that is the target file name
	if rename && !source.IsDir {
		// We must compute NewFileName first because it relies on the FullPath that is then impacted
		c.NewFileName = c.Base()
		c.FullPath = c.Dir()
	}

	return c
}

// Walk prepares the list of single upload/download nodes that we process in a second time.
func (c *CrawlNode) Walk(ctx context.Context, current ...string) (children []*CrawlNode, e error) {
	crt := ""
	if len(current) > 0 {
		crt = current[0]
	}

	// Source is a single file
	if !c.IsDir {
		c.RelPath = c.Base()
		children = append(children, c)
		return
	}

	if c.IsLocal {
		e = filepath.Walk(filepath.Join(c.FullPath, crt), func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasPrefix(filepath.Base(p), ".") {
				return nil
			}
			n := NewLocalNode(p, info)
			n.RelPath = strings.TrimPrefix(n.FullPath, c.FullPath)
			children = append(children, n)
			return nil
		})
	} else {
		nn, er := GetAllBulkMeta(ctx, path.Join(c.FullPath, crt, "*"))
		if er != nil {
			e = er
			return
		}
		for _, n := range nn {
			remote := NewRemoteNode(n)
			remote.RelPath = strings.TrimPrefix(remote.FullPath, c.FullPath)
			children = append(children, remote)
			if *n.Type == models.TreeNodeTypeCOLLECTION {
				cc, er := c.Walk(ctx, remote.RelPath)
				if er != nil {
					e = er
					return
				}
				children = append(children, cc...)
			}
		}
	}
	return
}

// MkdirAll prepares a recursive scp by first creating all necessary folders under the target root folder.
func (c *CrawlNode) MkdirAll(ctx context.Context, dd []*CrawlNode, pool *BarsPool) error {

	var createRoot bool
	var mm []*models.TreeNode
	if !c.IsLocal {
		// Remote : append root if required
		if tn, b := StatNode(ctx, c.FullPath); !b {
			mm = append(mm, &models.TreeNode{Path: c.FullPath, Type: models.NewTreeNodeType(models.TreeNodeTypeCOLLECTION)})
			createRoot = true
		} else if *tn.Type != models.TreeNodeTypeCOLLECTION {
			// target root is not a folder, fail fast.
			return fmt.Errorf("%s exists on the server and is not a folder, cannot upload there", c.FullPath)
		}
	} else {
		if _, e := os.Stat(c.FullPath); e != nil {
			if DryRun {
				fmt.Println("MkDir: \t", c.FullPath)
			} else if e1 := os.MkdirAll(c.FullPath, 0755); e1 != nil {
				return e1
			}
		}
	}
	for _, d := range dd {
		if !d.IsDir {
			continue
		}
		if d.RelPath == "" && createRoot {
			continue
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
	if !c.IsLocal && !DryRun && len(mm) > 0 {
		e := TreeCreateNodes(mm)
		if e != nil {
			return e
		}
		if pool != nil {
			for range mm {
				pool.Done()
			}
		}
		// TODO:  Stat all folders to make sure they are indexed ?
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
		bar := pool.Get(idx, int(barSize), d.Base())
		wg.Add(1)
		go func(src *CrawlNode, barId int) {
			defer func() {
				wg.Done()
				pool.Done()
				<-buf
			}()
			if !c.IsLocal {
				if e := c.upload(ctx, src, bar); e != nil {
					errs = append(errs, e)
				}
				if emptyFile {
					bar.Set(1)
				}
			} else {
				if e := c.download(ctx, src, bar); e != nil {
					errs = append(errs, e)
				}
				if emptyFile {
					bar.Set(1)
				}
			}
		}(d, idx)
	}
	wg.Wait()
	pool.Stop()
	return
}

func (c *CrawlNode) upload(ctx context.Context, src *CrawlNode, bar *uiprogress.Bar) error {
	file, e := os.Open(src.FullPath)
	if e != nil {
		return e
	}
	stats, _ := file.Stat()
	wrapper := &PgReader{
		Reader: file,
		Seeker: file,
		bar:    bar,
		total:  int(stats.Size()),
		double: true,
	}
	_, done := wrapper.CreateErrorChan()
	defer close(done)
	bName := src.RelPath
	if c.NewFileName != "" {
		bName = c.NewFileName
	}

	fullPath := c.Join(c.FullPath, bName)
	// Handle corner case when trying to upload a file and *folder* with same name already exists at target path
	if tn, b := StatNode(ctx, fullPath); b && *tn.Type == models.TreeNodeTypeCOLLECTION {
		// target root is not a folder, fail fast.
		return fmt.Errorf("cannot upload file to %s, a folder with same name already exists at target path", fullPath)
	}
	wrapper.double = false
	if stats.Size() <= common.UploadSwitchMultipart*(1024*1024) {
		if _, err := PutFile(ctx, fullPath, wrapper, false, wrapper.errChan); err != nil {
			return err
		}
	} else if err := uploadManager(ctx, stats, fullPath, wrapper, wrapper.errChan); err != nil {
		return err
	}
	return nil
}

func (c *CrawlNode) download(ctx context.Context, src *CrawlNode, bar *uiprogress.Bar) error {
	reader, length, e := GetFile(ctx, src.FullPath)
	if e != nil {
		return e
	}
	wrapper := &PgReader{
		Reader: reader,
		bar:    bar,
		total:  length,
	}
	bname := src.RelPath
	if c.NewFileName != "" {
		bname = c.NewFileName
	}
	downloadToLocation := c.Join(c.FullPath, bname)
	writer, e := os.OpenFile(downloadToLocation, os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer writer.Close()
	_, e = io.Copy(writer, wrapper)
	return e
}

// CopyAllVerbose performs the real transfer of files in parallel.
// It relies on the list that has been prepared during the Walk step,
// uses no progress bar and rather adds more logs.
func (c *CrawlNode) CopyAllVerbose(ctx context.Context, dd []*CrawlNode) (errs []error) {
	idx := -1
	buf := make(chan struct{}, PoolSize)
	wg := &sync.WaitGroup{}
	for _, d := range dd {
		if d.IsDir {
			continue
		}
		buf <- struct{}{}
		idx++
		wg.Add(1)
		go func(src *CrawlNode) {
			defer func() {
				wg.Done()
				<-buf
			}()
			if !c.IsLocal {
				if e := c.uploadVerbose(ctx, src); e != nil {
					errs = append(errs, e)
				}
			} else {
				if e := c.downloadVerbose(ctx, src); e != nil {
					errs = append(errs, e)
				}
			}
		}(d)
	}
	wg.Wait()
	return
}

func (c *CrawlNode) uploadVerbose(ctx context.Context, src *CrawlNode) error {
	file, e := os.Open(src.FullPath)
	if e != nil {
		return e
	}
	stats, e := file.Stat()
	if e != nil {
		fmt.Printf("[Error] could not stat file at %s, cause: %s", src.FullPath, e.Error())
		return e
	}
	bName := src.RelPath
	if c.NewFileName != "" {
		bName = c.NewFileName
	}

	fullPath := c.Join(c.FullPath, bName)
	// Handle corner case when trying to upload a file and *folder* with same name already exists at target path
	if tn, b := StatNode(ctx, fullPath); b && *tn.Type == models.TreeNodeTypeCOLLECTION {
		// target root is not a folder, fail fast.
		return fmt.Errorf("cannot upload file to %s, a folder with same name already exists at target path", fullPath)
	}
	if stats.Size() <= common.UploadSwitchMultipart*(1024*1024) {
		if _, err := PutFile(ctx, fullPath, file, false); err != nil {
			return err
		}
	} else if err := uploadManager(ctx, stats, fullPath, file); err != nil {
		return err
	}
	return nil
}

func (c *CrawlNode) downloadVerbose(ctx context.Context, src *CrawlNode) error {
	reader, length, e := GetFile(ctx, src.FullPath)
	if e != nil {
		return e
	}
	bName := src.RelPath
	if c.NewFileName != "" {
		bName = c.NewFileName
	}
	downloadToLocation := c.Join(c.FullPath, bName)
	writer, e := os.OpenFile(downloadToLocation, os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer writer.Close()
	written, e := io.Copy(writer, reader)
	if length != int(written) {
		fmt.Printf("[Warning] written size (%d) differs from expected length (%d) for %s\n", written, length, downloadToLocation)
	}
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

type BarsPool struct {
	*uiprogress.Progress
	showGlobal bool
	nodesBar   *uiprogress.Bar
}

func NewBarsPool(showGlobal bool, totalNodes int, refreshInterval time.Duration) *BarsPool {
	b := &BarsPool{}
	b.Progress = uiprogress.New()
	b.Progress.SetRefreshInterval(refreshInterval)
	b.showGlobal = showGlobal
	if showGlobal {
		b.nodesBar = b.AddBar(totalNodes)
		b.nodesBar.PrependCompleted()
		b.nodesBar.AppendFunc(func(b *uiprogress.Bar) string {
			if b.Current() == b.Total {
				return fmt.Sprintf("Transferred %d/%d files and folders (%s)", b.Current(), b.Total, b.TimeElapsedString())
			} else {
				return fmt.Sprintf("Transfering %d/%d files or folders", b.Current()+1, b.Total)
			}
		})
	}
	return b
}

func (b *BarsPool) Done() {
	if !b.showGlobal {
		return
	}
	b.nodesBar.Incr()
	if b.nodesBar.Current() == b.nodesBar.Total {
		// Finished, remove all bars
		b.Bars = []*uiprogress.Bar{b.nodesBar}
	}
}

func (b *BarsPool) Get(i int, total int, name string) *uiprogress.Bar {
	idx := i % PoolSize
	var nBars []*uiprogress.Bar
	if b.showGlobal {
		idx++
		nBars = append(nBars, b.nodesBar)
	}
	// Remove old bar
	for k, bar := range b.Bars {
		if k == idx || (b.showGlobal && bar == b.nodesBar) {
			continue
		}
		nBars = append(nBars, bar)
	}
	b.Bars = nBars
	bar := b.AddBar(total)
	bar.PrependCompleted()
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprint(name)
	})
	return bar
}

type PgReader struct {
	io.Reader
	io.Seeker
	bar   *uiprogress.Bar
	total int
	read  int

	double bool
	first  bool

	errChan chan error
}

func (r *PgReader) CreateErrorChan() (chan error, chan struct{}) {
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

func (r *PgReader) sendErr(err error) {
	r.bar.AppendFunc(func(b *uiprogress.Bar) string {
		return err.Error()
	})
}

func (r *PgReader) Read(p []byte) (n int, err error) {
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

func (r *PgReader) Seek(offset int64, whence int) (int64, error) {
	if r.double && r.first {
		r.read = r.total/2 + int(offset)/2
	} else {
		r.read = int(offset)
	}
	r.bar.Set(r.read)
	return r.Seeker.Seek(offset, whence)
}
