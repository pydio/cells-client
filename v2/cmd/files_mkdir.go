package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/micro/go-log"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v2/client/tree_service"
	"github.com/pydio/cells-sdk-go/v2/models"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

var createAncestors bool

var mkDir = &cobra.Command{
	Use:   "mkdir",
	Short: `Create folder(s) in the remote server`,
	Long: `
DESCRIPTION

  Create a folder in the remote Cells server instance. 
  You must specify the full path, including the slug of an existing workspace.
  
  If parent folder does not exists, the command fails unless the '-p' flag is set.
  In such a case, non-existing folders are recursively created. 
  
  Warning: even if '-p' flag is set, trying to create a folder in an unknown or non-existent 
  workspace fails with error.

EXAMPLES

  # Simply create a folder under an already existing folder 'test' in 'common-files' workspace
  ` + os.Args[0] + ` mkdir common-files/test/new-folder

  # Create a full tree
  ` + os.Args[0] + ` mkdir -p common-files/a/folder/that/does/not/exits
`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 1 {
			cmd.PrintErrln("please provide the target path")
			os.Exit(1)
		}
		dir := args[0]
		parts := strings.Split(dir, "/")
		if len(parts) < 2 {
			cmd.PrintErrln("Please provide at least a workspace segment in the path")
			os.Exit(1)
		}

		spinner, err := common.NewSpinner().Start("Init")
		if err != nil {
			cmd.PrintErrf("spinner failed %s", err)
			os.Exit(1)
		}
		defer spinner.Stop()

		if quiet {
			common.DisableSpinnerOutput()
		}

		// Connect to the Pydio API via the sdkConfig
		ctx, apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err)
		}
		var dirs []*models.TreeNode
		var paths []string
		var crt = parts[0]

		// Checking existence of parent workspace
		if _, e := apiClient.TreeService.HeadNode(&tree_service.HeadNodeParams{Node: crt, Context: ctx}); e != nil {
			spinner.Fail(fmt.Sprintf("Could not find workspace %s. Please specify a parent workspace that exists.", crt))
			os.Exit(1)
		}

		for i := 1; i < len(parts)-1; i++ {
			crt = path.Join(crt, parts[i])
			_, e := apiClient.TreeService.HeadNode(&tree_service.HeadNodeParams{Node: crt, Context: ctx})
			if e != nil {
				if createAncestors {
					dirs = append(dirs, &models.TreeNode{Path: crt})
					paths = append(paths, crt)
				} else {
					cmd.PrintErrf("Could not find folder at %s, double check and correct your path or use the '-p' flags if you want to force the creation of missing ancestors.", crt)
					os.Exit(1)
				}
			}
		}
		// always create the leaf folder
		crt = path.Join(crt, parts[len(parts)-1])
		dirs = append(dirs, &models.TreeNode{Path: crt})
		paths = append(paths, crt)

		if len(dirs) == 0 {
			spinner.Warning("All dirs already exist, exiting")
			return
		}
		spinner.UpdateText(fmt.Sprintf("Creating folder(s) %s", strings.Join(paths, ", ")))

		_, err = apiClient.TreeService.CreateNodes(&tree_service.CreateNodesParams{
			Body: &models.RestCreateNodesRequest{
				Nodes: dirs,
			},
			Context: ctx,
		})
		if err != nil {
			spinner.Fail("error while calling CreateNodes: ", err)
			os.Exit(1)
		}
		// Wait that it is indexed
		e := rest.RetryCallback(func() error {
			_, e := apiClient.TreeService.HeadNode(&tree_service.HeadNodeParams{Node: dir, Context: ctx})
			if e != nil {
				spinner.UpdateText("Waiting for folder to be correctly indexed...")
			}
			return e
		}, 10, 2*time.Second)

		if e != nil {
			log.Fatal(e)
		}
		spinner.Success("Created and indexed ", dir)
	},
}

func init() {

	flags := mkDir.PersistentFlags()
	flags.BoolVarP(&createAncestors, "parents", "p", false, "Force creation of non-existing ancestors")

	RootCmd.AddCommand(mkDir)
}
