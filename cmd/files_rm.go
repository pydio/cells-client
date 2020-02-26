package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
)

var rmCmdExample = `# Path
` + os.Args[0] + ` rm <workspace-slug>/path/to/resource

# Remove a single file
` + os.Args[0] + ` rm common-files/target.txt

# Remove recursively inside a folder
` + os.Args[0] + ` rm common-files/folder/*

# Remove a folder and all its children (even if it is not empty) 
` + os.Args[0] + ` rm common-files/folder

# Remove multiple files
` + os.Args[0] + ` rm common-files/file-1.txt common-files/file-2.txt

# You can force the deletion with the -f --force flag (to avoid the Yes or No)
` + os.Args[0] + ` rm -f common-files/file-1.txt
`

var (
	force bool
)

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

		// Ask for user approval before deleting
		p := promptui.Select{Label: "Are you sure", Items: []string{"No", "Yes"}}
		if !force {
			if _, resp, e := p.Run(); resp == "No" && e == nil {
				log.Println("Nothing will be deleted")
				return
			}
		}

		targetNodes := make([]string, 0)
		for _, arg := range args {
			_, exists := rest.StatNode(strings.TrimRight(arg, "*"))
			if !exists {
				log.Printf("Node not found %v, could not delete\n", arg)
			}
			if path.Base(arg) == "*" {
				nodes, err := rest.ListNodesPath(arg)
				if err != nil {
					log.Fatalf("Could not list nodes inside %s, aborting. Cause: %s\n", path.Dir(arg), err.Error())
				}
				targetNodes = append(targetNodes, nodes...)
			} else {
				targetNodes = append(targetNodes, arg)
			}
		}

		if len(targetNodes) <= 0 {
			log.Println("Nothing to delete")
			return
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

		fmt.Println("Nodes have been moved to the Recycle Bin")
	},
}

func init() {
	RootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&force, "force", "f", false, "Does not ask for user approval")
}
