package cmd

import (
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/rest"
)

// filesMvCmd represents the filesMv command
var filesMvCmd = &cobra.Command{
	Use:   "mv",
	Short: "Move and/or rename nodes on the server",
	Long: `
DESCRIPTION
	
  Synchronously move or rename one or more files or folders within your Cells server.
  It works within the same workspace or from one to another, as long as
  the current user has sufficient permission on both workspaces.

EXAMPLES

  Move a node:
  ` + os.Args[0] + ` mv common-files/picture.jpg personal-files/photos/

  Rename a node:
  ` + os.Args[0] + ` mv common-files/picture.jpg common-files/p2.jpg

  Move all nodes recursively:
  ` + os.Args[0] + ` mv common-files/photos/* personal-files/photos/
`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		source := args[0]
		target := args[1]

		ctx := cmd.Context()

		var sourceNodes []string
		if path.Base(source) == "*" {
			nodes, err := rest.ListNodesPath(ctx, source)
			if err != nil {
				log.Println("could not list the nodes path", err)
			}
			sourceNodes = nodes
		} else {
			_, exists := rest.StatNode(ctx, source)
			if !exists {
				log.Fatalf("This node does not exist: [%v]\n", source)
			}
			sourceNodes = []string{source}
		}

		params := rest.MoveParams(sourceNodes, target)
		jobID, err := rest.MoveJob(ctx, params)
		if err != nil {
			log.Fatalln("Could not run job:", err.Error())
		}

		err = rest.MonitorJob(ctx, jobID)
		if err != nil {
			log.Fatalln("Could not monitor job:", err.Error())
		}
	},
}

func init() {
	RootCmd.AddCommand(filesMvCmd)
}
