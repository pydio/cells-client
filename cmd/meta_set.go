package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v4/client"
	"github.com/pydio/cells-sdk-go/v4/client/user_meta_service"
	"github.com/pydio/cells-sdk-go/v4/models"
)

var (
	metaSetNodePath  string
	metaSetOperation string
	metaSetNamespace string
	metaSetJsonValue string
)

var metaSet = &cobra.Command{
	Use:   "set",
	Short: "Set specific metadata for node",
	Long: `
DESCRIPTION	

	Update or Delete metadata for given node.

EXAMPLE

# Update usermeta-tag-validation-status meta of node:

$` + os.Args[0] + ` meta set --path=personal/admin/test.txt --operation=update --meta-name=usermeta-tag-validation-status --value=Validated

`,
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: better parameters validation.

		if metaSetNodePath == "" {
			cmd.Printf("Node path is not found")
			return
		}
		p := strings.Trim(metaSetNodePath, "/")

		// Connect to the Cells API
		ctx := cmd.Context()
		apiClient := sdkClient.GetApiClient()

		// Check node's existence
		var node *models.TreeNode
		var exists bool
		if p != "" {
			node, exists = sdkClient.StatNode(ctx, p)
		}

		if !exists && node == nil {
			// Avoid 404 errors
			cmd.Printf("Could not stat node, no folder/file found at %s\n", p)
			return
		}

		// Validate namespace
		if metaSetNamespace == "" {
			cmd.Printf("namespace is not defined")
			return
		}
		err := validateMetaNamespace(ctx, apiClient, metaSetNamespace)
		if err != nil {
			cmd.PrintErr(err)
		}

		// Check operation	metaSetOperation
		var operation models.UpdateUserMetaRequestUserMetaOp
		value := metaSetJsonValue
		operation = models.UpdateUserMetaRequestUserMetaOpPUT

		switch metaSetOperation {
		case "update":
			// Validate value
			if metaSetJsonValue == "" {
				cmd.Printf("meta value is not defined")
				return
			}
		case "delete":
			value = "\"\""
		default:
			cmd.Printf("Operation parameter is required. Please provide either \"update\" or \"delete\"\n")
			return
		}

		// Perform effective operation
		params := &user_meta_service.UpdateUserMetaParams{
			Body: &models.IdmUpdateUserMetaRequest{
				MetaDatas: []*models.IdmUserMeta{
					{
						Namespace: metaSetNamespace,
						NodeUUID:  node.UUID,
						JSONValue: value,
					},
				},
				// Use PUT with empty value for DELETE
				Operation: &operation,
			},
			Context: ctx,
		}
		_, err = apiClient.UserMetaService.UpdateUserMeta(params)

		if err != nil {
			cmd.PrintErr(err)
		}
	},
}

func init() {
	flags := metaSet.PersistentFlags()
	flags.StringVarP(&metaSetNodePath, "path", "p", "", "Absolute path of node")
	flags.StringVarP(&metaSetOperation, "operation", "o", "", "Operation name: update|delete")
	flags.StringVarP(&metaSetNamespace, "namespace", "n", "", "Metadata namespace")
	flags.StringVarP(&metaSetJsonValue, "value", "v", "", "JSON-formated metadata value")
	metaCmd.AddCommand(metaSet)
}

func validateMetaNamespace(ctx context.Context, apiClient *client.PydioCellsRestAPI, namespace string) error {
	params := &user_meta_service.ListUserMetaNamespaceParams{
		Context: ctx,
	}
	result, err := apiClient.UserMetaService.ListUserMetaNamespace(params)

	if err != nil {
		return err
	}

	for _, n := range result.Payload.Namespaces {
		if n.Namespace == namespace {
			return nil
		}
	}
	return fmt.Errorf("invalid namespace %s", namespace)
}
