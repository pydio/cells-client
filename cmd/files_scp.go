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

	"github.com/pydio/cells-sdk-go/v5/models"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/common"
	"github.com/pydio/cells-client/v4/rest"
)

const (
	prefixA = "cells://"
	prefixB = "cells//"
)

var (
	scpNoProgress    bool
	scpQuiet         bool
	scpVerbose       bool
	scpVeryVerbose   bool
	scpMaxBackoffStr string
)

var scpFiles = &cobra.Command{
	Use:   "scp",
	Short: `Copy files from/to Cells`,
	Long: `
DESCRIPTION

  Copy files from the client machine to your Pydio Cells server instance (and vice versa).

  To differentiate local from remote, prefix remote paths with 'cells://' or with 'cells//' (without the column) if you have installed the completion and intend to use it.
  For the time being, copy can only be performed from the client machine to the server or the other way round:
  it is not yet possible to directly transfer files from one Cells instance to another.

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
		rest.DryRun = false // Debug option
		ctx := cmd.Context()
		from := args[0]
		to := args[1]
		scpCurrentPrefix := ""
		isSrcLocal := false

		if scpMaxBackoffStr != "" {
			// Parse and set a specific backoff duration
			var e error
			common.TransferRetryMaxBackoff, e = time.ParseDuration(scpMaxBackoffStr)
			if e != nil {
				log.Fatal("could not parse backoff duration:", e)
				return
			}
		}

		if scpVeryVerbose {
			common.CurrentLogLevel = common.Trace
		} else if scpVerbose {
			common.CurrentLogLevel = common.Debug
		} else {
			common.CurrentLogLevel = common.Info
		}

		// Handle multiple prefix cells:// (standard) and cells// (to enable completion)
		// Clever exclusive "OR"
		if strings.HasPrefix(from, prefixA) != strings.HasPrefix(to, prefixA) {
			scpCurrentPrefix = prefixA
		} else if strings.HasPrefix(from, prefixB) != strings.HasPrefix(to, prefixB) {
			scpCurrentPrefix = prefixB
		} else // Not a valid SCP transfer
		if strings.HasPrefix(from, prefixA) || strings.HasPrefix(from, prefixB) {
			log.Fatal("Rather use the cp command to copy one or more file on the server side.")
		} else {
			log.Fatal("Source and target are both on your client machine, copy from server to client or the opposite.")
		}
		// Now it's easy to check if we do upload or download (that is default)
		if strings.HasPrefix(to, scpCurrentPrefix) {
			isSrcLocal = true
		}

		// Prepare paths
		var crawlerPath, targetPath string
		var rename bool
		var err error
		if isSrcLocal { // Upload
			targetPath = strings.TrimPrefix(to, scpCurrentPrefix)
			crawlerPath, err = filepath.Abs(from)
			if err != nil {
				log.Fatalf("%s is not a valid source: %s", from, err)
			}
			srcName := filepath.Base(crawlerPath)
			if rename, err = prepareRemoteTargetPath(ctx, srcName, targetPath); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Uploading '%s' to '%s'\n", crawlerPath, to)
		} else { // Download
			crawlerPath = strings.TrimPrefix(from, scpCurrentPrefix)
			srcName := filepath.Base(crawlerPath)
			targetPath, err = filepath.Abs(to)
			if err != nil {
				log.Fatalf("%s is not a valid destination: %s", to, err)
			}
			err = prepareLocalTargetPath(srcName, targetPath)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Downloading '%s' to '%s'\n", from, targetPath)
		}

		// Now create source and target crawlers
		srcNode, e := rest.NewCrawler(ctx, crawlerPath, isSrcLocal)
		if e != nil {
			log.Fatal(e)
		}
		targetNode, e := rest.NewTarget(cmd.Context(), targetPath, srcNode, rename)
		if e != nil {
			log.Fatal(e)
		}

		// Walk the full source tree to prepare a list of node to create
		nn, e := srcNode.Walk(cmd.Context())
		if e != nil {
			log.Fatal(e)
		}

		var pool *rest.BarsPool = nil
		//if common.CurrentLogLevel == common.Info {
		if !scpNoProgress {
			refreshInterval := time.Millisecond * 10 // this is the default
			if scpQuiet {
				refreshInterval = time.Millisecond * 3000
			}
			pool := rest.NewBarsPool(len(nn) > 1, len(nn), refreshInterval)
			pool.Start()
		} else {
			fmt.Printf("... After walking the tree, found %d nodes to copy\n", len(nn))
			fmt.Println("... First creating folders")
		}

		// CREATE FOLDERS
		e = targetNode.MkdirAll(ctx, nn, pool)
		if e != nil {
			if pool != nil { // Force stop of the pool that stays blocked otherwise
				pool.Stop()
			}
			log.Fatal(e)
		}

		// UPLOAD / DOWNLOAD FILES
		errs := targetNode.CopyAll(ctx, nn, pool)
		if len(errs) > 0 {
			log.Fatal(errs)
		}
		if !scpNoProgress {
			fmt.Println("xxxxx") // Add a line to reduce glitches in the terminal
		}
	},
}

func prepareRemoteTargetPath(ctx context.Context, srcName string, toPath string) (bool, error) {
	// FIXME Check target path existence and handle rename corner cases
	targetParent, ok := rest.StatNode(ctx, toPath)
	if ok {
		if *targetParent.Type == models.TreeNodeTypeCOLLECTION {
			// TODO ensure it is writable
			_, ok2 := rest.StatNode(ctx, path.Join(toPath, srcName))
			if ok2 {
				// return true, nil
				return false, fmt.Errorf("a file or folder named '%s' already exists on the server at '%s', we cannot proceed", srcName, toPath)
			}
			return false, nil
		} else {
			return false, fmt.Errorf("target path %s is not a folder, we cannot proceed", toPath)
		}
	}
	parPath, _ := path.Split(toPath)
	if parPath == "" {
		// Non-existing workspace
		return false, fmt.Errorf("Please define at list a workspace on the server, e.g.: cells://common-filestarget parent path %s does not exist on the server, please double check and correct. ", toPath)
	}
	// Check if the parent exists
	parNode, ok2 := rest.StatNode(ctx, parPath)
	if !ok2 { // Target parent folder does not exist, we do not create it
		return false, fmt.Errorf("target parent folder %s does not exist on remote server. ", parPath)
	} else if *parNode.Type != models.TreeNodeTypeCOLLECTION {
		return false, fmt.Errorf("target parent %s is also not a folder on remote server. ", parPath)
	} else { // Parent folder exists, we will try to also create the target folder`
		return false, nil
	}
}

func prepareLocalTargetPath(srcName, targetPath string) error {
	toPath, e := filepath.Abs(targetPath)
	if e != nil {
		return e
	}
	_, e = os.Stat(toPath)
	if e != nil { // target does *not* exist on the client machine
		parPath := filepath.Dir(toPath)
		if parPath == "." { // this should never happen
			return fmt.Errorf("target path %s does not exist on client machine, please double check and correct. ", toPath)
		}
		// TODO We create the parent if it does not exists in the local machine, do we really want this?
		if ln, err2 := os.Stat(parPath); err2 != nil { // Target parent folder does not exist on client machine, we do not create it
			return fmt.Errorf("target parent folder %s does not exist in client machine. ", parPath)
		} else if !ln.IsDir() { // Local parent is not a folder
			return fmt.Errorf("target parent %s is not a folder, could not download to it. ", parPath)
		} else { // Parent folder exists on local
			return nil
		}
	}
	// target exists on the local machine, we ensure it does not contain an item with the given name
	_, e = os.Stat(filepath.Join(toPath, srcName))
	if e == nil {
		return fmt.Errorf("a file or folder named '%s' already exists on your machine at '%s', we cannot proceed", srcName, toPath)
	}
	return nil

}

func init() {
	flags := scpFiles.PersistentFlags()
	flags.BoolVarP(&scpNoProgress, "no_progress", "n", false, "Do not show progress bar. You can then fine tune the log level")
	flags.BoolVarP(&scpVerbose, "verbose", "v", false, "Hide progress bar and rather display more log info during the transfers")
	flags.BoolVarP(&scpVeryVerbose, "very_verbose", "w", false, "Hide progress bar and rather print out a maximum of log info")
	flags.BoolVarP(&scpQuiet, "quiet", "q", false, "Reduce refresh frequency of the progress bars, e.g when running cec in a bash script")
	flags.Int64Var(&common.UploadMaxPartsNumber, "max_parts_number", int64(5000), "Maximum number of parts, S3 supports 10000 but some storage require less parts.")
	flags.Int64Var(&common.UploadDefaultPartSize, "part_size", int64(50), "Default part size (MB), must always be a multiple of 10MB. It will be recalculated based on the max-parts-number value.")
	flags.IntVar(&common.UploadPartsConcurrency, "parts_concurrency", 3, "Number of concurrent part uploads.")
	flags.BoolVar(&common.UploadSkipMD5, "skip_md5", false, "Do not compute md5 (for files bigger than 5GB, it is not computed by default for smaller files).")
	flags.Int64Var(&common.UploadSwitchMultipart, "multipart_threshold", int64(100), "Files bigger than this size (in MB) will be uploaded using Multipart Upload.")
	flags.IntVar(&common.TransferRetryMaxAttempts, "retry_max_attempts", common.TransferRetryMaxAttemptsDefault, "Limit the number of attempts before aborting. '0' allows the SDK to retry all retryable errors until the request succeeds, or a non-retryable error is thrown.")
	flags.StringVar(&scpMaxBackoffStr, "retry_max_backoff", common.TransferRetryMaxBackoffDefault.String(), "Maximum duration to wait after a part transfer fails, before trying again, expressed in Go duration format, e.g., '20s' or '3m'.")
	RootCmd.AddCommand(scpFiles)
}
