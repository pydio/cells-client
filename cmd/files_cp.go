package cmd

import (
	"log"
	"path"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
)

const (
	cpCmdExample = `
Copy content in your Cells instance with,

# Copy file "test.txt" inside folder "folder-a"
./cec cp common-files/test.txt common-files/folder-a

# Copy file "test.txt" inside folder "folder-b" (located in another workspace/datasource)
./cec cp common-files/test.txt personal-files/folder-b

# Copiy all the content of folder "test" inside "folder-c"
./cec cp common-files/test/* common-files/folder-c
`
)

// cmCmd represents the rm command
var cpCmd = &cobra.Command{
	Use:     "cp",
	Short:   "Copy files",
	Example: cpCmdExample,
	Run: func(cmd *cobra.Command, args []string) {
		//TODO Maybe add the dot (.) behaviour to simulate the linux command
		if len(args) < 2 {
			cmd.Help()
			log.Fatalln("Missing Source and Target path")
		}
		source := args[0]
		target := args[1]

		var sourceNodes []string
		if path.Base(source) == "*" {
			nodes, _ := rest.ListNodesPath(source)
			sourceNodes = nodes
		} else {
			sourceNodes = []string{source}
		}

		//TODO refactor
		jobName := "copy"
		params := rest.CopyParams(sourceNodes, target)
		jobID, err := rest.RunJob(jobName, params)
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
	RootCmd.AddCommand(cpCmd)
}
