package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/common"
	"github.com/pydio/cells-client/v4/rest"
)

const (
	prefixA = "cells://"
	prefixB = "cells//"
)

var (
	scpCurrentPrefix string
	scpQuiet         bool
	scpVerbose       bool
	scpVeryVerbose   bool
)

var scpFiles = &cobra.Command{
	Use:   "scp",
	Short: `Copy files from/to Cells`,
	Long: `
DESCRIPTION

  Copy files from the client machine to your Pydio Cells server instance (and vice versa).

  To differentiate local from remote, prefix remote paths with 'cells://' or with 'cells//' (without the column) if you have installed the completion and intend to use it.
  For the time being, copy can only be performed from the client machine to the server or the otherway round:
  it is not yet possible to copy from one Cells instance to another.

SYNTAX

  Note that you can rename the file or base folder that you upload/download if:  
   - last part of the target path is a new name that *does not exist*,  
   - parent path exists and is a folder at target location.

EXAMPLES

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
`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		from := args[0]
		to := args[1]
		ctx := cmd.Context()

		if scpVeryVerbose {
			common.CurrentLogLevel = common.Trace
		} else if scpVerbose {
			common.CurrentLogLevel = common.Debug
		} else {
			common.CurrentLogLevel = common.Info
		}

		if strings.HasPrefix(from, prefixA) || strings.HasPrefix(to, prefixA) {
			scpCurrentPrefix = prefixA
		} else if strings.HasPrefix(from, prefixB) || strings.HasPrefix(to, prefixB) {
			scpCurrentPrefix = prefixB
		} else {
			// No prefix found
			log.Fatal("Source and target are both on the client machine, copy from server to client or the opposite.")
		}

		// Prepare paths
		rest.DryRun = false // Debug option
		isSrcLocal := true
		var crawlerPath, targetPath string
		var rename bool
		var err error
		if strings.HasPrefix(from, scpCurrentPrefix) {
			// Download
			isSrcLocal = false
			var isRemote bool
			crawlerPath = strings.TrimPrefix(from, scpCurrentPrefix)
			targetPath, isRemote, rename, err = targetToFullPath(ctx, from, to)
			if err != nil {
				log.Fatal(err)
			}
			if isRemote {
				log.Fatal("Source and target are both remote: you can only copy from client to remote Pydio Server or the opposite.")
			}
			fmt.Printf("Downloading %s to %s\n", from, to)
		} else {
			// Upload
			targetPath = strings.TrimPrefix(to, scpCurrentPrefix)
			// Check target path existence and handle rename corner cases
			if _, _, rename, err = targetToFullPath(ctx, from, to); err != nil {
				log.Fatal(err)
			}
			crawlerPath = from
			fmt.Printf("Uploading %s to %s\n", from, to)
		}

		crawler, e := rest.NewCrawler(ctx, crawlerPath, isSrcLocal)
		if e != nil {
			log.Fatal(e)
		}
		nn, e := crawler.Walk(cmd.Context())
		if e != nil {
			log.Fatal(e)
		}

		targetNode := rest.NewTarget(targetPath, crawler, rename)

		if common.CurrentLogLevel == common.Info {
			refreshInterval := time.Millisecond * 10 // this is the default
			if scpQuiet {
				refreshInterval = time.Millisecond * 3000
			}
			pool := rest.NewBarsPool(len(nn) > 1, len(nn), refreshInterval)
			pool.Start()

			// CREATE FOLDERS
			e = targetNode.MkdirAll(ctx, nn, pool)
			if e != nil {
				// Force stop of the pool that stays blocked otherwise:
				// It is launched *before* the MkdirAll but only managed during the CopyAll phase.
				pool.Stop()
				log.Fatal(e)
			}

			// UPLOAD / DOWNLOAD FILES
			errs := targetNode.CopyAll(ctx, nn, pool)
			if len(errs) > 0 {
				log.Fatal(errs)
			}
		} else { // Rather display logs than the progress bar
			fmt.Printf("... After walking the tree, found %d nodes to copy\n", len(nn))

			e = targetNode.MkdirAll(ctx, nn, nil)
			if e != nil {
				log.Fatal(e)
			}

			errs := targetNode.CopyAllVerbose(ctx, nn)
			if len(errs) > 0 {
				log.Fatal(errs)
			}
		}
		fmt.Println("") // Add a line to reduce glitches in the terminal
	},
}

func targetToFullPath(ctx context.Context, from, to string) (string, bool, bool, error) {
	var toPath string
	//var isDir bool
	var isRemote bool
	var e error
	if strings.HasPrefix(to, scpCurrentPrefix) {
		// This is remote: UPLOAD
		isRemote = true
		toPath = strings.TrimPrefix(to, scpCurrentPrefix)
		_, ok := rest.StatNode(ctx, toPath)
		if !ok {

			parPath, _ := path.Split(toPath)
			if parPath == "" {
				// unexisting workspace
				return toPath, true, false, fmt.Errorf("target path %s does not exist on remote server, please double check and correct. ", toPath)
			}

			// Check if parent exists. In such case, we rename the file or root folder that has been passed as local source
			// Typically, `cec scp README.txt cells//common-files/readMe.md` or `cec scp local-folder cells//common-files/remote-folder`
			if _, ok2 := rest.StatNode(ctx, parPath); !ok2 {
				// Target parent folder does not exist, we do not create it
				return toPath, true, false, fmt.Errorf("target parent folder %s does not exist on remote server. ", parPath)
			}

			// Parent folder exists on remote, we rename src file or folder
			return toPath, true, true, nil

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
				return toPath, true, false, fmt.Errorf("target path %s does not exist on client machine, please double check and correct. ", toPath)
			}

			// Check if parent exists. In such case, we rename the file or root folder that has been passed as remote source
			if ln, err2 := os.Stat(parPath); err2 != nil {
				// Target parent folder does not exist on client machine, we do not create it
				return "", true, false, fmt.Errorf("target parent folder %s does not exist in client machine. ", parPath)
			} else if !ln.IsDir() {
				// Local parent is not a folder
				return "", true, false, fmt.Errorf("target parent %s is not a folder, could not download to it. ", parPath)
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
	flags.BoolVarP(&scpVerbose, "verbose", "v", false, "Hide progress bar and rather display more log info during the transfers")
	flags.BoolVarP(&scpVeryVerbose, "very-verbose", "w", false, "Hide progress bar and rather print out a maximum of log info")
	flags.BoolVarP(&scpQuiet, "quiet", "q", false, "Reduce refresh frequency of the progress bars, e.g when runing cec in a bash script")
	flags.Int64Var(&common.UploadMaxPartsNumber, "max-parts-number", int64(5000), "Maximum number of parts, S3 supports 10000 but some storage require less parts.")
	flags.Int64Var(&common.UploadDefaultPartSize, "part-size", int64(50), "Default part size (MB), must always be a multiple of 10MB. It will be recalculated based on the max-parts-number value.")
	flags.IntVar(&common.UploadPartsConcurrency, "parts-concurrency", 3, "Number of concurrent part uploads.")
	flags.BoolVar(&common.UploadSkipMD5, "skip-md5", false, "Do not compute md5 (for files bigger than 5GB, it is not computed by default for smaller files).")
	flags.Int64Var(&common.UploadSwitchMultipart, "multipart-threshold", int64(100), "Files bigger than this size (in MB) will be uploaded using Multipart Upload.")
	flags.Int64Var(&common.S3RequestTimeout, "timeout", int64(-1), "Set an arbitrary timeout for each upload request (in sec.), default is no timeout.")
	RootCmd.AddCommand(scpFiles)
}
