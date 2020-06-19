package cmd

import (
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var filesMvCmdExample = `
# Move a node
` + os.Args[0] + ` mv common-files/formula-one.jpg personal-files/photos/

# Rename a node
` + os.Args[0] + ` mv common-files/formula-one.jpg common-files/f1.jpg

# Move all nodes recursively 
` + os.Args[0] + ` mv common-files/photos/* personal-files/photos/
`

// filesMvCmd represents the filesMv command
var filesMvCmd = &cobra.Command{
	Use:     "mv",
	Short:   "Move and/or rename nodes on the server",
	Example: filesMvCmdExample,
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		source := args[0]
		target := args[1]

		var sourceNodes []string
		if path.Base(source) == "*" {
			nodes, err := rest.ListNodesPath(source)
			if err != nil {
				log.Println("could not list the nodes path", err)
			}
			sourceNodes = nodes
		} else {
			_, exists := rest.StatNode(source)
			if !exists {
				log.Fatalf("This node does not exist: [%v]\n", source)
			}
			sourceNodes = []string{source}
		}

		params := rest.MoveParams(sourceNodes, target)
		jobID, err := rest.MoveJob(params)
		if err != nil {
			log.Fatalln("Could not run job:", err.Error())
		}

		err = rest.MonitorJob(jobID)
		if err != nil {
			log.Fatalln("Could not monitor job:", err.Error())
		}
	},
}

func init() {
	RootCmd.AddCommand(filesMvCmd)
}
