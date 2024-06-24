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

	"github.com/gosuri/uiprogress"
	"github.com/pkg/errors"

	"github.com/pydio/cells-sdk-go/v5/client/tree_service"
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
		//if !scpNoProgress {
		//	fmt.Println("") // Add a line to reduce glitches in the terminal
		//}
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
func NewLocalNode(sdkClient *SdkClient, absPath, relPath string, i os.FileInfo) *CrawlNode {
	n := &CrawlNode{
		sdkClient: sdkClient,
		IsLocal:   true,
		IsDir:     i.IsDir(),
		FullPath:  absPath,
		RelPath:   relPath,
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

// Walk prepares the list of single upload/download nodes that we process in a second time.
func (c *CrawlNode) Walk(ctx context.Context, target *CrawlNode) (
	toTransfer, toCreate, toDelete []*CrawlNode, err error) {
	var tt, tc, td []*CrawlNode
	if c.IsLocal {
		err = c.localWalk(ctx, target, &tt, &tc, &td)
	} else {
		err = c.remoteWalk(ctx, target, &tt, &tc, &td)
	}
	return tt, tc, td, err
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

// localWalk prepares the list of single upload nodes that we process in a second time.
// we cannot use the native walk method to be able to merge more efficiently (we skip merge check on descendants as soon as a folder must not be merged)
func (c *CrawlNode) localWalk(ctx context.Context, targetFolder *CrawlNode,
	tt, tc, td *[]*CrawlNode, givenRelPath ...string) (err error) {

	relPath := ""
	currTargetFolder := targetFolder

	if len(givenRelPath) == 0 {
		c.RelPath = c.base()
		currTargetFolder, err = c.checkRemoteTarget(ctx, c, targetFolder, tt, tc, td)
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
		_ = dir.Close() // TODO Check if ignoring this has side effects
	}(dir)
	files, err2 := dir.Readdir(-1) // -1 means to read all the files
	if err2 != nil {
		err = err2
		return
	}

	// Iterate over the files
	for _, fileInfo := range files {
		fullPath := filepath.Join(c.FullPath, fileInfo.Name())
		relPath := path.Join(relPath, fileInfo.Name())
		currLocal := NewLocalNode(c.sdkClient, fullPath, relPath, fileInfo)

		// Check current node and append where necessary
		targetChild, err3 := c.checkRemoteTarget(ctx, currLocal, currTargetFolder, tt, tc, td)
		if err3 != nil {
			err = err3
			return
		}

		if fileInfo.IsDir() {
			// walk recursively
			err4 := currLocal.localWalk(ctx, targetChild, tt, tc, td, relPath)
			if err4 != nil { // fail fast
				return
			}
		}
	}
	return
}

// remoteWalk prepares recursively the lists of nodes that we process in a second time.
// If we are in a merging process and when c is a directory, we also ensure that child folders also need check and merge.
func (c *CrawlNode) remoteWalk(ctx context.Context, targetFolder *CrawlNode,
	tt, tc, td *[]*CrawlNode, givenRelPath ...string) (err error) {
	relPath := ""

	currTargetFolder := targetFolder

	if len(givenRelPath) == 0 {
		c.RelPath = c.base()
		currTargetFolder, err = c.checkLocalTarget(c, targetFolder, tt, tc, td)
		if !c.IsDir { // Source is a single file
			return
		}
		Log.Infoln("Walking remote tree to prepare download")
		relPath = c.RelPath
	} else {
		relPath = givenRelPath[0]
	}

	// Log.Debugln("About to get bulk meta for", c.FullPath)
	nn, err2 := c.sdkClient.GetAllBulkMeta(ctx, path.Join(c.FullPath, "*"))
	if err2 != nil {
		err = err2
		return
	}
	// Log.Debugln("Now iterating over children")
	for _, n := range nn {
		// Prepare current node
		remote := NewRemoteNode(c.sdkClient, n)
		remote.RelPath = path.Join(relPath, filepath.Base(n.Path))
		// Check and append where necessary
		targetChild, err3 := c.checkLocalTarget(remote, currTargetFolder, tt, tc, td)
		if err3 != nil { // fail fast
			return
		}
		// walk recursively
		if remote.IsDir {
			err = remote.remoteWalk(ctx, targetChild, tt, tc, td, remote.RelPath)
			if err != nil { // fail fast
				return
			}
		}
	}
	return
}

// checkLocalTarget compares a remote node to the local target where it should be downloaded and append
// necessary nodes to the array for process on the second pass.
// If we are in a merging process and when c is a directory, we also ensure that child folders
// also need checking and merge, otherwise, we return a nil targetChild that will prevent further checks down the tree.
func (c *CrawlNode) checkLocalTarget(src *CrawlNode, targetFolder *CrawlNode, tt, tc, td *[]*CrawlNode) (
	targetChild *CrawlNode, err error) {
	if !src.IsDir { // want to download a file
		if targetFolder == nil { // no need to merge
			*tt = append(*tt, src)
		} else {
			targetChild = targetFolder.targetChild(src.base(), false)
			info, err2 := os.Stat(targetChild.FullPath)
			if err2 != nil { // Nothing found at this path => we can DL
				*tt = append(*tt, src)
			} else if info.IsDir() { // Got a directory, must be removed before trying to force DL
				*td = append(*td, targetChild)
				*tt = append(*tt, src)
			} else { // We erase the local file
				*tt = append(*tt, src)
			}
		}
		return
	}
	if targetFolder == nil { // no need to merge
		*tc = append(*tc, src)
	} else {
		targetChild = targetFolder.targetChild(src.base(), true)
		info, err2 := os.Stat(targetChild.FullPath)
		if err2 != nil { // Nothing found at this path => we can create folder
			*tc = append(*tc, src)
			targetChild = nil // after this point, no need to check for merging: we are in a new subtree
		} else if info.IsDir() {
			// Got a directory, and we are already merging: nothing to do.
		} else { // We erase the local file
			*td = append(*td, targetChild)
			*tc = append(*tc, src)
		}
	}
	return
}

func (c *CrawlNode) checkRemoteTarget(ctx context.Context, src *CrawlNode, targetFolder *CrawlNode,
	toTransfer, toCreate, toDelete *[]*CrawlNode) (targetChild *CrawlNode, err error) {
	if !src.IsDir { // want to Upload a file
		if targetFolder == nil { // no need to merge
			*toTransfer = append(*toTransfer, src)
		} else {
			targetChild = targetFolder.targetChild(src.base(), false)
			treeNode, found := c.sdkClient.StatNode(ctx, targetChild.FullPath)
			if !found { // Nothing found at this path => we can DL
				*toTransfer = append(*toTransfer, src)
			} else if treeNode.Type != nil && *treeNode.Type == models.TreeNodeTypeCOLLECTION { // Got a directory, must be removed before trying to force DL
				*toDelete = append(*toTransfer, targetChild)
				*toTransfer = append(*toTransfer, src)
			} else { // We overwrite the remote file
				*toTransfer = append(*toTransfer, src)
			}
		}
		return
	}
	if targetFolder == nil { // no need to merge
		*toCreate = append(*toCreate, src)
	} else {
		targetChild = targetFolder.targetChild(src.base(), true)
		treeNode, found := c.sdkClient.StatNode(ctx, targetChild.FullPath)
		if !found { // Nothing found at this path => we can create folder
			*toCreate = append(*toCreate, src)
			targetChild = nil // after this point, no need to check for merging: we are in a new subtree
		} else if treeNode.Type != nil && *treeNode.Type == models.TreeNodeTypeCOLLECTION {
			// Got a directory, and we are already merging: nothing to do.
		} else { // We erase the remote file
			*toDelete = append(*toTransfer, targetChild)
			*toTransfer = append(*toTransfer, src)
		}
	}
	return
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
			Log.Infof("... Deleted %d nodes in the remote server\n", end)
		}
	}
	return nil
}

// CreateFolders prepares a recursive scp by first creating all necessary folders under the target root folder.
func (c *CrawlNode) CreateFolders(_ context.Context, target *CrawlNode, dd []*CrawlNode, pool *BarsPool) error {
	if DryRun {
		c.dryRunCreate(dd)
		return nil
	} else if c.IsLocal {
		return c.createLocalFolders(dd, pool)
	} else {
		return c.createRemoteFolders(target, dd, pool)
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
			if len(errs) > 0 { // We skip launching new jobs as soon as we get an error
				Log.Debugf("... Skipping transfer for %s", src.FullPath)
				wg.Done()
				if pool != nil {
					pool.Done()
				}
				<-buf
			}

			defer func() {
				// TODO also find a way to display error messages with the pool
				if pool == nil {
					if len(errs) > 0 && IsDebugEnabled() {
						Log.Errorf("Transfer for %s aborted with error: %s", src.FullPath, errs[0].Error())
					} else {
						Log.Debugf("Transfer for %s terminated", src.FullPath)
					}
				}
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
					contextualizedErr := fmt.Errorf("could not download '%s' to '%s': %s", src.FullPath, c.FullPath, e.Error())
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
		return fmt.Errorf("could not stat file at %s, cause: %s", src.FullPath, e.Error())
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
		// also initialise the error chan in no-progress mode
		var done chan struct{}
		handleError := func(e error) {
			// TODO we still handle the errors at another level
			// fmt.Println("Error:", e)
		}
		errChan, done = newErrorChan(handleError)
		defer close(done)

		content = file
	}

	bName := src.RelPath
	fullPath := c.join(c.FullPath, bName)

	var upErr error
	if stats.Size() <= UploadSwitchMultipart*(1024*1024) {
		if _, e = c.sdkClient.PutFile(ctx, fullPath, content, false); e != nil {
			upErr = fmt.Errorf("could not upload single part file %s: %s", fullPath, e.Error())
		}
		if bar == nil {
			Log.Debugf("\t%s: uploaded\n", fullPath)
		}
	} else {
		upErr = c.sdkClient.s3Upload(ctx, fullPath, content, stats.Size(), IsDebugEnabled(), errChan)
	}
	// fmt.Println("... About to return from upload, error:", upErr)
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

	localTargetPath := c.join(c.FullPath, targetName)
	writer, e := os.OpenFile(localTargetPath, os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer func(writer *os.File) {
		err := writer.Close()
		if err != nil && bar == nil { // Only in no progress mode.
			Log.Warnf(
				"could not close writer after creating %s: %s\n",
				localTargetPath,
				err.Error(),
			)
		}
	}(writer)
	written, e := io.Copy(writer, content)
	if e != nil {
		return e
	} else if written != int64(length) {
		Log.Warnf("written length (%d) does not fit with source file length (%d) for %s\n",
			written, int64(length), src.FullPath)
	}
	return nil
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
func (c *CrawlNode) createRemoteFolders(target *CrawlNode, toCreateDirs []*CrawlNode, pool *BarsPool) error {

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
			if IsDebugEnabled() {
				return errors.Errorf("could not create folders at %s, cause: %s", target.FullPath, err.Error())
			}
			return errors.Errorf("could not prepare tree at %s", target.FullPath)
		}
		// TODO:  Stat all folders to make sure they are indexed ?
		if pool != nil {
			for range subArray {
				pool.Done()
			}
		} else {
			Log.Infof("... Created %d folders on remote server", end)
		}
	}
	return nil
}

/* Local Helpers */

// dryRunCreate simply list the folders that should be created.
func (c *CrawlNode) dryRunCreate(toCreateDirs []*CrawlNode) {
	// TODO handle parent folder
	for _, d := range toCreateDirs {
		newFolder := c.join(c.FullPath, d.RelPath)
		Log.Infof("MkDir: %s", newFolder)
	}
}

// dryRunDelete simply list the folders that should be deleted.
func (c *CrawlNode) dryRunDelete(toDeleteItems []*CrawlNode) {
	for _, d := range toDeleteItems {
		toDelete := c.join(c.FullPath, d.RelPath)
		Log.Infof("Delete: %s", toDelete)
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
