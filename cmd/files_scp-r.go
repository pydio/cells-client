package cmd

import (
	"io"
	"log"
	"os"
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
	//sourcePath, targetPath string
	recursive bool
)

var scprCmd = &cobra.Command{
	Use:   "scpr",
	Short: "scp recursive test",
	Run: func(cmd *cobra.Command, args []string) {

		//TODO parse args

		//if arg[Ø] starts with cells:// = download from remote -> to target
		//example cec scp cells://common-files/formula-one

		// if arg[1] starts with cells:// it means upload to
		// example cec scp /Users/j/Downloads/ cells://personal-files/formula-one

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
