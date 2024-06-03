package cmd

import (
	"os"
	"path"
	"strings"
	"sync"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/rest"
)

var (
	rmPermanently  bool
	rmForce        bool
	rmWildcardChar = "%"
)

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Trash files or folders",
	Long: `
DESCRIPTION
	
  Delete specified files or folders. 
	
  By default, we only move specified files or folders to the recycle bin 
  that is at the root of the corresponding workspace. The trashed items 
  can be then restored from the web UI (this feature is not yet implemented 
  in the Cells Client). Use the 'permanently' flag to skip the recycle and 
  definitively remove the corresponding items.

EXAMPLES

  # Generic example:
  ` + os.Args[0] + ` rm <workspace-slug>/path/to/resource

  # Remove a single file:
  ` + os.Args[0] + ` rm common-files/target.txt

  # Remove recursively inside a folder, the wildcard is '%':
  ` + os.Args[0] + ` rm common-files/folder/%

  # Remove a folder and all its children (even if it is not empty)
  ` + os.Args[0] + ` rm common-files/folder

  # Remove multiple files
  ` + os.Args[0] + ` rm common-files/file-1.txt common-files/file-2.txt

  # You can force the deletion with the '--force' flag (to avoid the Yes or No)
  ` + os.Args[0] + ` rm -f common-files/file-1.txt

  # Skip the recycle and permanently remove a file
  ` + os.Args[0] + ` rm -p common-files/file-1.txt

  # DANGER: directly and permanently remove a folder and all its children
  ` + os.Args[0] + ` rm -pf common-files/folder

`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		// Ask for user approval before deleting
		if !rmForce {
			p := promptui.Select{Label: "Are you sure", Items: []string{"No", "Yes"}}
			if _, resp, e := p.Run(); resp == "No" && e == nil {
				cmd.Println(promptui.IconBad, "Aborted by user")
				return
			}
		}
		ctx := cmd.Context()
		targetNodes := make([]string, 0)
		for _, arg := range args {
			_, exists := sdkClient.StatNode(ctx, strings.TrimRight(arg, rmWildcardChar))
			if !exists {
				rest.Log.Warnf("Node %s not found: it could not be deleted", arg)
				continue
			}
			if path.Base(arg) == rmWildcardChar {
				dir, _ := path.Split(arg)
				newArg := path.Join(dir, "*")
				nodes, err := sdkClient.ListNodesPath(ctx, newArg)

				// Remove recycle_bin from targetedNodes
				for i, c := range nodes {
					if path.Base(c) == "recycle_bin" {
						nodes = append(nodes[:i], nodes[i+1:]...)
					}
				}

				if err != nil {
					rest.Log.Fatalf("Could not list nodes inside %s, aborting. Cause: %s\n", path.Dir(arg), err.Error())
				}
				targetNodes = append(targetNodes, nodes...)
			} else {
				targetNodes = append(targetNodes, arg)
			}
		}

		if len(targetNodes) <= 0 {
			cmd.Println("Nothing to delete")
			return
		}

		jobUUID, err := sdkClient.DeleteNodes(ctx, targetNodes, rmPermanently)
		if err != nil {
			rest.Log.Fatalf("could not delete nodes, cause: %s\n", err)
		}

		var wg sync.WaitGroup
		for _, id := range jobUUID {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				err := sdkClient.MonitorJob(ctx, id)
				if err != nil {
					rest.Log.Warnf("could not monitor job %s: %s", id, err.Error())
				}
			}(id)
		}
		wg.Wait()

		if rmPermanently {
			if len(targetNodes) == 1 {
				cmd.Println("Node has been permanently removed")
			} else {
				cmd.Println("Nodes have been permanently removed")
			}
		} else {
			if len(targetNodes) == 1 {
				cmd.Println("Node has been moved to the Recycle Bin")
			} else {
				cmd.Println("Nodes have been moved to the Recycle Bin")
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Do not ask for user approval")
	rmCmd.Flags().BoolVarP(&rmPermanently, "permanently", "p", false, "Skip recycle bin and directly permanently delete the target files. Warning: this is not un-doable")
}
