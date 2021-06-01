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

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

var (
	force        bool
	wildcardChar = "%"
)

// generates the description
func rmDescription(bin string) string {
	return `
DESCRIPTION
	
  Delete specified files or folders. 
	
  In fact, it only moves specified files or folders to the recycle bin 
  that is at the root of the corresponding workspace, the trashed objects 
  can be restored (from the web UI, this feature is not yet implemented 
  in the Cells Client) 

EXAMPLES

  # Generic example:
  ` + bin + ` rm <workspace-slug>/path/to/resource

  # Remove a single file:
  ` + bin + ` rm common-files/target.txt

  # Remove recursively inside a folder, the wildcard is '%':
  ` + bin + ` rm common-files/folder/%

  # Remove a folder and all its children (even if it is not empty)
  ` + bin + ` rm common-files/folder

  # Remove multiple files
  ` + bin + ` rm common-files/file-1.txt common-files/file-2.txt

  # You can force the deletion with the '--force' flag (to avoid the Yes or No)
  ` + bin + ` rm -f common-files/file-1.txt
`
}

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Trash files or folders",
	Long:  rmDescription(os.Args[0]),
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		// Ask for user approval before deleting
		p := promptui.Select{Label: "Are you sure", Items: []string{"No", "Yes"}}
		if !force {
			if _, resp, e := p.Run(); resp == "No" && e == nil {
				log.Println("Nothing will be deleted")
				return
			}
		}

		spinner, err := common.NewSpinner().Start("Removing Nodes")
		if err != nil {
			log.Println("spinner failed", err)
		}
		defer spinner.Stop()

		if quiet {
			common.DisableSpinnerOutput()
		}

		targetNodes := make([]string, 0)
		for _, arg := range args {
			_, exists := rest.StatNode(strings.TrimRight(arg, wildcardChar))
			if !exists {
				spinner.Fail(fmt.Sprintf("Node not found [%v], could not delete\n", arg))
				os.Exit(1)
			}
			if path.Base(arg) == wildcardChar {
				dir, _ := path.Split(arg)
				newArg := path.Join(dir, "*")
				nodes, err := rest.ListNodesPath(newArg)

				// Remove recycle_bin from targetedNodes
				for i, c := range nodes {
					if path.Base(c) == "recycle_bin" {
						nodes = append(nodes[:i], nodes[i+1:]...)
					}
				}

				if err != nil {
					spinner.Fail(fmt.Sprintf("Could not list nodes inside %s, aborting. Cause: %s\n", path.Dir(arg), err.Error()))
					os.Exit(1)
				}
				targetNodes = append(targetNodes, nodes...)
			} else {
				targetNodes = append(targetNodes, arg)
			}
		}

		if len(targetNodes) <= 0 {
			spinner.Warning("Nothing to delete")
			os.Exit(1)
		}

		jobUUID, err := rest.DeleteNode(targetNodes)
		if err != nil {
			spinner.Fail(fmt.Sprintf("could not delete nodes, cause: %s\n", err))
			os.Exit(1)
		}

		var wg sync.WaitGroup
		for _, id := range jobUUID {
			wg.Add(1)
			go func(id string) {
				err := rest.MonitorJob(id)
				defer wg.Done()
				if err != nil {
					spinner.Warning("could not monitor job, %s\n", id)
				}
			}(id)
		}
		wg.Wait()

		spinner.Success("Nodes have been moved to the Recycle Bin")
	},
}

func init() {
	RootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&force, "force", "f", false, "Do not ask for user approval")
}
