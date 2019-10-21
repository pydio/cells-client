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

	"github.com/pydio/cells-client/rest"
	"github.com/pydio/cells-sdk-go/client/meta_service"
	"github.com/pydio/cells-sdk-go/models"
	"github.com/pydio/cells/common"
)

var (
	lsDetails, lsRaw, lsExists bool
)

var listFiles = &cobra.Command{
	Use:   "ls",
	Short: "List files on pydio cells",
	Long: `List files on Pydio Cells

Use as a normal ls, with additional path to list sub-folders or read info about a node.
You can use the optional -d (--details) flag to display more information, -r (--raw) flag 
to only list found file (& folder) paths or -f (--exists) flag to only check if given path
exists on the server.
Note that you can only use *one* of the three above flags at a time.

# Examples

1/ Listing the content of the personal-files workspace

$ ./cec ls personal-files
+--------+--------------------------+
|  TYPE  |           NAME           |
+--------+--------------------------+
| Folder | personal-files           |
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

$ ./cec ls personal-files/P5021040.jpg -d
Listing: 1 results for personal-files/P5021040.jpg
+------+--------------------------------------+-----------------------------+--------+------------+
| TYPE |                 UUID                 |            NAME             |  SIZE  |  MODIFIED  |
+------+--------------------------------------+-----------------------------+--------+------------+
| File | 98bbd86c-acb9-4b56-a6f3-837609155ba6 | personal-files/P5021040.jpg | 3.1 MB | 5 days ago |
+------+--------------------------------------+-----------------------------+--------+------------+


Will show the metadata for this node (uuid, size, modification date)

3/ Only listing files and folders, one per line.

$ ./cec ls personal-files/P5021040.jpg -r
personal-files/P5021040.jpg

$ ./cec ls personal-files -r
personal-files
Huge Photo-1.jpg
Huge Photo.jpg
IMG_9723.JPG
(...)

4/ Check path existance.

$ ./cec ls personal-files/P5021040.jpg -f
true

$ ./cec ls personal-files/P5021040-not-here -f
false
...


`,
	Run: func(cmd *cobra.Command, args []string) {

		// Check that we do not have multiple flags
		nb := 0
		if lsDetails {
			nb++
		}
		if lsExists {
			nb++
		}
		if lsRaw {
			nb++
		}

		if nb > 1 {
			log.Fatal("please use at most *one* modifier flag")
		}

		//connects to the pydio api via the sdkConfig
		ctx, apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err)
		}

		lsPath := ""
		if len(args) > 0 {
			lsPath = args[0]
		}
		p := strings.Trim(lsPath, "/")

		if lsExists { // check existence and returns
			_, exists := rest.StatNode(p)
			cmd.Println(exists)
			return
		}

		/*
			GetBulkMetaParams contains all the parameters to send to the API endpoint
			for the get bulk meta operation typically these are written to a http.Request
		*/
		params := &meta_service.GetBulkMetaParams{
			Body: &models.RestGetBulkMetaRequest{NodePaths: []string{
				//the workspaces from whom the files are listed
				p, p + "/*",
			}},
			Context: ctx,
		}

		//assigns the files data retrieved above in the results variable
		result, err := apiClient.MetaService.GetBulkMeta(params)
		if err != nil {
			fmt.Printf("could not list files: %s\n", err.Error())
			log.Fatal(err)
		}

		//prints the path therefore the name of the files listed
		if len(result.Payload.Nodes) > 0 {
			if lsRaw {
				for _, node := range result.Payload.Nodes {
					if path.Base(node.Path) == common.PYDIO_SYNC_HIDDEN_FILE_META {
						continue
					}
					// if strings.Trim(node.Path, "/") == p {
					// 	continue
					// }
					cmd.Println(node.Path)
				}
				return
			}

			fmt.Printf("Listing: %d results for %s\n", len(result.Payload.Nodes), p)
			var wsLevel bool
			if len(result.Payload.Nodes) > 1 {
				n0 := result.Payload.Nodes[1]
				if n0.MetaStore != nil {
					_, wsLevel = n0.MetaStore["ws_scope"]
				}
			}
			if !lsDetails {
				fmt.Println("Get more info by adding the -d (details) flag")
			}
			table := tablewriter.NewWriter(os.Stdout)
			if lsDetails {
				if wsLevel {
					table.SetHeader([]string{"Type", "Uuid", "Name", "Label", "Description", "Permissions"})
				} else {
					table.SetHeader([]string{"Type", "Uuid", "Name", "Size", "Modified"})
				}
			} else {
				table.SetHeader([]string{"Type", "Name"})
			}
			for _, node := range result.Payload.Nodes {
				if path.Base(node.Path) == common.PYDIO_SYNC_HIDDEN_FILE_META {
					continue
				}
				t := "File"
				if node.MetaStore != nil && node.MetaStore["ws_scope"] == "\"ROOM\"" {
					t = "Cell"
				} else if node.MetaStore != nil && node.MetaStore["ws_scope"] != "" {
					t = "Workspace"
				} else if node.Type == models.TreeNodeTypeCOLLECTION {
					t = "Folder"
					// The below does not work, we should rather use strings.Trim(node.Path, "/")
					// but then the number of nodes count is false.
					// TODO specify and enhance.
					if node.Path == p {
						continue
					}
				}
				if lsDetails {
					if wsLevel {
						store := node.MetaStore
						fromStore := func(key string) string {
							if v, ok := store[key]; ok {
								return strings.Trim(v, "\"")
							}
							return ""
						}
						table.Append([]string{
							t,
							fromStore("ws_uuid"),
							path.Base(node.Path),
							fromStore("ws_label"),
							fromStore("ws_description"),
							fromStore("ws_permissions"),
						})
					} else {
						table.Append([]string{t, node.UUID, node.Path, sizeToBytes(node.Size), stampToDate(node.MTime)})
					}
				} else {
					table.Append([]string{t, path.Base(node.Path)})
				}
			}
			table.Render()
		}

	},
}

func init() {

	flags := listFiles.PersistentFlags()
	flags.BoolVarP(&lsDetails, "details", "d", false, "Show more information about files")
	flags.BoolVarP(&lsRaw, "raw", "r", false, "List only found paths (one per line) with no further info to be able to use returned results in later commands")
	flags.BoolVarP(&lsExists, "exists", "f", false, "Only check if the passed path exists on the server")

	RootCmd.AddCommand(listFiles)
}

func sizeToBytes(size string) string {
	if size == "" {
		return "-"
	}
	if i, e := strconv.ParseUint(size, 10, 64); e == nil {
		return humanize.Bytes(i)
	} else {
		return "-"
	}
}

func stampToDate(stamp string) string {
	if stamp == "" {
		return "-"
	}
	if i, e := strconv.ParseInt(stamp, 10, 64); e == nil {
		t := time.Unix(i, 0)
		return humanize.Time(t)
	} else {
		return "-"
	}
}
