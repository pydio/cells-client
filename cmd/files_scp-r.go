package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gosuri/uiprogress"
	"github.com/pydio/cells-sdk-go/models"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
)

var (
	targetPath, sourcePath string
)

var scprCmd = &cobra.Command{
	Use:   "scpr",
	Short: "scp recursive test",
	Run: func(cmd *cobra.Command, args []string) {

		downloadFrom := "personal-files/formula-one"
		downloadTo := "/Users/jay/Downloads/lulu"

		//// Load all tree and create folders locally
		//nodes, err := walkRemote(downloadFrom, downloadTo, true)
		//if err != nil {
		//	log.Fatalln("", err)
		//}
		//if len(nodes) < 0 {
		//
		//}
		//download(nodes, downloadTo, uiprogress.Bar{}, 0)

		//source := "/Users/jay/Downloads/toto"
		////If targeted folder does not exist
		//target := "common-files/"
		////TODO add a flag if recursive to run recursive function
		//err := uploadRecursive(source, target)
		//if err != nil {
		//	log.Fatalln("", err)
		//}
		err := downloadRecursive(downloadFrom, downloadTo)
		if err != nil {
			log.Fatal(err)
		}

	},
}

func init() {
	RootCmd.AddCommand(scprCmd)
}

// TODO look at targetPath, sourcePath
//TODO split download / download recursive
func download(nodes []*models.TreeNode, to string, pgBar uiprogress.Bar, totalPg int64) error {
	// Download all files
	wg := &sync.WaitGroup{}
	buf := make(chan struct{}, 3)
	for _, n := range nodes {
		if n.Type == models.TreeNodeTypeCOLLECTION {
			continue
		}
		buf <- struct{}{}
		wg.Add(1)
		uiprogress.Start()
		go func(remotePath string) {
			defer func() {
				<-buf
				wg.Done()
			}()
			downloadPath := targetLocation(targetPath, sourcePath, remotePath)
			reader, length, e := rest.GetFile(remotePath)
			if e != nil {
				log.Println("could not GetFile ", e)
			}

			bar := uiprogress.AddBar(length).PrependElapsed().AppendCompleted()
			bar.PrependFunc(func(b *uiprogress.Bar) string {
				return "file :"
			})

			wrapper := &PgReader{
				Reader: reader,
				bar:    bar,
				total:  length,
			}

			writer, e := os.OpenFile(downloadPath, os.O_CREATE|os.O_WRONLY, 0755)
			if e != nil {
				log.Println("could not OpenFile ", e)
			}
			defer writer.Close()

			_, e = io.Copy(writer, wrapper)
			if e != nil {
				log.Println("could not Copy", e)
			}
			for bar.Incr() {
				<-time.After(500 * time.Millisecond)
			}
		}(n.Path)
	}
	wg.Wait()
	return nil
}

func downloadRecursive(from, to string) error {
	nodes, err := walkRemote(from, to, true)
	if err != nil {
		log.Fatalln("", err)
	}
	wg := &sync.WaitGroup{}
	buf := make(chan struct{}, 3)
	for _, n := range nodes {
		if n.Type == models.TreeNodeTypeCOLLECTION {
			continue
		}
		buf <- struct{}{}
		wg.Add(1)
		uiprogress.Start()
		go func(remotePath string) {
			defer func() {
				<-buf
				wg.Done()
			}()
			downloadPath := targetLocation(from, to, remotePath)
			reader, length, e := rest.GetFile(remotePath)
			if e != nil {
				log.Println("could not GetFile ", e)
			}

			bar := uiprogress.AddBar(length).PrependElapsed().AppendCompleted()
			bar.PrependFunc(func(b *uiprogress.Bar) string {
				return "file :"
			})

			wrapper := &PgReader{
				Reader: reader,
				bar:    bar,
				total:  length,
			}

			writer, e := os.OpenFile(downloadPath, os.O_CREATE|os.O_WRONLY, 0755)
			if e != nil {
				log.Println("could not OpenFile ", e)
			}
			defer writer.Close()

			_, e = io.Copy(writer, wrapper)
			if e != nil {
				log.Println("could not Copy", e)
			}
			for bar.Incr() {
				<-time.After(500 * time.Millisecond)
			}
		}(n.Path)
	}
	wg.Wait()
	return nil

}

func uploadRecursive(from, to string) error {
	wg := &sync.WaitGroup{}
	buf := make(chan struct{}, 3)
	//TODO make sure to add error checks
	ll, err := walkLocal(from, to, true)
	for _, l := range ll {
		buf <- struct{}{}
		wg.Add(1)
		uiprogress.Start()
		go func(uploadFrom, uploadTo string) {
			defer func() {
				<-buf
				wg.Done()
			}()
			upload(from, to)
		}(l.localNodePath, l.remoteNodePath)
		wg.Wait()
		//TODO add concurrent upload as seen with the downloads
		upload(l.localNodePath, l.remoteNodePath)
	}
	if err != nil {
		return err
	}
	return nil
}

// upload take a local resource and puts it in the remote location
func upload(from string, to string) {
	reader, e := os.Open(from)
	if e != nil {
		return
	}
	stats, _ := reader.Stat()
	bar := uiprogress.AddBar(int(stats.Size())).PrependElapsed().AppendCompleted()
	wrapper := &PgReader{
		Reader: reader,
		Seeker: reader,
		bar:    bar,
		total:  int(stats.Size()),
		double: true,
	}

	_, e = rest.PutFile(to, wrapper, false)
	if e != nil {
		return
	}
	// Now stat Node to make sure it is indexed
	e = rest.RetryCallback(func() error {
		//fmt.Println(" ## Waiting for file to be indexed...")
		_, ok := rest.StatNode(to)
		if !ok {
			return fmt.Errorf("cannot stat node just after PutFile operation")
		}
		return nil
	}, 3, 3*time.Second)
	if e != nil {
		log.Fatal("File does not seem to be indexed!")
	}
	for bar.Incr() {
		<-time.After(500 * time.Millisecond)
	}
	//fmt.Println(" ## File correctly indexed")
}

// TODO get rid of this confusing struct or not
type localTree struct {
	localNodePath  string
	remoteNodePath string
	info           os.FileInfo
}

// walkLocal walks the localtree and returns a struct with the localNode-path and the remoteNode-path to ease the upload
func walkLocal(from, to string, createRemote bool) ([]localTree, error) {

	var l localTree
	var ll []localTree
	var remoteDirPath []string

	err := filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}
		if info.IsDir() {
			//TODO modify the dirpath to have <target-path>/<source-folder>/...
			remoteUploadPath := targetLocation(to, from, path)
			remoteDirPath = append(remoteDirPath, remoteUploadPath)
		} else {
			l.remoteNodePath = targetLocation(to, from, path)
			l.localNodePath = path
			l.info = info
			ll = append(ll, l)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if createRemote {
		var nodes []*models.TreeNode
		for _, dirP := range remoteDirPath {
			nodes = append(nodes, &models.TreeNode{Path: dirP, Type: models.TreeNodeTypeCOLLECTION})
		}
		err = rest.TreeCreateNodes(nodes)
		if err != nil {
			return nil, err
		}
		//FIXME make sure to index after the creation - maybe add the index inside the tree function
		//TODO stat node or something to make sure that the nodes are there
		//rest.RunJob("datasource-resync", "{\"dsName\":\"pydiods1\"}")
	}
	return ll, nil
}

// walkRemote lists all the nodes and if createLocal is set will create the tree on local system
func walkRemote(from, to string, createLocal ...bool) (nodes []*models.TreeNode, err error) {
	var localTarget string
	// lists nodes from server
	nn, e := rest.GetBulkMetaNode(from)
	if e != nil {
		err = e
		return
	}

	for _, n := range nn {
		nodes = append(nodes, n)
		if n.Type == models.TreeNodeTypeCOLLECTION {
			if len(createLocal) > 0 && createLocal[0] {
				//Trim the star to avoid errors during local path construction
				from = strings.Trim(from, "*")
				localTarget = targetLocation(targetPath, sourcePath, n.Path)
				if err = os.MkdirAll(localTarget, 0755); err != nil {
					return
				}
			}
			if children, e := walkRemote(path.Join(n.Path, "*"), to, createLocal...); e != nil {
				err = e
				return
			} else {
				nodes = append(nodes, children...)
			}
		}
	}
	return
}

// tbd: download -> /Users/j/downloads/ + /personal-files/folder + /personal-files/folder/meteo.jpg = /Users/j/downloads/folder/meteo.jpg
func targetLocation(localPath, remotePath, nodePath string) string {

	remotePath = strings.Trim(remotePath, "/")
	nodePath = strings.Trim(nodePath, "/")
	serverBase := path.Base(remotePath)
	relativePath := strings.TrimPrefix(nodePath, remotePath)

	return filepath.Join(localPath, serverBase, relativePath)
}
