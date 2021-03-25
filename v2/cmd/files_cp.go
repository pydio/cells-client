package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v2/models"

	"github.com/pydio/cells-client/v2/rest"
)

var cpCmd = &cobra.Command{
	Use:   "cp",
	Short: "Copy files from A to B within your remote server",
	Long: `
DESCRIPTION

  Copy files from one location to another *within* a *single* Pydio Cells instance. 
  To copy files from the client machine to your server (and vice versa), rather see the '` + os.Args[0] + ` scp' command.

WILD-CHARS

  In version 2.1, we only support the '*' wild char as a standalone token of the source path, that is:
    - '` + os.Args[0] + ` cp common-files/test/* common-files/target' is legit and will copy 
	  all files and folder found in test folder on your server to the target folder
	- '` + os.Args[0] + ` cp common-files/test/*.jpg ...' or '` + os.Args[0] + ` cp common-files/test/ima* ...'  
	  will *not* work as some might expect: the system looks for a single file named respectively '*.jpg' or 'ima*'
	  and will probably fail to find it (unless if a file with this name exists on your server...)

EXAMPLE

  # Copy file "test.txt" from workspace root inside target "folder-a":
  ` + os.Args[0] + ` cp common-files/test.txt common-files/folder-a

  # Copy a file from a workspace to another:
  ` + os.Args[0] + ` cp common-files/test.txt personal-files/folder-b

  # Copy the full content of a folder inside another
  ` + os.Args[0] + ` cp common-files/test/* common-files/folder-c
`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fromPath := args[0]
		toPath := args[1]
		targetParent := true

		// Pre-process source path
		var sourceNodes []string
		if path.Base(fromPath) == "*" {
			nodes, err := rest.ListNodesPath(fromPath)
			if err != nil {
				log.Fatalf("Preparing grouped copy, could not list all nodes under %s, cause: %s", path.Dir(fromPath), err.Error())
			}
			sourceNodes = nodes
		} else if strings.HasSuffix(path.Base(fromPath), "*") {
			fmt.Println(promptui.IconWarn + " We currently only support the '*' wild char without prefix, see help for further details")
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
				log.Fatalf("A file already exists at %s. \nThe cells-client does not yet handle this case. If you want to overwrite, first delete the existing target file.", toPath)
			}
		} else { // We assume we have been given full path including target file name
			parPath, _ := path.Split(toPath)
			if parPath == "" {
				log.Fatalf("Target location %s does not exist on server, double check your parameters.", toPath)
			}
			targetNode, targetExists := rest.StatNode(parPath)
			if !targetExists {
				log.Fatalf("Parent target location %s does not exist on server, double check your parameters.", parPath)
			} else if targetNode.Type != models.TreeNodeTypeCOLLECTION {
				log.Fatalf("Parent target location %s exists on server but is not a folder. It cannot be used as a copy target location.", parPath)
			}
			// parent exists and is a folder => we assume we have been passed a full target path including target file name.
			targetParent = false
		}

		// Prepare and launch effective copy
		params := rest.BuildParams(sourceNodes, toPath, targetParent)
		jobID, err := rest.CopyJob(params)
		if err != nil {
			log.Fatalln("could not run job:", err.Error())
		}

		err = rest.MonitorJob(jobID)
		if err != nil {
			log.Fatalln("could not monitor job", err.Error())
		}
	},
}

func init() {
	RootCmd.AddCommand(cpCmd)
}
