package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

func mvDescription(bin string) string {
	return `
DESCRIPTION
	
  Synchronously move or rename one or more files or folders within your Cells server.
  It works within the same workspace or from one to another, as long as
  the current user has sufficient permission on both workspaces.

EXAMPLES

  Move a node:
  ` + bin + ` mv common-files/picture.jpg personal-files/photos/

  Rename a node:
  ` + bin + ` mv common-files/picture.jpg common-files/p2.jpg

  Move all nodes recursively:
  ` + bin + ` mv common-files/photos/* personal-files/photos/
`
}

// filesMvCmd represents the filesMv command
var filesMvCmd = &cobra.Command{
	Use:   "mv",
	Short: "Move and/or rename nodes on the server",
	Long:  mvDescription(os.Args[0]),
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		source := args[0]
		target := args[1]

		spinner, err := common.NewSpinner().Start()
		if err != nil {
			cmd.PrintErrf("spinner failed %s", err)
			os.Exit(1)
		}
		defer spinner.Stop()

		if quiet {
			common.DisableSpinnerOutput()
		}

		var sourceNodes []string
		if path.Base(source) == "*" {
			nodes, err := rest.ListNodesPath(source)
			if err != nil {
				spinner.Warning("could not list the nodes path", err)
			}
			sourceNodes = nodes
		} else {
			_, exists := rest.StatNode(source)
			if !exists {
				spinner.Fail(fmt.Sprintf("This node does not exist: [%v]\n", source))
				return
			}
			sourceNodes = []string{source}
		}

		params := rest.MoveParams(sourceNodes, target)
		jobID, err := rest.MoveJob(params)
		if err != nil {
			spinner.Fail("Could not run job:", err.Error())
			return
		}

		err = rest.MonitorJob(jobID)
		if err != nil {
			spinner.Fail("Could not monitor job:", err.Error())
			return
		}
		spinner.Success(fmt.Sprintf("moved %s to %s", source, target))
	},
}

func init() {
	RootCmd.AddCommand(filesMvCmd)
}
