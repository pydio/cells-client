package cmd

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pydio/cells-sdk-go/v5/models"

	"github.com/pydio/cells-client/v4/rest"
)

const (
	standardPrefix   = "cells://"
	completionPrefix = "cells//"

	// SDK Debug Flags for verbose modes
	//verbose
	vFlags = "retries | signing" // | request
	// very verbose
	vvFlags = "request | response | signing | retries | deprecated_usage"
	// For the record, there are 4 more flags:
	//  request_event_message, response_event_message, request_with_body & response_with_body
)

var (
	scpForce         bool
	scpNoProgress    bool
	scpQuiet         bool
	scpVerbose       bool
	scpVeryVerbose   bool
	scpMaxBackoffStr string
	scpS3DebugFlags  string
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

  Depending on your use-case, you might want to choose to use the 'scp' command in interactive mode, with a progress bar or with log messages, typically when launched from a script.
  
TROUBLESHOOTING 

  If you have problems with the transfer of big files and/or large tree structures, we strongly suggest to use the 'scp' command with a PAT and the '--no-progress' flag set. 

  You can also adjust the log level, e.g. with '--log debug' and choose which events are logged by the AWS SDK that performs the real work under the hood for multipart uploads.

  Known events type are: 
     signing, retries, request, request_with_body, response, response_with_body, deprecated_usage, request_event_message, response_event_message 
  that respectively turn on following AWS SDK log type: 
     aws.LogSigning, aws.LogRetries, aws.LogRequest, aws.LogRequestWithBody, aws.LogResponse, aws.LogResponseWithBody, aws.LogDeprecatedUsage, aws.LogRequestEventMessage, aws.LogResponseEventMessage
  You define the required mix with e.g: '--multipart-debug-flags="signing | retries"' (spaces are optional)  

  For convenience and retro-compatibility, we defined two 'shortcut' flags:
  '--verbose' is equivalent to '--no-progress --log info --multipart-debug-flags="signing | retries"'   
  '--very-verbose' is equivalent to '--no-progress --log debug --multipart-debug-flags="request | response | signing | retries | deprecated_usage"'   

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

		// Retrieve flags
		scpForce = viper.GetBool("force")
		scpNoProgress = viper.GetBool("no-progress")
		scpQuiet = viper.GetBool("quiet")
		scpVerbose = viper.GetBool("verbose")
		scpVeryVerbose = viper.GetBool("very-verbose")
		scpMaxBackoffStr = viper.GetString("retry-max-backoff")
		scpS3DebugFlags = viper.GetString("multipart-debug-flags")
		rest.UploadMaxPartsNumber = viper.GetInt64("max-parts-number")
		rest.UploadDefaultPartSize = viper.GetInt64("part-size")
		rest.UploadPartsConcurrency = viper.GetInt("parts-concurrency")
		rest.UploadSkipMD5 = viper.GetBool("skip-md5")
		rest.UploadSwitchMultipart = viper.GetInt64("multipart-threshold")
		rest.TransferRetryMaxAttempts = viper.GetInt("retry-max-attempts")

		// Keep backward retro-compatibility until v5 for old flags
		if viper.GetBool("no_progress") {
			scpNoProgress = true
		}
		if viper.GetBool("very_verbose") {
			scpVeryVerbose = true
		}
		if viper.GetBool("skip_md5") {
			rest.UploadSkipMD5 = true
		}
		if c := viper.GetInt("parts_concurrency"); c > -1 {
			rest.UploadPartsConcurrency = c
		}
		if c := viper.GetInt("retry_max_attempts"); c > -1 {
			rest.TransferRetryMaxAttempts = c
		}
		if c := viper.GetInt64("max_parts_number"); c > -1 {
			rest.UploadMaxPartsNumber = c
		}
		if c := viper.GetInt64("part_size"); c > -1 {
			rest.UploadDefaultPartSize = c
		}
		if c := viper.GetInt64("multipart_threshold"); c > -1 {
			rest.UploadSwitchMultipart = c
		}
		if c := viper.GetString("retry_max_backoff"); c != "" {
			scpMaxBackoffStr = c
		}

		// Handle aliases
		if scpVeryVerbose {
			scpNoProgress = true
			scpS3DebugFlags = vvFlags
			logger := rest.SetLogger(zapcore.DebugLevel)
			defer func(logger *zap.Logger) {
				_ = logger.Sync()
			}(logger)
		} else if scpVerbose {
			scpNoProgress = true
			scpS3DebugFlags = vFlags
			logger := rest.SetLogger(zapcore.InfoLevel)
			defer func(logger *zap.Logger) {
				_ = logger.Sync()
			}(logger)
		}

		if scpS3DebugFlags != "" {
			// We force recreation of the S3Client, to ensure the debug flags are correctly set
			e := sdkClient.ConfigureS3Logger(ctx, scpS3DebugFlags)
			if e != nil {
				rest.Log.Fatal(e)
			}
		}

		if scpMaxBackoffStr != "" {
			// Parse and set a specific backoff duration
			var e error
			rest.TransferRetryMaxBackoff, e = time.ParseDuration(scpMaxBackoffStr)
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
			rest.Log.Infof("After walking the tree, found %d nodes to delete, %d to create and %d to transfer", len(d), len(c), len(t))
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
			rest.Log.Infof("Now transferring files")
		}

		errs := targetNode.TransferAll(ctx, t, pool)
		if len(errs) > 0 {
			rest.Log.Infof("\nTransfer aborted after %d errors:", len(errs))
			for i, currErr := range errs {
				rest.Log.Infof("\t#%d: %s\n", i+1, currErr)
			}
			os.Exit(1)
		} else {
			rest.Log.Infoln("Transfer terminated")
		}
	},
}

func init() {
	flags := scpFiles.PersistentFlags()

	flags.BoolP("force", "f", false, "*DANGER* turns overwrite mode on: for a given item in the source tree, if a file or folder with same name already exists on the target side, it is merged or replaced.")
	flags.BoolP("no-progress", "n", false, "Do not show progress bar. You can then fine tune the log level")
	flags.BoolP("verbose", "v", false, "Hide progress bar and rather display more log info during the transfers")
	flags.BoolP("very-verbose", "w", false, "Hide progress bar and rather print out a maximum of log info")
	flags.BoolP("quiet", "q", false, "Reduce refresh frequency of the progress bars, e.g when running cec in a bash script")
	flags.Int64("max-parts-number", int64(5000), "Maximum number of parts, S3 supports 10000 but some storage require less parts.")
	flags.Int64("part-size", int64(50), "Default part size (MB), must always be a multiple of 10MB. It will be recalculated based on the max-parts-number value.")
	flags.Int("parts-concurrency", 3, "Number of concurrent part uploads.")
	flags.Bool("skip-md5", false, "Do not compute md5 (for files bigger than 5GB, it is not computed by default for smaller files).")
	flags.Int64("multipart-threshold", int64(100), "Files bigger than this size (in MB) will be uploaded using Multipart Upload.")
	flags.String("multipart-debug-flags", "", "Define flags to fine tune debug messages emitted by the underlying AWS SDK during multi-part uploads")
	flags.Int("retry-max-attempts", rest.TransferRetryMaxAttemptsDefault, "Limit the number of attempts before aborting. '0' allows the SDK to retry all retryable errors until the request succeeds, or a non-retryable error is thrown.")
	flags.String("retry-max-backoff", rest.TransferRetryMaxBackoffDefault.String(), "Maximum duration to wait after a part transfer fails, before trying again, expressed in Go duration format, e.g., '20s' or '3m'.")

	flags.Bool("no_progress", false, "Deprecated, rather use no-progress flag")
	flags.Bool("very_verbose", false, "Deprecated, rather use very-verbose flag")
	flags.Int64("max_parts_number", int64(-1), "Deprecated, rather use max-parts-number flag")
	flags.Int64("part_size", int64(-1), "Deprecated, rather use part-size flag")
	flags.Int("parts_concurrency", -1, "Deprecated, rather use parts-concurrency flag")
	flags.Bool("skip_md5", false, "Deprecated, rather use skip-md5 flag")
	flags.Int64("multipart_threshold", int64(-1), "Deprecated, rather use multipart-threshold flag")
	flags.Int("retry_max_attempts", -1, "Deprecated, rather use retry-max-attempts flag")
	flags.String("retry_max_backoff", "", "Deprecated, rather use retry-max-backoff flag")

	// Keep backward compatibility until v5 for old flag names
	// This does not work as expected, default from old command overwrite passed value from new flag
	//replaceMap := map[string]string{
	//	"no_progress": "no-progress",
	//	"very_verbose":        "very-verbose",
	//	"max_parts_number":    "max-parts-number",
	//	"part_size":           "part-size",
	//	"parts_concurrency":   "parts-concurrency",
	//	"skip_md5":            "skip-md5",
	//	"multipart_threshold": "multipart-threshold",
	//	"retry_max_attempts":  "retry-max-attempts",
	//	"retry_max_backoff":   "retry-max-backoff",
	//}
	// We pass an empty map and do retro-compatibility "manually" when retrieving the flags
	replaceMap := map[string]string{}

	if os.Getenv(EnvDisplayHiddenFlags) == "" {
		_ = flags.MarkHidden("no_progress")
		_ = flags.MarkHidden("very_verbose")
		_ = flags.MarkHidden("max_parts_number")
		_ = flags.MarkHidden("part_size")
		_ = flags.MarkHidden("parts_concurrency")
		_ = flags.MarkHidden("skip_md5")
		_ = flags.MarkHidden("multipart_threshold")
		_ = flags.MarkHidden("retry_max_attempts")
		_ = flags.MarkHidden("retry_max_backoff")
	}
	bindViperFlags(flags, replaceMap)
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
