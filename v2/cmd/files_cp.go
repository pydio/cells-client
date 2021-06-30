package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v2/models"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

func cpDescription(bin string) string {
	return `
DESCRIPTION

  Copy files from one location to another *within* a *single* Pydio Cells instance. 
  To copy files from the client machine to your server (and vice versa), rather see the '` + bin + ` scp' command.

WILD-CHARS

  In version 2.1, we only support the '*' wild char as a standalone token of the source path, that is:
    - '` + bin + ` cp common-files/test/* common-files/target' is legit and will copy 
	  all files and folder found in test folder on your server to the target folder
	- '` + bin + ` cp common-files/test/*.jpg ...' or '` + bin + ` cp common-files/test/ima* ...'  
	  will *not* work as some might expect: the system looks for a single file named respectively '*.jpg' or 'ima*'
	  and will probably fail to find it (unless if a file with this name exists on your server...)

EXAMPLE

  # Copy file "test.txt" from workspace root inside target "folder-a":
  ` + bin + ` cp common-files/test.txt common-files/folder-a

  # Copy a file from a workspace to another:
  ` + bin + ` cp common-files/test.txt personal-files/folder-b

  # Copy the full content of a folder inside another
  ` + bin + ` cp common-files/test/* common-files/folder-c
`
}

var cpCmd = &cobra.Command{
	Use:   "cp",
	Short: "Copy files from A to B within your remote server",
	Long:  cpDescription(os.Args[0]),
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fromPath := args[0]
		toPath := args[1]
		targetParent := true

		spinner, err := common.NewSpinner().Start(fmt.Sprintf("Copying %s to %s", fromPath, toPath))
		if err != nil {
			cmd.PrintErrf("spinner failed %s", err)
			os.Exit(1)
		}
		defer spinner.Stop()

		if quiet {
			common.DisableSpinnerOutput()
		}

		// Pre-process source path
		var sourceNodes []string
		if path.Base(fromPath) == "*" {
			nodes, err := rest.ListNodesPath(fromPath)
			if err != nil {
				spinner.Fail(fmt.Sprintf("Preparing grouped copy, could not list all nodes under %s, cause: %s", path.Dir(fromPath), err.Error()))
				os.Exit(1)
			}
			sourceNodes = nodes
		} else if strings.HasSuffix(path.Base(fromPath), "*") {
			spinner.Warning("We currently only support the '*' wild char without prefix, see help for further details")
			sourceNodes = []string{fromPath}
		} else {
			sourceNodes = []string{fromPath}
		}

		// Pre-process target path
		targetNode, targetExists := rest.StatNode(toPath)
		if targetExists {
			if targetNode.Type == models.TreeNodeTypeCOLLECTION {
				// target is a folder as expected nothing to do
			} else {
				// Target is an existing file, we throw an error for the time being
				spinner.Fail(fmt.Sprintf("A file already exists at %s. \nThe cells-client does not yet handle this case. If you want to overwrite, first delete the existing target file.", toPath))
				os.Exit(1)
			}
		} else { // We assume we have been given full path including target file name
			parPath, _ := path.Split(toPath)
			if parPath == "" {
				spinner.Fail(fmt.Sprintf("Target location %s does not exist on server, double check your parameters.", toPath))
				os.Exit(1)
			}
			targetNode, targetExists := rest.StatNode(parPath)
			if !targetExists {
				log.Fatalf("Parent target location %s does not exist on server, double check your parameters.", parPath)
			} else if targetNode.Type != models.TreeNodeTypeCOLLECTION {
				spinner.Fail(fmt.Sprintf("Parent target location %s exists on server but is not a folder. It cannot be used as a copy target location.", parPath))
				os.Exit(1)
			}
			// parent exists and is a folder => we assume we have been passed a full target path including target file name.
			targetParent = false
		}

		// Prepare and launch effective copy
		params := rest.BuildParams(sourceNodes, toPath, targetParent)
		jobID, err := rest.CopyJob(params)
		if err != nil {
			spinner.Fail(fmt.Sprintf("could not run job: %s", err.Error()))
			os.Exit(1)
		}

		err = rest.MonitorJob(jobID)
		if err != nil {
			spinner.Fail(fmt.Sprintf("could not monitor job: %s", err.Error()))
			os.Exit(1)
		}
		spinner.Success(fmt.Sprintf("Copy from %s to %s finished", fromPath, toPath))
	},
}

func init() {
	RootCmd.AddCommand(cpCmd)
}
