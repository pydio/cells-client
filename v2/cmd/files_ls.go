package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v3/client/meta_service"
	"github.com/pydio/cells-sdk-go/v3/models"

	"github.com/pydio/cells-client/v2/rest"
)

var lsCmdExample = `
1/ Listing the content of the personal-files workspace

$ ` + os.Args[0] + ` ls personal-files
+--------+--------------------------+
|  TYPE  |           NAME           |
+--------+--------------------------+
| Folder | .			            |
| File   | Huge Photo-1.jpg         |
| File   | Huge Photo.jpg           |
| File   | IMG_9723.JPG             |
| File   | P5021040.jpg             |
| Folder | UPLOAD                   |
| File   | anothercopy              |
| File   | cec22                    |
| Folder | recycle_bin              |
| File   | test_crud-1545206681.txt |
| File   | test_crud-1545206846.txt |
| File   | test_file2.txt           |
+--------+--------------------------+

2/ Showing details about a file

$ ` + os.Args[0] + ` ls personal-files/P5021040.jpg -d
Listing: 1 results for personal-files/P5021040.jpg
+------+--------------------------------------+-----------------------------+--------+------------+
| TYPE |                 UUID                 |            NAME             |  SIZE  |  MODIFIED  |
+------+--------------------------------------+-----------------------------+--------+------------+
| File | 98bbd86c-acb9-4b56-a6f3-837609155ba6 | personal-files/P5021040.jpg | 3.1 MB | 5 days ago |
+------+--------------------------------------+-----------------------------+--------+------------+


Will show the metadata for this node (uuid, size, modification date)

3/ Only listing files and folders, one per line.

$ ` + os.Args[0] + ` ls personal-files/P5021040.jpg -r
personal-files/P5021040.jpg

$ ` + os.Args[0] + ` ls personal-files -r
Huge Photo-1.jpg
Huge Photo.jpg
IMG_9723.JPG
(...)

4/ Check path existence.

$ ` + os.Args[0] + ` ls personal-files/P5021040.jpg -f
true

$ ` + os.Args[0] + ` ls personal-files/P5021040-not-here -f
false
...
`

const (
	exists      = "EXISTS"
	raw         = "RAW"
	defaultList = "DEFAULT"
	details     = "DETAILS"
)

var (
	lsDetails bool
	lsRaw     bool
	lsExists  bool
)

var listFiles = &cobra.Command{
	Use:   "ls",
	Short: "List files in your Cells server",
	Long: `
DESCRIPTION

  List files in your Cells server.

SYNTAX

  Use as a normal ls, with additional path to list sub-folders or read info about a node.
  You can use one of the below optional flags: 
   - d (--details) flag to display more information, 
   - r (--raw) flag to only list the paths of found files and folders
   - f (--exists) flag to only check if given path exists on the server.

  Note that you can only use *one* of the three above flags at a time.

EXAMPLES

` + lsCmdExample + `
`,
	Run: func(cmd *cobra.Command, args []string) {

		// Retrieve requested display type and check it is valid
		dt := sanityCheck()

		// Retrieve and pre-process path if defined
		lsPath := ""
		if len(args) > 0 {
			lsPath = args[0]
		}
		p := strings.Trim(lsPath, "/")

		// Connect to the Cells API
		ctx, apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err)
		}

		var exists bool
		if p != "" {
			_, exists = rest.StatNode(p)
		}

		if lsExists {
			// Only check existence and return
			cmd.Println(exists)
			return
		} else if !exists && p != "" {
			// Avoid 404 errors
			cmd.Printf("Could not list content, no folder found at %s\n", p)
			return
		}

		// Perform effective listing
		params := &meta_service.GetBulkMetaParams{
			// list folder (p) and its content (p/*) => folder is always first return
			Body: &models.RestGetBulkMetaRequest{NodePaths: []string{
				p, p + "/*",
			}},
			Context: ctx,
		}
		result, err := apiClient.MetaService.GetBulkMeta(params)
		if err != nil {
			cmd.Printf("Could not list files at %s, cause: %s\n", p, err.Error())
			os.Exit(1)
		}
		if len(result.Payload.Nodes) == 0 {
			// Nothing to list: should never happen, we always have at least the current path.
			return
		}

		// Not very elegant way to check if we are at the workspace level
		var wsLevel bool
		if len(result.Payload.Nodes) > 1 {
			firstChild := result.Payload.Nodes[1]
			if firstChild.MetaStore != nil {
				_, wsLevel = firstChild.MetaStore["ws_scope"]
			}
		}

		table := tablewriter.NewWriter(os.Stdout)

		hiddenRowNb := 0
		// Process the results
		for i, node := range result.Payload.Nodes {

			currPath := node.Path
			currName := path.Base(currPath)

			// Useless, hidden foldersare not returned anyway
			// // First, filter out unwanted nodes
			// if currName == common.PYDIO_SYNC_HIDDEN_FILE_META {
			// 	continue
			// }

			t := "File"
			if node.MetaStore != nil && node.MetaStore["ws_scope"] == "\"ROOM\"" {
				t = "Cell"
			} else if node.MetaStore != nil && node.MetaStore["ws_scope"] != "" {
				t = "Workspace"
			} else if node.Type != nil && *node.Type == models.TreeNodeTypeCOLLECTION {
				t = "Folder"
			}

			// Corner case of the 1st result
			if i == 0 {
				// Do not list root of the repo
				if currPath == "" && wsLevel {
					hiddenRowNb++
					continue // processingLoop
				} else if lsRaw && (t == "Folder" || t == "Workspace") {
					// We do not want to list parent folder or workspace in simple lists
					hiddenRowNb++
					continue
				} else if node.Type != nil && *node.Type == models.TreeNodeTypeCOLLECTION {
					// replace path by "." notation
					currName = "."
				}
			}

			switch dt {
			case details:
				if wsLevel {
					table.Append([]string{
						t,
						fromMetaStore(node, "ws_uuid"),
						currName,
						fromMetaStore(node, "ws_label"),
						fromMetaStore(node, "ws_description"),
						fromMetaStore(node, "ws_permissions"),
					})
				} else {
					table.Append([]string{t, node.UUID, currName, sizeToBytes(node.Size), stampToDate(node.MTime)})
				}
			case raw:
				if node.Type != nil && *node.Type == models.TreeNodeTypeCOLLECTION {
					out := currPath + "/"
					_, _ = fmt.Fprintln(os.Stdout, out)
				} else {
					_, _ = fmt.Fprintln(os.Stdout, node.Path)
				}
			default:
				table.Append([]string{t, currName})
			}
		}

		// Add meta-info and table headers and render (if necessary)
		rowNb := len(result.Payload.Nodes) - hiddenRowNb
		legend := fmt.Sprintf("Listing: %d results for %s", rowNb, p)
		if p == "" { // root of the server
			legend = fmt.Sprintf("Listing %d workspaces", rowNb)
		}
		switch dt {
		case details:
			fmt.Println(legend)
			if wsLevel {
				table.SetHeader([]string{"Type", "Uuid", "Name", "Label", "Description", "Permissions"})
			} else {
				table.SetHeader([]string{"Type", "Uuid", "Name", "Size", "Modified"})
			}
			table.Render()
		case raw: // Nothing to add: we just want the raw values that we already displayed while looping
			break
		default:
			fmt.Println(legend)
			fmt.Println("Get more info by adding the -d (details) flag")
			table.SetHeader([]string{"Type", "Name"})
			table.Render()
		}
	},
}

func sanityCheck() string {
	// Check that we do not have multiple flags
	displayType := defaultList
	nb := 0
	if lsDetails {
		nb++
		displayType = details
	}
	if lsExists {
		nb++
		displayType = exists
	}
	if lsRaw {
		nb++
		displayType = raw
	}
	if nb > 1 {
		log.Fatal("Please use at most *one* modifier flag")
	}
	return displayType
}

func fromMetaStore(node *models.TreeNode, key string) string {
	if v, ok := node.MetaStore[key]; ok {
		return strings.Trim(v, "\"")
	}
	return ""
}

func sizeToBytes(size string) string {
	if size == "" {
		return "-"
	}
	if i, e := strconv.ParseUint(size, 10, 64); e == nil {
		return humanize.Bytes(i)
	}
	return "-"
}

func stampToDate(stamp string) string {
	if stamp == "" {
		return "-"
	}
	if i, e := strconv.ParseInt(stamp, 10, 64); e == nil {
		t := time.Unix(i, 0)
		return humanize.Time(t)
	}
	return "-"
}

func init() {
	flags := listFiles.PersistentFlags()
	flags.BoolVarP(&lsDetails, "details", "d", false, "Show more information about retrieved objects")
	flags.BoolVarP(&lsRaw, "raw", "r", false, "List found paths (one per line) with no further info to be able to use returned results in later commands")
	flags.BoolVarP(&lsExists, "exists", "f", false, "Check if the passed path exists on the server and return non zero status code if not")

	RootCmd.AddCommand(listFiles)
}
