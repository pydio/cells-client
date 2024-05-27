package rest

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/pydio/cells-sdk-go/v5/client/tree_service"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosuri/uiprogress"

	"github.com/pydio/cells-client/v4/common"
	"github.com/pydio/cells-sdk-go/v5/models"
)

var (
	// TODO this must be defined and managed by the SdkClient
	DryRun   bool
	PoolSize = 3
)

// CrawlNode enables processing the scp command step by step.
type CrawlNode struct {
	IsLocal   bool
	sdkClient *SdkClient

	IsDir       bool
	FullPath    string
	RelPath     string
	MTime       time.Time
	Size        int64
	NewFileName string

	needMerge bool

	os.FileInfo
	models.TreeNode
}

func NewCrawler(ctx context.Context, sdkClient *SdkClient, basePath string, isLocal bool) (*CrawlNode, error) {
	if isLocal {
		// We expect a clean absolute path to an existing file or folder on the local machine at this point
		fileInfo, e := os.Stat(basePath)
		if e != nil {
			return nil, e
		}
		return NewLocalBaseNode(sdkClient, basePath, fileInfo), nil
	} else {
		n, b := sdkClient.StatNode(ctx, basePath)
		if !b {
			return nil, fmt.Errorf("no node found at %s", basePath)
		}
		return NewRemoteNode(sdkClient, n), nil
	}
}

// NewLocalBaseNode creates the base node for crawling in case of an upload.
func NewLocalBaseNode(sdkClient *SdkClient, absPath string, i os.FileInfo) *CrawlNode {
	n := &CrawlNode{
		sdkClient: sdkClient,
		IsLocal:   true,
		IsDir:     i.IsDir(),
		FullPath:  absPath,
		RelPath:   filepath.Base(absPath),
		MTime:     i.ModTime(),
		Size:      i.Size(),
	}
	n.FileInfo = i
	return n
}

// NewLocalNode creates while crawling local tree before transfer.
func NewLocalNode(sdkClient *SdkClient, absPath, relpath string, i os.FileInfo) *CrawlNode {
	n := &CrawlNode{
		sdkClient: sdkClient,
		IsLocal:   true,
		IsDir:     i.IsDir(),
		FullPath:  absPath,
		RelPath:   relpath,
		MTime:     i.ModTime(),
		Size:      i.Size(),
	}
	n.FileInfo = i
	return n
}

// NewRemoteNode creates the base node for crawling in case of a download.
func NewRemoteNode(sdkClient *SdkClient, t *models.TreeNode) *CrawlNode {
	n := &CrawlNode{
		sdkClient: sdkClient,
		IsDir:     t.Type != nil && *t.Type == models.TreeNodeTypeCOLLECTION,
		FullPath:  strings.Trim(t.Path, "/"),
	}
	n.Size, _ = strconv.ParseInt(t.Size, 10, 64)
	unixTime, _ := strconv.ParseInt(t.MTime, 10, 32)
	n.MTime = time.Unix(unixTime, 0)
	n.TreeNode = *t
	return n
}

func NewTarget(sdkClient *SdkClient, target string, isLocal, isDir, merge bool) *CrawlNode {
	return &CrawlNode{
		sdkClient: sdkClient,
		needMerge: merge,
		IsLocal:   isLocal,
		IsDir:     isDir,
		FullPath:  target,
		RelPath:   "",
	}
}

func (c *CrawlNode) targetChild(name string, isDir bool) *CrawlNode {

	return &CrawlNode{
		sdkClient: c.sdkClient,
		needMerge: c.needMerge,
		IsLocal:   c.IsLocal,
		IsDir:     isDir,
		FullPath:  path.Join(c.FullPath, name),
		RelPath:   path.Join(c.RelPath, name),
	}
}

// Walk prepares the list of single upload/download nodes that we process in a second time.
func (c *CrawlNode) Walk(ctx context.Context, target *CrawlNode) (
	toTransfer, toCreate, toDelete []*CrawlNode, e error) {
	if c.IsLocal {
		return c.localWalk(ctx, target)
	} else {
		return c.remoteWalk(ctx, target)
	}
}

// localWalk prepares the list of single upload nodes that we process in a second time.
// we cannot use the native walk method to be able to merge more efficiently (we skip merge check on descendants as soon as a folder must not be merged)
func (c *CrawlNode) localWalk(ctx context.Context, targetFolder *CrawlNode, givenRelPath ...string) (
	toTransfer, toCreate, toDelete []*CrawlNode, err error) {

	relPath := ""

	currTargetFolder := targetFolder

	if len(givenRelPath) == 0 {
		c.RelPath = c.base()
		toTransfer, toCreate, toDelete, currTargetFolder, err = c.checkRemoteTarget(ctx, c, targetFolder)
		if err != nil {
			return
		}
		if !c.IsDir { // Source is a single file
			return
		}
		relPath = c.RelPath
	} else {
		relPath = givenRelPath[0]
	}

	// Open local current directory for listing
	dir, err2 := os.Open(c.FullPath)
	if err2 != nil {
		err = err2
		return
	}
	defer func(dir *os.File) {
		_ = dir.Close()
		// TODO check this?
		//if err != nil {
		//
		//}
	}(dir)
	files, err2 := dir.Readdir(-1) // -1 means to read all the files
	if err2 != nil {
		err = err2
		return
	}

	// Iterate over the files
	for _, fileinfo := range files {
		fullPath := filepath.Join(c.FullPath, fileinfo.Name())
		relpath := path.Join(relPath, fileinfo.Name())
		currLocal := NewLocalNode(c.sdkClient, fullPath, relpath, fileinfo)

		// Check current node and append where necessary
		var targetChild *CrawlNode
		t, c, d, targetChild, err2 := c.checkLocalTarget(currLocal, currTargetFolder)
		if err2 != nil {
			err = err2 // fail fast
			return
		}

		if len(t) > 0 {
			toTransfer = append(toTransfer, t...)
		}
		if len(c) > 0 {
			toCreate = append(toCreate, c...)
		}
		if len(d) > 0 {
			toDelete = append(toDelete, d...)
		}

		if fileinfo.IsDir() {
			// walk recursively
			t2, c2, d2, err2 := currLocal.localWalk(ctx, targetChild, relpath)
			if err2 != nil { // fail fast
				return
			}
			if len(t2) > 0 {
				toTransfer = append(toTransfer, t2...)
			}
			if len(c2) > 0 {
				toCreate = append(toCreate, c2...)
			}
			if len(d2) > 0 {
				toDelete = append(toDelete, d2...)
			}
		}
	}

	return

	/*


		if !c.IsDir { // Source is a single file
			c.RelPath = c.base()
			toCreateNodes = append(toCreateNodes, c)
		}

		rootPath := filepath.Join(c.FullPath)
		parentPath := filepath.Dir(rootPath)
		e = filepath.WalkDir(rootPath, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}

			// Skip hidden file TODO make this OS independent
			if strings.HasPrefix(filepath.Base(p), ".") {
				return nil
			}
			n := NewLocalBaseNode(c.sdkClient, p, info)
			n.RelPath = strings.TrimPrefix(p, parentPath+"/")
			toCreateNodes = append(toCreateNodes, n)
			return nil
		})
		return

	*/
}

// remoteWalk prepares recursively the list of single download nodes that we process in a second time.
func (c *CrawlNode) remoteWalk(ctx context.Context, targetFolder *CrawlNode, givenRelPath ...string) (
	toTransfer, toCreate, toDelete []*CrawlNode, err error) {
	relPath := ""

	currTargetFolder := targetFolder

	if len(givenRelPath) == 0 {
		c.RelPath = c.base()
		toTransfer, toCreate, toDelete, currTargetFolder, err = c.checkLocalTarget(c, targetFolder)
		if !c.IsDir { // Source is a single file
			return
		}
		relPath = c.RelPath
	} else {
		relPath = givenRelPath[0]
	}

	nn, err2 := c.sdkClient.GetAllBulkMeta(ctx, path.Join(c.FullPath, "*"))
	if err2 != nil {
		err = err2
		return
	}
	for _, n := range nn {
		remote := NewRemoteNode(c.sdkClient, n)
		remote.RelPath = path.Join(relPath, filepath.Base(n.Path))

		// Check current node and append where necessary
		var targetChild *CrawlNode
		toTransfer, toCreate, toDelete, targetChild, err = c.checkLocalTarget(remote, currTargetFolder)
		if err != nil { // fail fast
			return
		}

		if remote.IsDir { // walk recursively
			toTransfer, toCreate, toDelete, err = c.remoteWalk(ctx, targetChild, remote.RelPath)
			if err != nil { // fail fast
				return
			}
		}
	}
	return
}

func (c *CrawlNode) checkLocalTarget(src *CrawlNode, targetFolder *CrawlNode) (
	toTransfer, toCreate, toDelete []*CrawlNode, targetChild *CrawlNode, err error) {
	if !src.IsDir { // want to download a file
		if targetFolder == nil { // no need to merge
			toTransfer = append(toTransfer, src)
		} else {
			targetChild = targetFolder.targetChild(src.base(), false)
			info, err2 := os.Stat(targetChild.FullPath)
			if err2 != nil { // Nothing found at this path => we can DL
				toTransfer = append(toTransfer, src)
			} else if info.IsDir() { // Got a directory, must be removed before trying to force DL
				toDelete = append(toTransfer, targetChild)
				toTransfer = append(toTransfer, src)
			} else { // We erase the local file
				toTransfer = append(toTransfer, src)
			}
		}
		return
	}
	if targetFolder == nil { // no need to merge
		toCreate = append(toCreate, src)
	} else {
		targetChild = targetFolder.targetChild(src.base(), true)
		info, err2 := os.Stat(targetChild.FullPath)
		if err2 != nil { // Nothing found at this path => we can create folder
			toCreate = append(toCreate, src)
			targetChild = nil // after this point, no need to check for merging: we are in a new subtree
		} else if info.IsDir() {
			// Got a directory, and we are already merging: nothing to do.
		} else { // We erase the local file
			toDelete = append(toDelete, targetChild)
			toCreate = append(toCreate, targetChild)
		}
	}
	return
}

func (c *CrawlNode) checkRemoteTarget(ctx context.Context, src *CrawlNode, targetFolder *CrawlNode) (
	toTransfer, toCreate, toDelete []*CrawlNode, targetChild *CrawlNode, err error) {
	if !src.IsDir { // want to Upload a file
		if targetFolder == nil { // no need to merge
			toTransfer = append(toTransfer, src)
		} else {
			targetChild = targetFolder.targetChild(src.base(), false)
			treeNode, found := c.sdkClient.StatNode(ctx, targetChild.FullPath)
			if !found { // Nothing found at this path => we can DL
				toTransfer = append(toTransfer, src)
			} else if treeNode.Type != nil && *treeNode.Type == models.TreeNodeTypeCOLLECTION { // Got a directory, must be removed before trying to force DL
				toDelete = append(toTransfer, targetChild)
				toTransfer = append(toTransfer, src)
			} else { // We overwrite the remote file
				toTransfer = append(toTransfer, src)
			}
		}
		return
	}
	if targetFolder == nil { // no need to merge
		toCreate = append(toCreate, src)
	} else {
		targetChild = targetFolder.targetChild(src.base(), true)
		treeNode, found := c.sdkClient.StatNode(ctx, targetChild.FullPath)
		if !found { // Nothing found at this path => we can create folder
			toCreate = append(toCreate, src)
			targetChild = nil // after this point, no need to check for merging: we are in a new subtree
		} else if treeNode.Type != nil && *treeNode.Type == models.TreeNodeTypeCOLLECTION {
			// Got a directory, and we are already merging: nothing to do.
		} else { // We erase the remote file
			toDelete = append(toDelete, targetChild)
			toCreate = append(toCreate, targetChild)
		}
	}
	return
}

// DeleteForMerge first explicitly delete problematic targets before creating folders and transferring files.
func (c *CrawlNode) DeleteForMerge(ctx context.Context, dd []*CrawlNode, pool *BarsPool) error {
	if DryRun {
		c.dryRunDelete(dd)
		return nil
	} else if c.IsLocal {
		return c.deleteLocalItems(dd, pool)
	} else {
		return c.deleteRemoteItems(ctx, dd, pool)
	}
}

func (c *CrawlNode) deleteLocalItems(dd []*CrawlNode, pool *BarsPool) error {
	for _, d := range dd {
		toDelete := c.join(c.FullPath, d.RelPath)
		if e := os.RemoveAll(toDelete); e != nil {
			return e
		} else if pool != nil {
			pool.Done()
		}
		fmt.Println("Deleted: \t", toDelete)
	}
	return nil
}

func (c *CrawlNode) deleteRemoteItems(ctx context.Context, dd []*CrawlNode, pool *BarsPool) error {
	for i := 0; i < len(dd); i += pageSize {
		end := i + pageSize
		if end > len(dd) {
			end = len(dd)
		}
		subArray := dd[i:end]

		var mm []string

		for _, d := range subArray {
			newFolder := path.Join(c.FullPath, d.RelPath)
			mm = append(mm, newFolder)
		}

		_, err := c.sdkClient.DeleteNodes(ctx, mm, true)
		if err != nil {
			return errors.Errorf("could not delete nodes: %s", err.Error())
		}
		// TODO: ensure jobs have terminated
		if pool != nil {
			for range subArray {
				pool.Done()
			}
		} else { // verbose mode
			fmt.Printf("... Deleted %d nodes in the remote server\n", end)
		}
	}
	return nil
}

// CreateFolders prepares a recursive scp by first creating all necessary folders under the target root folder.
func (c *CrawlNode) CreateFolders(ctx context.Context, dd []*CrawlNode, pool *BarsPool) error {

	//var createParent bool
	//var mm []*models.TreeNode
	//// Manage current folder
	//if c.IsLocal {
	//	if _, e := os.Stat(c.FullPath); e != nil {
	//		// Create base folder if necessary
	//		if DryRun {
	//			fmt.Println("MkDir: \t", c.FullPath)
	//		} else if e1 := os.MkdirAll(c.FullPath, 0755); e1 != nil {
	//			return e1
	//		}
	//	}
	//} else { //  Remote
	//	if tn, b := c.sdkClient.StatNode(ctx, c.FullPath); !b { // Also create remote parent if required
	//		mm = append(mm, &models.TreeNode{Path: c.FullPath, Type: models.NewTreeNodeType(models.TreeNodeTypeCOLLECTION)})
	//		createParent = true
	//	} else if *tn.Type != models.TreeNodeTypeCOLLECTION { // Sanity check
	//		// Target root is not a folder: failing fast
	//		return fmt.Errorf("%s exists on the server and is not a folder, cannot upload there", c.FullPath)
	//	}
	//}

	if DryRun {
		c.dryRunCreate(dd)
		return nil
	} else if c.IsLocal {
		return c.createLocalFolders(dd, pool)
	} else {
		return c.createRemoteFolders(ctx, dd, pool)
	}
}

// TransferAll performs the real parallel transfer of files, after they have been prepared during the Walk step.
func (c *CrawlNode) TransferAll(ctx context.Context, dd []*CrawlNode, pool *BarsPool) (errs []error) {

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
			bar = pool.Get(idx, int(barSize), d.base())
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
	//bName := filepath.base(src.RelPath)
	//if c.NewFileName != "" {
	//	bName = c.NewFileName
	//}
	fullPath := c.join(c.FullPath, bName)

	//// TODO Handle corner case when trying to upload a file and *folder* with same name already exists at target path
	//if tn, b := StatNode(ctx, fullPath); b && *tn.Type == models.TreeNodeTypeCOLLECTION {
	//	// target root is not a folder, fail fast.
	//	return fmt.Errorf("cannot upload *file* to %s, a *folder* with same name already exists at the target path", fullPath)
	//}

	var upErr error
	if stats.Size() <= common.UploadSwitchMultipart*(1024*1024) {
		if _, e = c.sdkClient.PutFile(ctx, fullPath, file, false); e != nil {
			upErr = fmt.Errorf("could not upload single part file %s: %s", fullPath, e.Error())
		}
		if bar == nil { // TODO this must be a debug level msg
			fmt.Printf("\t%s: uploaded\n", fullPath)
		}
	} else {
		upErr = c.sdkClient.s3Upload(ctx, fullPath, content, stats.Size(), bar == nil, errChan)
	}

	return upErr
}

func (c *CrawlNode) download(ctx context.Context, src *CrawlNode, bar *uiprogress.Bar) error {
	reader, length, e := c.sdkClient.GetFile(ctx, src.FullPath)
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
	//	// TODO check if NewFileName is a base Name or really a rel path at it is implied here
	//	targetName = c.NewFileName
	//}
	localTargetPath := c.join(c.FullPath, targetName)
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

// createLocalFolders creates necessary folders on the client machine.
func (c *CrawlNode) createLocalFolders(toCreateDirs []*CrawlNode, pool *BarsPool) error {
	// TODO handle parent folder
	for _, d := range toCreateDirs {
		newFolder := c.join(c.FullPath, d.RelPath)
		if e := os.MkdirAll(newFolder, 0755); e != nil {
			return e
		} else if pool != nil {
			pool.Done()
		}
	}
	return nil
}

// createRemoteFolders creates necessary folders on the distant server.
func (c *CrawlNode) createRemoteFolders(ctx context.Context, toCreateDirs []*CrawlNode, pool *BarsPool) error {

	for i := 0; i < len(toCreateDirs); i += pageSize {
		end := i + pageSize
		if end > len(toCreateDirs) {
			end = len(toCreateDirs)
		}
		subArray := toCreateDirs[i:end]

		var mm []*models.TreeNode

		for _, d := range subArray {
			newFolder := c.join(c.FullPath, d.RelPath)
			mm = append(mm, &models.TreeNode{Path: newFolder, Type: models.NewTreeNodeType(models.TreeNodeTypeCOLLECTION)})
		}

		params := tree_service.NewCreateNodesParams()
		params.Body = &models.RestCreateNodesRequest{
			Nodes:     mm,
			Recursive: false,
		}
		_, err := c.sdkClient.GetApiClient().TreeService.CreateNodes(params)
		if err != nil {
			return errors.Errorf("could not create folders: %s", err.Error())
		}
		// TODO:  Stat all folders to make sure they are indexed ?
		if pool != nil {
			for range subArray {
				pool.Done()
			}
		} else { // verbose mode
			fmt.Printf("... Created %d folders on remote server\n", end)
		}
	}
	return nil
}

// dryRunCreate simply list the folders that should be created.
func (c *CrawlNode) dryRunCreate(toCreateDirs []*CrawlNode) {
	// TODO handle parent folder
	for _, d := range toCreateDirs {
		newFolder := c.join(c.FullPath, d.RelPath)
		fmt.Println("MkDir: \t", newFolder)
	}
}

// dryRunDelete simply list the folders that should be deleted.
func (c *CrawlNode) dryRunDelete(toDeleteItems []*CrawlNode) {
	for _, d := range toDeleteItems {
		toDelete := c.join(c.FullPath, d.RelPath)
		fmt.Println("Delete: \t", toDelete)
	}
}

// Local path helpers for the crawler.

func (c *CrawlNode) join(p ...string) string {
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

func (c *CrawlNode) base() string {
	if c.IsLocal {
		return filepath.Base(c.FullPath)
	} else {
		return path.Base(c.FullPath)
	}
}

func (c *CrawlNode) dir() string {
	if c.IsLocal {
		return filepath.Dir(c.FullPath)
	} else {
		return path.Dir(c.FullPath)
	}
}
