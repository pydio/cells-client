package cmd

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pydio/cells-sdk-go/v5/models"

	"github.com/pydio/cells-client/v4/common"
	"github.com/pydio/cells-client/v4/rest"
)

const (
	standardPrefix   = "cells://"
	completionPrefix = "cells//"
)

var (
	scpForce         bool
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

  For convenience, if the *target* folder does not exist (but its parent does), we create it.

  On the other hand, we check if an item with the same name already exists on the target side and abort the transfer with an error in such case. 
  You might want to enable the "force" mode. 
  Then, when 'old' (existing) and 'new' item have the same name, if:    
    - 'old' and 'new' are both files: 'new' replaces 'old'
    - 'old' and 'new' are of a different type: we first erase 'old' in the target and then copy (recursively) 'new'
    - both folder: for each child of 'new' we try to copy in 'old'. If an item with same name already exists on the target side, we apply the rules recursively.
  WARNING: this could lead to erasing data on the target side. Only use with extra care.
  
EXAMPLES

  1/ Uploading a file to the server:
  $ ` + os.Args[0] + ` scp ./README.md cells://common-files
  Copying ./README.md to cells://common-files
  Waiting for file to be indexed...
  File correctly indexed

  2/ Download a file from server:
  $ ` + os.Args[0] + ` scp cells://personal-files/funnyCat.jpg ./
  Copying cells://personal-files/funnyCat.jpg to /home/pydio/downloads

  3/ Download a folder to an existing target, using existing folders when they are already here but re-downloading files: 
  $ ` + os.Args[0] + ` scp --force cells//common-files/my-folder ./tests
  Downloading cells://common-files/my-folder to /home/pydio/downloads/tests

  Copying cells//common-files/my-folder to /home/pydio/tests	
`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		rest.DryRun = false // Debug option
		ctx := cmd.Context()
		from := args[0]
		to := args[1]
		scpCurrentPrefix := ""
		isSrcLocal := false

		// Handle aliases
		if scpVeryVerbose {
			scpNoProgress = true
			common.CurrentLogLevel = common.Trace
			logger := rest.SetLogger(zapcore.DebugLevel)
			defer func(logger *zap.Logger) {
				_ = logger.Sync()
			}(logger)
		} else if scpVerbose {
			scpNoProgress = true
			common.CurrentLogLevel = common.Debug
			logger := rest.SetLogger(zapcore.InfoLevel)
			defer func(logger *zap.Logger) {
				_ = logger.Sync()
			}(logger)
		} else {
			common.CurrentLogLevel = common.Info
		}

		if scpMaxBackoffStr != "" {
			// Parse and set a specific backoff duration
			var e error
			common.TransferRetryMaxBackoff, e = time.ParseDuration(scpMaxBackoffStr)
			if e != nil {
				rest.Log.Fatalln("could not parse backoff duration:", e)
			}
		}

		// Handle multiple prefix cells:// (standard) and cells// (to enable completion)
		// Clever exclusive "OR"
		if strings.HasPrefix(from, standardPrefix) != strings.HasPrefix(to, standardPrefix) {
			scpCurrentPrefix = standardPrefix
		} else if strings.HasPrefix(from, completionPrefix) != strings.HasPrefix(to, completionPrefix) {
			scpCurrentPrefix = completionPrefix
		} else // Not a valid SCP transfer
		if strings.HasPrefix(from, standardPrefix) || strings.HasPrefix(from, completionPrefix) {
			rest.Log.Fatalln("Rather use the cp command to copy one or more file on the server side")
		} else {
			rest.Log.Fatalln("Source and target are both on your client machine, copy from server to client or the opposite")
		}
		// Now it's easy to check if we do upload or download (that is default)
		if strings.HasPrefix(to, scpCurrentPrefix) {
			isSrcLocal = true
		}

		// Prepare paths
		var srcPath, targetPath string
		var needMerge bool
		var err error
		if isSrcLocal { // Upload
			srcPath, err = filepath.Abs(from)
			if err != nil {
				rest.Log.Fatalf("%s is not a valid source: %s", from, err)
			}
			srcName := filepath.Base(srcPath)
			targetPath = strings.TrimPrefix(to, scpCurrentPrefix)
			if needMerge, err = preProcessRemoteTarget(ctx, sdkClient, srcName, targetPath, scpForce); err != nil {
				rest.Log.Fatalln(err)
			}
			rest.Log.Infof("Uploading %s to %s", srcPath, standardPrefix+targetPath)
		} else { // Download

			srcPath = strings.TrimPrefix(from, scpCurrentPrefix)
			srcName := filepath.Base(srcPath)

			targetPath, err = filepath.Abs(to)
			if err != nil {
				rest.Log.Fatalf("%s is not a valid destination: %s", to, err)
			}
			if needMerge, err = preProcessLocalTarget(srcName, targetPath, scpForce); err != nil {
				rest.Log.Fatalln(err)
			}
			rest.Log.Infof("Downloading %s to %s", standardPrefix+srcPath, targetPath)
		}

		// Now create source and target crawlers
		srcNode, e := rest.NewCrawler(ctx, sdkClient, srcPath, isSrcLocal)
		if e != nil {
			rest.Log.Fatalln(e)
		}

		targetNode := rest.NewTarget(sdkClient, targetPath, !isSrcLocal, srcNode.IsDir, scpForce)
		if e != nil {
			rest.Log.Fatalln(e)
		}

		// Walk the full source tree to prepare a list of node to create
		var tf *rest.CrawlNode
		if needMerge {
			tf = targetNode
		}
		t, c, d, e := srcNode.Walk(cmd.Context(), tf)
		if e != nil {
			rest.Log.Fatal(e)
		}

		var pool *rest.BarsPool = nil
		if !scpNoProgress {
			refreshInterval := time.Millisecond * 10 // this is the default
			if scpQuiet {
				refreshInterval = time.Millisecond * 3000
			}
			pool = rest.NewBarsPool(len(t)+len(c)+len(d) > 1, len(t)+len(c)+len(d), refreshInterval)
			pool.Start()
		} else {
			rest.Log.Infof("After walking the tree, found %d nodes to delete, %d to create and %d transfer", len(d), len(c), len(t))
		}

		// Delete necessary items
		e = targetNode.DeleteForMerge(ctx, d, pool)
		if e != nil {
			if pool != nil { // Force stop of the pool that stays blocked otherwise
				pool.Stop()
			}
			rest.Log.Fatal(e)
		}

		// CREATE FOLDERS
		e = targetNode.CreateFolders(ctx, targetNode, c, pool)
		if e != nil {
			if pool != nil { // Force stop of the pool that stays blocked otherwise
				pool.Stop()
			}
			rest.Log.Fatal(e)
		}

		// UPLOAD / DOWNLOAD FILES
		if scpNoProgress {
			rest.Log.Infof("... Now transferring files")
		}

		errs := targetNode.TransferAll(ctx, t, pool)
		if len(errs) > 0 {
			//if pool != nil { // Force stop of the pool that stays blocked otherwise
			//	pool.Stop()
			//}
			rest.Log.Infof("\n... Transfer aborted after %d errors:", len(errs))
			for i, currErr := range errs {
				rest.Log.Infof("\t#%d: %s\n", i+1, currErr)
			}
			os.Exit(1)
		} else {
			rest.Log.Infoln("... Transfer terminated")
		}
		//if !scpNoProgress {
		//	fmt.Println("") // Add a line to reduce glitches in the terminal
		//}
	},
}

func init() {
	flags := scpFiles.PersistentFlags()
	flags.BoolVarP(&scpForce, "force", "f", false, "*DANGER* turns overwrite mode on: for a given item in the source tree, if a file or folder with same name already exists on the target side, it is merged or replaced.")
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

func preProcessRemoteTarget(ctx context.Context, sdkClient *rest.SdkClient, srcName string, toPath string, force bool) (bool, error) {
	targetParent, ok := sdkClient.StatNode(ctx, toPath)
	if ok {
		if *targetParent.Type == models.TreeNodeTypeCOLLECTION {
			// TODO ensure it is writable
			_, ok2 := sdkClient.StatNode(ctx, path.Join(toPath, srcName))
			if ok2 { // an item with same name exists at target location
				if force {
					return true, nil
				} else {
					return false, fmt.Errorf("a file or folder named '%s' already exists on the server at '%s', we cannot proceed", srcName, toPath)
				}
			}
			return false, nil // Happy path
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
	parNode, ok2 := sdkClient.StatNode(ctx, parPath)
	if !ok2 { // Target parent folder does not exist, we do not create it
		return false, fmt.Errorf("target parent folder %s does not exist on remote server. ", parPath)
	} else if *parNode.Type != models.TreeNodeTypeCOLLECTION {
		return false, fmt.Errorf("target parent %s is also not a folder on remote server. ", parPath)
	} else { // Parent folder exists, we will try to also create the target folder`
		return false, nil
	}
}

func preProcessLocalTarget(srcName, targetPath string, force bool) (bool, error) {
	toPath, e := filepath.Abs(targetPath)
	if e != nil {
		return false, e
	}
	_, e = os.Stat(toPath)
	if e != nil { // target does *not* exist on the client machine
		parPath := filepath.Dir(toPath)
		if parPath == "." { // this should never happen
			return false, fmt.Errorf("target path %s does not exist on client machine, please double check and correct. ", toPath)
		}
		if ln, err2 := os.Stat(parPath); err2 != nil { // Target parent folder does not exist on client machine, we do not create it
			return false, fmt.Errorf("target parent folder %s does not exist in client machine. ", parPath)
		} else if !ln.IsDir() { // Local parent is not a folder
			return false, fmt.Errorf("target parent %s is not a folder, could not download to it. ", parPath)
		} else { // Parent folder exists on local -> we rely on the following step to create the target folder.
			return false, nil
		}
	}
	// Target folder exists locally, check if an item with srcName exists
	_, e = os.Stat(filepath.Join(toPath, srcName))
	if e != nil { // nope -> Happy path
		return false, nil
	} else if force { // Exists but who cares, yolo!
		return true, nil
	} else {
		return false, fmt.Errorf("a file or folder named '%s' already exists on your machine at '%s', we cannot proceed", srcName, toPath)
	}
}
