package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	. "github.com/pydio/cells-client/rest"
)

var scpFileExample = `
1/ Uploading a file to the server:
  $ ` + os.Args[0] + ` scp ./README.md cells://common-files/
  Copying ./README.md to cells://common-files/
  Waiting for file to be indexed...
  File correctly indexed

2/ Download a file from server:
  $ ` + os.Args[0] + ` scp cells://personal-files/funnyCat.jpg ./
  Copying cells://personal-files/funnyCat.jpg to /home/pydio/downloads/

3/ Download a file changing its name - remember: this will fail if a 'cat2.jpg' file already exists: 
  $ ` + os.Args[0] + ` scp cells://personal-files/funnyCat.jpg ./cat2.jpg
  Copying cells://personal-files/funnyCat.jpg to /home/pydio/downloads/
`

const (
	prefixA = "cells://"
	prefixB = "cells//"
)

var (
	scpCurrentPrefix string
	scpQuiet         bool
)

var scpFiles = &cobra.Command{
	Use:   "scp",
	Short: `Copy files from/to Cells`,
	Long: `
Copy files from your local machine to your Pydio Cells server instance (and vice versa).

To differentiate local from remote, prefix remote paths with 'cells://' or with 'cells//' (without the column) if you have installed the completion and intend to use it.
For the time being, copy can only be performed with both different ends.

Note that you can rename the file or base folder that you upload/download if:  

- last part of the target path is a new name that *does not exists*,  
- parent path exists and is a folder a target location.
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

		// Prepare paths
		DryRun = false // Debug option
		isSrcLocal := true
		var crawlerPath, targetPath string
		var rename bool
		var err error
		if strings.HasPrefix(from, scpCurrentPrefix) {
			// Download
			isSrcLocal = false
			var isRemote bool
			crawlerPath = strings.TrimPrefix(from, scpCurrentPrefix)
			targetPath, isRemote, rename, err = targetToFullPath(from, to)
			if err != nil {
				log.Fatal(err)
			}
			if isRemote {
				log.Fatal("Source and target are both remote, copy remote to local or the opposite.")
			}
			fmt.Printf("Downloading %s to %s\n", from, to)
		} else {
			// Upload
			targetPath = strings.TrimPrefix(to, scpCurrentPrefix)
			// Check target path existence and handle rename corner cases
			if _, _, rename, err = targetToFullPath(from, to); err != nil {
				log.Fatal(err)
			}
			crawlerPath = from
			fmt.Printf("Uploading %s to %s\n", from, to)
		}

		crawler, e := NewCrawler(crawlerPath, isSrcLocal)
		if e != nil {
			log.Fatal(e)
		}
		nn, e := crawler.Walk()
		if e != nil {
			log.Fatal(e)
		}

		targetNode := NewTarget(targetPath, crawler, rename)

		refreshInterval := time.Millisecond * 10 // this is the default
		if scpQuiet {
			refreshInterval = time.Millisecond * 1000
		}
		pool := NewBarsPool(len(nn) > 1, len(nn), refreshInterval)
		pool.Start()

		// CREATE FOLDERS
		e = targetNode.MkdirAll(nn, pool)
		if e != nil {
			// Force stop of the pool that stays blocked otherwise:
			// It is launched *before* the MkdirAll but only managed during the CopyAll phase.
			pool.Stop()
			log.Fatal(e)
		}

		// UPLOAD / DOWNLOAD FILES
		errs := targetNode.CopyAll(nn, pool)
		//pool.Stop()
		if len(errs) > 0 {
			log.Fatal(errs)
		}
		fmt.Println("") // Add a line to reduce glitches in the terminal
	},
}

func targetToFullPath(from, to string) (string, bool, bool, error) {
	var toPath string
	//var isDir bool
	var isRemote bool
	var e error
	if strings.HasPrefix(to, scpCurrentPrefix) {
		// This is remote: UPLOAD
		isRemote = true
		toPath = strings.TrimPrefix(to, scpCurrentPrefix)
		_, ok := StatNode(toPath)
		if !ok {

			parPath, _ := path.Split(toPath)
			if parPath == "" {
				// unexisting workspace
				return toPath, true, false, fmt.Errorf("Target path %s does not exist on remote server, please double check and correct.", toPath)
			}

			// Check if parent exists. In such case, we rename the file or root folder that has been passed as local source
			// Typically, `cec scp README.txt cells//common-files/readMe.md` or `cec scp local-folder cells//common-files/remote-folder`
			if _, ok2 := StatNode(parPath); !ok2 {
				// Target parent folder does not exist, we do not create it
				return toPath, true, false, fmt.Errorf("Target parent folder %s does not exist on remote server. ", parPath)
			} else {
				// Parent folder exists on remote, we rename src file or folder
				return toPath, true, true, nil
			}
		}
	} else {
		// This is local: DOWNLOAD
		toPath, e = filepath.Abs(to)
		if e != nil {
			return "", false, false, e
		}
		_, e := os.Stat(toPath)
		if e != nil {

			parPath := filepath.Dir(toPath)
			if parPath == "." {
				// this should never happen
				return toPath, true, false, fmt.Errorf("Target path %s does not exist on local server, please double check and correct.", toPath)
			}

			// Check if parent exists. In such case, we rename the file or root folder that has been passed as remote source
			if ln, err2 := os.Stat(parPath); err2 != nil {
				// Target parent folder does not exist on local machine, we do not create it
				return "", true, false, fmt.Errorf("Target parent folder %s does not exist on local server. ", parPath)
			} else if !ln.IsDir() {
				// Local parent is not a folder
				return "", true, false, fmt.Errorf("Target parent %s is not a folder, could not download to it.", parPath)
			} else {
				// Parent folder exists on local, we rename src file or folder
				return toPath, false, true, nil
			}
		}
	}

	return toPath, isRemote, false, nil
}

func init() {

	flags := scpFiles.PersistentFlags()
	flags.BoolVarP(&scpQuiet, "quiet", "q", false, "Reduce the amount of logs")
	RootCmd.AddCommand(scpFiles)
}
