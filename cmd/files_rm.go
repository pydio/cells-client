package cmd

import (
	"log"
	"path"
	"sync"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
)

const rmCmdExample = `# Path
./cec rm <workspace-slug>/path/to/resource

# Remove a single file
./cec rm common-files/target.txt

# Remove recursively inside a folder
./cec rm common-files/folder/*

# Remove a folder and all its children (even if it is not empty) 
./cec rm common-files/folder

# Remove multiple files
./cec rm common-files/file-1.txt common-files/file-2.txt
`

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Trash files or folders",
	Long: `
Deleting specified files or folders. In fact, it moves specified files or folders to the recycle bin that is at the root of the corresponding workspace.
`,
	Example: rmCmdExample,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Help()
			log.Fatalln("missing targets to remove")
		}
		targetNodes := make([]string, 0)
		for _, arg := range args {
			_, exists := rest.StatNode(arg)
			if !exists {
				log.Fatalf("No node found at %v, could not delete\n", arg)
			}
			if path.Base(arg) == "*" {
				nodes, err := rest.ListNodesPath(arg)
				if err != nil {
					log.Fatalf("Could not list nodes inside %s, aborting. Cause: %s\n", path.Dir(arg), err.Error())
				}
				targetNodes = nodes
			} else {
				targetNodes = append(targetNodes, arg)
			}
		}

		jobUUID, err := rest.DeleteNode(targetNodes)
		if err != nil {
			log.Fatalf("could not delete nodes, cause: %s\n", err)
		}

		var wg sync.WaitGroup
		for _, id := range jobUUID {
			wg.Add(1)
			go func(id string) {
				err := rest.MonitorJob(id)
				defer wg.Done()
				if err != nil {
					log.Printf("could not monitor job, %s\n", id)
				}
			}(id)
		}
		wg.Wait()
	},
}

func init() {
	RootCmd.AddCommand(rmCmd)
}
