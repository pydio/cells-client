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

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/models"

	. "github.com/pydio/cells-client/rest"
)

var scpFileExample = `
Prefix remote paths with cells:// to differentiate local from remote. Currently, copy can only be performed with both different ends.
For example:

1/ Uploading a file to server

$ ` + os.Args[0] + ` scp ./README.md cells://common-files/
Copying ./README.md to cells://common-files/
 ## Waiting for file to be indexed...
 ## File correctly indexed

2/ Download a file from server

$ ` + os.Args[0] + ` scp cells://personal-files/IMG_9723.JPG ./
Copying cells://personal-files/IMG_9723.JPG to /home/pydio/downloads/
`

const (
	prefixA = "cells://"
	prefixB = "cells//"
)

var (
	currentPrefix          string
	sourcePath, targetPath string
)

var scpFiles = &cobra.Command{
	Use:     "scp",
	Short:   `Copy files from/to Cells`,
	Long:    `Copy files between local server and remote Cells server`,
	Example: scpFileExample,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 2 {
			cmd.Help()
			log.Fatal(fmt.Errorf("please provide at least a source and a destination target"))
		}

		from := args[0]
		to := args[1]

		if strings.HasPrefix(from, prefixA) || strings.HasPrefix(to, prefixA) {
			currentPrefix = prefixA
		} else if strings.HasPrefix(from, prefixB) || strings.HasPrefix(to, prefixB) {
			currentPrefix = prefixB
		}

		fmt.Printf("Copying %s to %s\n", from, to)

		if strings.HasPrefix(from, prefixA) || strings.HasPrefix(from, prefixB) {
			// Download
			fromPath := strings.TrimPrefix(from, currentPrefix)
			_, remote, e := targetToFullPath(to, from)
			if e != nil {
				log.Fatal(e)
			}
			if remote {
				log.Fatal(fmt.Errorf("source and target are both remote, copy remote to local or the opposite"))
			}

			//TODO toPath already creates the targetedLocation -- just append the node

			sourcePath = fromPath
			targetPath = to
			// dl rec
			if err := downloadRecursive(fromPath, to); err != nil {
				log.Fatal(err)
			}

			log.Println("Download finished !")
			//reader, length, e := rest.GetFile(fromPath)
			//if e != nil {
			//	log.Fatal(e)
			//}
			//bar := uiprogress.AddBar(length).PrependCompleted()
			//wrapper := &PgReader{
			//	Reader: reader,
			//	bar:    bar,
			//	total:  length,
			//}
			//writer, e := os.OpenFile(toPath, os.O_CREATE|os.O_WRONLY, 0755)
			//if e != nil {
			//	log.Fatal(e)
			//}
			//defer writer.Close()
			//uiprogress.Start()
			//_, e = io.Copy(writer, wrapper)
			//if e != nil {
			//	log.Fatal(e)
			//}
			//// Wait that progress bar finish rendering
			//<-time.After(100 * time.Millisecond)
			//uiprogress.Stop()

		} else {
			// Upload
			_, remote, e := targetToFullPath(to, from)
			if e != nil {
				log.Fatal(e)
			}
			if !remote {
				log.Fatal(fmt.Errorf("source and target are both local, copy remote to local or the opposite"))
			}

			// upload
			to = strings.TrimPrefix(to, currentPrefix)
			sourcePath = from
			targetPath = to
			//TODO make sure to index just once, because right now it indexes after each upload which makes the operation slow
			if err := UploadRecursive(from, to); err != nil {
				log.Fatal(err)
			}

			//var length int64
			//if s, e := os.Stat(from); e != nil || s.IsDir() {
			//	log.Fatal(fmt.Errorf("local source is not a valid file"))
			//} else {
			//	length = s.Size()
			//}
			//reader, e := os.Open(from)
			//bar := uiprogress.AddBar(int(length)).PrependCompleted()
			//wrapper := &PgReader{
			//	Reader: reader,
			//	Seeker: reader,
			//	bar:    bar,
			//	total:  int(length),
			//	double: true,
			//}
			//if e != nil {
			//	log.Fatal(e)
			//}
			//uiprogress.Start()
			//_, e = rest.PutFile(toPath, wrapper, false)
			//if e != nil {
			//	log.Fatal(e)
			//}
			//<-time.After(500 * time.Millisecond)
			//uiprogress.Stop()
			//// Now stat Node to make sure it is indexed
			//e = rest.RetryCallback(func() error {
			//	fmt.Println(" ## Waiting for file to be indexed...")
			//	_, ok := rest.StatNode(toPath)
			//	if !ok {
			//		return fmt.Errorf("cannot stat node just after PutFile operation")
			//	}
			//	return nil
			//
			//}, 3, 3*time.Second)
			//if e != nil {
			//	log.Fatal("File does not seem to be indexed!")
			//}
			//fmt.Println(" ## File correctly indexed")
		}
	},
}

func init() {
	RootCmd.AddCommand(scpFiles)
}

func targetToFullPath(to, from string) (string, bool, error) {
	var toPath string
	var isDir bool
	var isRemote bool
	var e error
	if strings.HasPrefix(to, currentPrefix) {
		// This is remote
		isRemote = true
		toPath = strings.TrimPrefix(to, currentPrefix)
		target, ok := StatNode(toPath)
		if !ok {
			// Does not exists => will be created
			return toPath, isRemote, nil
		}
		isDir = target.Type == models.TreeNodeTypeCOLLECTION
	} else {
		// This is local
		toPath, e = filepath.Abs(to)
		if e != nil {
			return "", false, e
		}
		s, e := os.Stat(toPath)
		if e != nil {
			return "", false, e
		}
		isDir = s.IsDir()
	}

	if isDir {
		toPath = path.Join(toPath, path.Base(from))
	}
	return toPath, isRemote, nil
}

type PgReader struct {
	io.Reader
	io.Seeker
	bar   *uiprogress.Bar
	total int
	read  int

	double bool
	first  bool
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

func downloadRecursive(downloadFrom, downloadTo string) error {
	nodes, err := walkRemote(downloadFrom, downloadTo, true)
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
		go func(nodePath string) {
			defer func() {
				<-buf
				wg.Done()
				//for bar.Incr() {
				//	<-time.After(500 * time.Millisecond)
				//}
			}()

			reader, _, e := GetFile(nodePath)
			if e != nil {
				log.Println("could not GetFile ", e)
			}
			//bar = uiprogress.NewBar(length)
			//wrapper := &PgReader{
			//	Reader: reader,
			//	bar:    bar,
			//	total:  length,
			//}
			downloadToLocation := TargetLocation(downloadTo, downloadFrom, nodePath)
			writer, e := os.OpenFile(downloadToLocation, os.O_CREATE|os.O_WRONLY, 0755)
			if e != nil {
				log.Println("could not OpenFile ", e)
			}
			defer writer.Close()
			_, e = io.Copy(writer, reader)
			if e != nil {
				log.Println("could not Copy", e)
			}

		}(n.Path)
	}
	wg.Wait()
	//uiprogress.Stop()
	return nil
}

func UploadRecursive(from, to string) error {
	wg := &sync.WaitGroup{}
	buf := make(chan struct{}, 3)
	//TODO make sure to add error checks
	ll, err := walkLocal(from, to, true)
	uiprogress.Start()
	for _, l := range ll {
		buf <- struct{}{}
		wg.Add(1)

		go func(uploadFrom, uploadTo string) {
			defer func() {
				<-buf
				wg.Done()
			}()
			Upload(uploadFrom, uploadTo)
		}(l.localNodePath, l.remoteNodePath)
		wg.Wait()

	}
	if err != nil {
		return err
	}
	uiprogress.Stop()
	return nil
}

// upload take a local resource and puts it in the remote location
func Upload(source string, target string) {
	reader, e := os.Open(source)
	if e != nil {
		return
	}
	stats, _ := reader.Stat()
	bar := uiprogress.AddBar(int(stats.Size())).PrependElapsed()
	//bar.PrependFunc(func(b *uiprogress.Bar) string {
	//	return "file: " + path.Base(target)
	//})
	wrapper := &PgReader{
		Reader: reader,
		Seeker: reader,
		bar:    bar,
		total:  int(stats.Size()),
		double: true,
	}

	_, e = PutFile(target, wrapper, false)
	if e != nil {
		return
	}
	//FIXME disabled (will be re enabled later)
	//// Now stat Node target make sure it is indexed
	//e = RetryCallback(func() error {
	//	//fmt.Println(" ## Waiting for file target be indexed...")
	//	_, ok := StatNode(target)
	//	if !ok {
	//		return fmt.Errorf("cannot stat node just after PutFile operation")
	//	}
	//	return nil
	//}, 3, 3*time.Second)
	//if e != nil {
	//	log.Fatal("File does not seem target be indexed!")
	//}
	for bar.Incr() {
		<-time.After(500 * time.Millisecond)
	}
	fmt.Println(" ## File correctly indexed")
}

// TODO keep that?
type localTree struct {
	localNodePath  string
	remoteNodePath string
	info           os.FileInfo
}

// walkLocal walks the localTree and returns a struct with the localNode-path and the remoteNode-path to ease the Upload
func walkLocal(fromLocal, toRemote string, createRemote bool) ([]localTree, error) {

	var l localTree
	var ll []localTree
	var remoteDirPath []string

	err := filepath.Walk(fromLocal, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}
		if info.IsDir() {
			//TODO modify the dirpath toRemote have <target-path>/<source-folder>/...
			remoteUploadPath := TargetLocation(toRemote, fromLocal, path)
			remoteDirPath = append(remoteDirPath, remoteUploadPath)
		} else {
			l.remoteNodePath = TargetLocation(toRemote, fromLocal, path)
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
		err = TreeCreateNodes(nodes)
		if err != nil {
			return nil, err
		}
		//FIXME make sure toRemote index after the creation - maybe add the index inside the tree function
		//TODO stat node or something toRemote make sure that the nodes are there
		//rest.RunJob("datasource-resync", "{\"dsName\":\"pydiods1\"}")
	}
	return ll, nil
}

// walkRemote lists all the nodes and if createLocal is set will create the tree on local system
func walkRemote(fromRemote, toLocal string, createLocal ...bool) (nodes []*models.TreeNode, err error) {
	var localTarget string
	// lists nodes fromRemote server
	nn, e := GetBulkMetaNode(fromRemote)
	if e != nil {
		err = e
		return
	}

	for _, n := range nn {
		nodes = append(nodes, n)
		if n.Type == models.TreeNodeTypeCOLLECTION {
			if len(createLocal) > 0 && createLocal[0] {
				//Trim the star toLocal avoid errors during local path construction

				fromRemote = strings.Trim(fromRemote, "*")
				localTarget = TargetLocation(targetPath, sourcePath, n.Path)

				if err = os.MkdirAll(localTarget, 0755); err != nil {
					return
				}
			}
			if children, e := walkRemote(path.Join(n.Path, "*"), toLocal, createLocal...); e != nil {
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
func TargetLocation(target, source, nodeSource string) string {

	source = strings.Trim(source, "/")
	nodeSource = strings.Trim(nodeSource, "/")
	serverBase := path.Base(source)
	relativePath := strings.TrimPrefix(nodeSource, source)

	return filepath.Join(target, serverBase, relativePath)
}
