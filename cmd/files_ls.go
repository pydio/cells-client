package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v4/client/meta_service"
	"github.com/pydio/cells-sdk-go/v4/models"
)

var lsCmdExample = ` 1/ Listing the content of a folder
  
  $ ` + os.Args[0] + ` ls common-files/Test
  Found 6 nodes at common-files/Test:
  +--------+-----------------+
  |  TYPE  |      NAME       |
  +--------+-----------------+
  | Folder | .               |
  | Folder | Archives        |
  | File   | Garden.jpeg     |
  | File   | Nighthawks.jpeg |
  | File   | ReadMe.md       |
  | File   | Summer.jpeg     |
  +--------+-----------------+

  
 2/ Showing details about a file
  
  $ ` + os.Args[0] + ` ls -d common-files/Test/Garden.jpeg
  Found 1 node at common-files/Test/Garden.jpeg:
  +------+--------------------------------------+-------------+---------+----------------+----------------------------------+
  | TYPE |                 UUID                 |    NAME     |  SIZE   |    MODIFIED    |          INTERNAL HASH           |
  +------+--------------------------------------+-------------+---------+----------------+----------------------------------+
  | File | e50c9d8a-a84c-4b32-908a-408927657810 | Garden.jpeg | 442 KiB | 52 minutes ago | a6676657eb373c7f3e3c4e01be817fac |
  +------+--------------------------------------+-------------+---------+----------------+----------------------------------+
 
  Will show the metadata for this node (uuid, size, modification date and internal hash)
  
 3/ Only listing files and folders, one per line.
  
  $ ` + os.Args[0] + ` ls -r common-files/Test
  common-files/Test/Archives/
  common-files/Test/Garden.jpeg
  common-files/Test/Nighthawks.jpeg
  common-files/Test/ReadMe.md
  ...
  
 4/ Using a template:

  $ ` + os.Args[0] + ` ls personal-files --format '"{{.Name}}";"{{.Type}}";"{{.Path}}";"{{.HumanSize}}";"{{.Date}}"' personal-files/
  "Cat.jpg";"File";"personal-files/Cat.jpg";"1.3 MB";"2 months ago"
  "Others";"Folder";"personal-files/Others";"12 MB";"11 minutes ago"
  "Photo.png";"File";"personal-files/Photo.png";"8.1 MB";"3 minutes ago"
  ...
  
 5/ Check path existence.
  
  $ ` + os.Args[0] + ` ls personal-files/info.txt -f
  true
  
  $ ` + os.Args[0] + ` ls personal-files/not-here -f
  false
`

// List the various modes that have been implemented
const (
	exists      = "EXISTS"
	raw         = "RAW"
	defaultList = "DEFAULT"
	details     = "DETAILS"
	goTemplate  = "TEMPLATE"
)

// Known node meta data
const (
	metaType      = "Type"
	metaUuid      = "Uuid"
	metaName      = "Name"
	metaHash      = "Hash"
	metaPath      = "Path"
	metaHumanSize = "HumanSize"
	metaSizeBytes = "SizeBytes"
	metaTimestamp = "TimeStamp"
	medaDate      = "Date"
)

// Store options
var (
	lsDetails bool
	lsRaw     bool
	lsExists  bool
	lsFormat  string

	parsedTemplate *template.Template
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
   - format flag with a valid go template to get a custom listing.

  Note that you can only use *one* of the above flags at a time.

  As reference, known attributes for the Go templates are:
   - Type: File, Folder or Workspace
   - Uuid: the unique ID of the corresponding node in the Cells Server
   - Hash: in case of a file, the internal hash computed by the server 
   - Name: name of the item
   - Path: the path from the root of the server
   - HumanSize: a human-friendly formatted size
   - SizeBytes: the size of the object in bytes 
   - TimeStamp: number of seconds since 1970 when the item was last modified 
   - Date: a human-friendly date for the last modification

Note that the last 4 meta-data are only indicative for folders: they might be out of date, if the listing happens shortly after a modification in the sub-tree.

EXAMPLES

` + lsCmdExample + `
`,
	Run: func(cmd *cobra.Command, args []string) {

		// Retrieve requested display type and check it is valid
		displayMode := sanityCheck()

		// Retrieve and pre-process path if defined
		lsPath := ""
		if len(args) > 0 {
			lsPath = args[0]
		}
		p := strings.Trim(lsPath, "/")

		// Connect to the Cells API
		ctx := cmd.Context()
		apiClient := sdkClient.GetApiClient()
		var exists bool
		if p != "" {
			_, exists = sdkClient.StatNode(ctx, p)
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
			if p == "" {
				cmd.Printf("Could not list workspaces, cause: %s\n", err.Error())
			} else {
				cmd.Printf("Could not list files at %s, cause: %s\n", p, err.Error())
			}
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

		table := tablewriter.NewTable(os.Stdout,
			tablewriter.WithConfig(tablewriter.Config{
				Row: tw.CellConfig{
					Formatting: tw.CellFormatting{AutoWrap: tw.WrapNone},
					Alignment:  tw.CellAlignment{Global: tw.AlignLeft},
				},
			}),
		)

		// table := tablewriter.NewWriter(os.Stdout)

		hiddenRowNb := 0
		// Process the results
		for i, node := range result.Payload.Nodes {

			currPath := node.Path
			currName := path.Base(currPath)

			// Useless, hidden folders are not returned anyway
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
				} else if (displayMode == raw || displayMode == goTemplate) && (t == "Folder" || t == "Workspace") {
					// We do not want to list parent folder or workspace in simple lists
					hiddenRowNb++
					continue
				} else if node.Type != nil && *node.Type == models.TreeNodeTypeCOLLECTION {
					// replace path by "." notation
					currName = "."
				}
			}

			iHash := ""
			if t == "File" {
				// Retrieve the internal hash
				if node.MetaStore != nil {
					if v, ok := node.MetaStore["x-cells-hash"]; ok {
						iHash = strings.Trim(v, "\"")
					}
				}
			}

			switch displayMode {
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
					table.Append([]string{t, node.UUID, currName, sizeToHuman(node.Size), stampToDate(node.MTime), iHash})
				}
			case raw:
				if node.Type != nil && *node.Type == models.TreeNodeTypeCOLLECTION {
					out := currPath + "/"
					_, _ = fmt.Fprintln(os.Stdout, out)
				} else {
					_, _ = fmt.Fprintln(os.Stdout, node.Path)
				}
			case goTemplate:

				values := map[string]string{
					metaType:      t,
					metaUuid:      node.UUID,
					metaName:      currName,
					metaPath:      node.Path,
					metaHumanSize: sizeToHuman(node.Size),
					metaSizeBytes: node.Size,
					metaTimestamp: node.MTime,
					medaDate:      stampToDate(node.MTime),
					metaHash:      iHash,
				}

				if err = parsedTemplate.Execute(os.Stdout, values); err != nil {
					log.Fatalln("could not execute template", err)
				}
				fmt.Println("") // explicit carriage return

			default:
				table.Append([]string{t, currName})
			}
		}

		// Add meta-info and table headers and render (if necessary)

		rowNb := len(result.Payload.Nodes) - hiddenRowNb
		legend := fmt.Sprintf("Found %d nodes at %s:", rowNb, p)
		if p == "" { // root of the server
			legend = fmt.Sprintf("Listing %d workspaces:", rowNb)
		}
		switch displayMode {
		case details:
			fmt.Println(legend)
			if wsLevel {
				table.Header([]string{"Type", "Uuid", "Name", "Label", "Description", "Permissions"})
			} else {
				table.Header([]string{"Type", "Uuid", "Name", "Size", "Modified", "Internal Hash"})
			}
			table.Render()
		case raw, goTemplate: // Nothing to add: we just want the raw values that we already displayed while looping
			return
		default:
			fmt.Println(legend)
			table.Header([]string{"Type", "Name"})
			table.Render()
			fmt.Println("Get more info by adding the -d (details) flag")
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
	if lsFormat != "" {
		nb++
		displayType = goTemplate
		// for go templates, also validate the passed template
		tmpl, err := template.New("lsNode").Parse(lsFormat)
		if err != nil {
			log.Fatalln("failed to parse template:", err)
		}
		parsedTemplate = tmpl
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

func sizeToHuman(size string) string {
	if size == "" {
		return "-"
	}
	if i, e := strconv.ParseUint(size, 10, 64); e == nil {
		return humanize.IBytes(i)
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
	flags.StringVar(&lsFormat, "format", "", "Use go template to format each line of the output listing")

	RootCmd.AddCommand(listFiles)
}
