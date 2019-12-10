package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	. "github.com/pydio/cells-client/rest"
)

var scpFileExample = `
1/ Uploading a file to the server:
  $ ` + os.Args[0] + ` scp ./README.md cells://common-files/
  Copying ./README.md to cells://common-files/
  ## Waiting for file to be indexed...
  ## File correctly indexed

2/ Download a file from server:
  $ ` + os.Args[0] + ` scp cells://personal-files/funnyCat.jpg ./
  Copying cells://personal-files/funnyCat.jpg to /home/pydio/downloads/
`

const (
	prefixA = "cells://"
	prefixB = "cells//"
)

var (
	scpCurrentPrefix   string
	scpCreateAncestors bool
)

var scpFiles = &cobra.Command{
	Use:   "scp",
	Short: `Copy files from/to Cells`,
	Long: `
Copy files from your local machine to your Pydio Cells server instance (and vice versa).

To differentiate local from remote, prefix remote paths with 'cells://'. 
For the time being, copy can only be performed with both different ends.
`,
	Example: scpFileExample,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 2 {
			cmd.Help()
			log.Fatal("Please provide at least a source *and* a destination target.")
		}

		from := args[0]
		to := args[1]

		if strings.HasPrefix(from, prefixA) || strings.HasPrefix(to, prefixA) {
			scpCurrentPrefix = prefixA
		} else if strings.HasPrefix(from, prefixB) || strings.HasPrefix(to, prefixB) {
			scpCurrentPrefix = prefixB
		} else {
			// No prefix found
			log.Fatal("Source and target are both local, copy remote to local or the opposite.")
		}

		DryRun = false // Debug option
		isSrcLocal := true
		var crawlerPath, targetPath string

		// Prepare paths
		if strings.HasPrefix(from, scpCurrentPrefix) {
			// Download
			fromPath := strings.TrimPrefix(from, scpCurrentPrefix)
			localTarget, isRemote, e := targetToFullPath(to, from)
			if e != nil {
				log.Fatal(e)
			}
			if isRemote {
				log.Fatal("Source and target are both remote, copy remote to local or the opposite.")
			}
			crawlerPath = fromPath
			isSrcLocal = false
			targetPath = localTarget
			fmt.Printf("Downloading %s to %s\n", from, to)

		} else {
			// Upload
			// Called to check target path existence
			if _, _, e := targetToFullPath(to, from); e != nil {
				log.Fatal(e)
			}
			crawlerPath = from
			targetPath = strings.TrimPrefix(to, scpCurrentPrefix)
			fmt.Printf("Uploading %s to %s\n", from, to)
		}

		// FLAG RENAME ON THE FLY
		// scp /Users/charles/tmp/toto  => cells//common-files/tutu

		crawler, e := NewCrawler(crawlerPath, isSrcLocal)
		if e != nil {
			log.Fatal(e)
		}
		nn, e := crawler.Walk()
		if e != nil {
			log.Fatal(e)
		}
		targetNode := NewTarget(targetPath, crawler)

		// FLAG RENAME ON THE FLY
		// [CrawlNode{FullPath:/Users/charles/tmp/toto, RelPath:""}]
		// [CrawlNode{FullPath:/Users/charles/tmp/toto/A.txt, RelPath:"A.txt"}]
		// [CrawlNode{FullPath:/Users/charles/tmp/toto/B.txt, RelPath:"B.txt"}]
		// ==> patchRelativePath(nn)
		// [CrawlNode{FullPath:/Users/charles/tmp/toto, RelPath:"tutu"}]
		// [CrawlNode{FullPath:/Users/charles/tmp/toto/A.txt, RelPath:"tutu/A.txt"}]
		// [CrawlNode{FullPath:/Users/charles/tmp/toto/B.txt, RelPath:"tutu/A.txt"}]

		// [CrawlNode{FullPath:/Users/charles/tmp/toto.txt, RelPath:"toto.txt"}]
		// ==> patchRelativePath(nn)
		// [CrawlNode{FullPath:/Users/charles/tmp/toto.txt, RelPath:"tutu.txt"}]

		pool := NewBarsPool(len(nn) > 1, len(nn))
		pool.Start()

		// CREATE FOLDERS
		e = targetNode.MkdirAll(nn, pool)
		if e != nil {
			log.Fatal(e)
		}

		// UPLOAD / DOWNLOAD FILES
		errs := targetNode.CopyAll(nn, pool)
		//pool.Stop()
		if len(errs) > 0 {
			log.Fatal(errs)
		}
	},
}

func init() {

	flags := scpFiles.PersistentFlags()
	flags.BoolVarP(&scpCreateAncestors, "parents", "p", false, "Force creation of non-existing ancestors on remote Cells server")
	RootCmd.AddCommand(scpFiles)
}

func targetToFullPath(to, from string) (string, bool, error) {
	var toPath string
	//var isDir bool
	var isRemote bool
	var e error
	if strings.HasPrefix(to, scpCurrentPrefix) {
		// This is remote
		isRemote = true
		toPath = strings.TrimPrefix(to, scpCurrentPrefix)
		_, ok := StatNode(toPath)
		if !ok {
			if scpCreateAncestors {
				// Does not exists => will be created
				return toPath, true, nil
			} else {
				// No force creation flag && target not exist=> error
				return toPath, true, fmt.Errorf("Target folder %s does not exits on remote server. Consider using the '-p' flag to force creation of non existing ancestors.", toPath)
			}
		}
	} else {
		// This is local
		toPath, e = filepath.Abs(to)
		if e != nil {
			return "", false, e
		}
		_, e := os.Stat(toPath)
		if e != nil {
			return "", false, e
		}
	}

	return toPath, isRemote, nil
}
