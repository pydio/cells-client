package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v4/client/meta_service"
	"github.com/pydio/cells-sdk-go/v4/client/user_meta_service"
	"github.com/pydio/cells-sdk-go/v4/models"
)

var (
	metaGetNodePath       string
	metaGetFormat         string // json|table default: table
	metaGetListNamespaces bool
	metaGetNameSpace      string // empty means get all metadata of given node
)

var metaGet = &cobra.Command{
	Use:   "get",
	Short: "Get node's metadata",
	Long: `
DESCRIPTION	

	Get metadata of given node.

EXAMPLE

# Get all usermeta-tag-validation-status meta of node:

$` + os.Args[0] + ` meta get --path=personal/admin/test.txt --namespace=usermeta-tag-validation-status --format=json  

# List available user-meta namespaces

$` + os.Args[0] + ` meta get -all=true
`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: better parameters validation.

		// Connect to the Cells API
		ctx := cmd.Context()
		apiClient := sdkClient.GetApiClient()

		// List all available namespaces
		if metaGetListNamespaces {
			params := &user_meta_service.ListUserMetaNamespaceParams{
				Context: ctx,
			}
			result, _ := apiClient.UserMetaService.ListUserMetaNamespace(params)
			printMetaNamespaces(result.Payload.Namespaces)
			return
		}

		if metaGetNodePath == "" {
			cmd.Printf("Node path is not found")
			return
		}
		p := strings.Trim(metaGetNodePath, "/")

		var exists bool
		if p != "" {
			_, exists = sdkClient.StatNode(ctx, p)
		}

		if !exists && p != "" {
			// Avoid 404 errors
			cmd.Printf("Could not stat node, no folder/file found at %s\n", p)
			return
		}

		// Perform effective listing
		params := &meta_service.GetBulkMetaParams{
			// list folder (p) and its content (p/*) => folder is always first return
			Body: &models.RestGetBulkMetaRequest{
				NodePaths:        []string{p},
				AllMetaProviders: true,
			},
			Context: ctx,
		}

		result, err := apiClient.MetaService.GetBulkMeta(params)
		if err != nil {
			if p == "" {
				cmd.Printf("Could not get metadata for workspaces, cause: %s\n", err.Error())
			} else {
				cmd.Printf("Could not get metadata for files at %s, cause: %s\n", p, err.Error())
			}
			os.Exit(1)
		}
		if len(result.Payload.Nodes) == 0 {
			// Nothing to list: should never happen, we always have at least the current path.
			return
		}

		node := result.Payload.Nodes[0]

		switch metaGetFormat {
		case "json":
			if mv, ok := node.MetaStore[metaGetNodePath]; ok {
				fmt.Printf("{\"%s\": %s}", metaGetNodePath, mv)
				return
			}

			// trim quotes
			// cleaned := make(map[string]string)
			// for k, v := range node.MetaStore {
			// 	if strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") {
			// 		cleaned[k] = strings.Trim(v, "\"")
			// 	}
			// }
			// data, _ := json.MarshalIndent(cleaned, "", "  ")

			data, _ := json.MarshalIndent(node.MetaStore, "", "  ")
			fmt.Printf("%s", data)
			return
		case "table":
			table := tablewriter.NewTable(os.Stdout,
				tablewriter.WithConfig(tablewriter.Config{
					Row: tw.CellConfig{
						Formatting: tw.CellFormatting{AutoWrap: tw.WrapNone},
						Alignment:  tw.CellAlignment{Global: tw.AlignLeft},
					},
				}),
			)
			table.Header([]string{"Meta name", "Value"})
			if metaGetNameSpace != "" {
				if mv, ok := node.MetaStore[metaGetNameSpace]; ok {
					table.Append([]string{metaGetNameSpace, mv})
					table.Render()
					return
				}
			}
			for m, v := range node.MetaStore {
				table.Append([]string{m, v})
			}
			table.Render()
			return
		default:
			cmd.Printf("format must be either json or table\n")
			return
		}
	},
}

func init() {
	flags := metaGet.PersistentFlags()
	flags.StringVarP(&metaGetNodePath, "path", "p", "", "Node's absolute path")
	flags.StringVarP(&metaGetFormat, "format", "f", "table", "Output format json|table")
	flags.BoolVarP(&metaGetListNamespaces, "all", "a", false, "Get available namespaces")
	flags.StringVarP(&metaGetNameSpace, "namespace", "n", "", "Metadata namespace")
	metaCmd.AddCommand(metaGet)
}

func printMetaNamespaces(namespaces []*models.IdmUserMetaNamespace) {
	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithConfig(tablewriter.Config{
			Row: tw.CellConfig{
				Formatting: tw.CellFormatting{AutoWrap: tw.WrapNone},
				Alignment:  tw.CellAlignment{Global: tw.AlignLeft},
			},
		}),
	)
	table.Header([]string{"Namespace", "Label", "JSONDefinition"})
	for _, n := range namespaces {
		table.Append([]string{n.Namespace, n.Label, n.JSONDefinition})
	}
	table.Render()
}
