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

	. "github.com/pydio/cells-client/rest"
)

const (
	scprCmdExample = `
`
)

var (
	sourcePath, targetPath string
	recursive              bool
)

var scprCmd = &cobra.Command{
	Use:   "scpr",
	Short: "scp recursive test",
	Run: func(cmd *cobra.Command, args []string) {

		//TODO parse args if arg[Ø] starts with cells:// = download from remote -> to target
		//example cec scp cells://common-files/formula-one
		// if arg[1]

		sourcePath = "personal-files/Top-left_triangle_rasterization_rule.gif"
		targetPath = "/Users/jay/Downloads/lulu/"

		if _, status := StatNode(sourcePath); status != true {
			log.Fatalf("Cannot download this node, it does not exist, node : [%s]\n", sourcePath)
		}

		//// Load all tree and create folders locally
		//nodes, err := walkRemote(sourcePath, downloadTo, true)
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
		err := downloadRecursive(sourcePath, targetPath)
		if err != nil {
			log.Fatal(err)
		}

	},
}

func init() {
	RootCmd.AddCommand(scprCmd)

	scprCmd.PersistentFlags().BoolVarP(&recursive, " recursive", "r", false, "Apply recursion to the operation (behaviour similar to the -r option of the linux commands) ")
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
			downloadPath := TargetLocation(targetPath, sourcePath, remotePath)
			reader, length, e := GetFile(remotePath)
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
		uiprogress.Start()

		go func(nodePath string) {
			defer func() {
				<-buf
				wg.Done()
			}()

			reader, length, e := GetFile(nodePath)
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
			downloadToLocation := TargetLocation(downloadTo, downloadFrom, nodePath)
			writer, e := os.OpenFile(downloadToLocation, os.O_CREATE|os.O_WRONLY, 0755)
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
func upload(source string, target string) {
	reader, e := os.Open(source)
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

	_, e = PutFile(target, wrapper, false)
	if e != nil {
		return
	}
	// Now stat Node target make sure it is indexed
	e = RetryCallback(func() error {
		//fmt.Println(" ## Waiting for file target be indexed...")
		_, ok := StatNode(target)
		if !ok {
			return fmt.Errorf("cannot stat node just after PutFile operation")
		}
		return nil
	}, 3, 3*time.Second)
	if e != nil {
		log.Fatal("File does not seem target be indexed!")
	}
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

// walkLocal walks the localtree and returns a struct with the localNode-path and the remoteNode-path to ease the upload
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
