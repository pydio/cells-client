package cmd

import (
	"log"
	"path"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
)

const filesMvCmdExample = `
# Move a node
./cec mv common-files/formula-one.jpg personal-files/photos/

# Rename a node
./cec mv common-files/formula-one.jpg common-files/f1.jpg

# Move all nodes recursively 
./cec mv common-files/photos/* personal-files/photos/
`

// filesMvCmd represents the filesMv command
var filesMvCmd = &cobra.Command{
	Use:     "mv",
	Short:   "Move, Rename nodes",
	Example: filesMvCmdExample,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			cmd.Help()
			log.Fatalln("Missing Source and Target path")
		}
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
			log.Fatalln("could not run job")
		}

		err = rest.MonitorJob(jobID)
		if err != nil {
			log.Fatalln("could not monitor job")
		}
	},
}

func init() {
	RootCmd.AddCommand(filesMvCmd)
}
