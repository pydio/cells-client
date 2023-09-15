package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v4/client/acl_service"
	"github.com/pydio/cells-sdk-go/v4/models"

	"github.com/pydio/cells-client/v4/rest"
)

var (
	listAclsByNodeIds    []string
	listAclsDeleteResult bool
)

var listAcls = &cobra.Command{
	Use:   "list-acls",
	Short: "List acls by node Uuid",
	Long: `
DESCRIPTION	

  List all workspaces on which the current logged in user has *at least* Read Access.

`,
	Run: func(cm *cobra.Command, args []string) {

		ctx, apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err)
		}

		//retrieves the users using the searchWorkspacesParams function
		params := &acl_service.SearchAclsParams{
			Body: &models.RestSearchACLRequest{
				Queries: []*models.IdmACLSingleQuery{{
					NodeIDs: listAclsByNodeIds,
				}},
			},
			Context: ctx,
		}

		//assigns the workspaces data retrieved above in the results variable
		result, err := apiClient.ACLService.SearchAcls(params)
		if err != nil {
			fmt.Printf("could not list acls: %s\n", err.Error())
			log.Fatal(err)
		}

		//prints the workspace label
		if len(result.Payload.ACLs) > 0 {
			fmt.Printf("* %d ACLs found\n", len(result.Payload.ACLs))
			for _, u := range result.Payload.ACLs {
				fmt.Println("  - " + u.NodeID + " | " + u.RoleID + " | " + u.Action.Name + " | " + u.WorkspaceID)
			}
		}

		if listAclsDeleteResult {
			pr := promptui.Prompt{Label: "Do you want to delete all these ACLs ?", IsConfirm: true}
			if _, e := pr.Run(); e != nil {
				fmt.Println("Aborting operation")
				return
			}
			for _, u := range result.Payload.ACLs {
				_, er := apiClient.ACLService.DeleteACL(&acl_service.DeleteACLParams{
					Body:    u,
					Context: ctx,
				})
				if er != nil {
					log.Fatal("Could not delete ACL", u.ID, ":", er.Error())
				} else {
					fmt.Println(" - Removed ACL " + u.ID)
					<-time.After(100 * time.Millisecond)
				}
			}
		}

	},
}

func init() {
	listAcls.Flags().StringSliceVarP(&listAclsByNodeIds, "node-id", "n", []string{}, "Search by node ID")
	listAcls.Flags().BoolVarP(&listAclsDeleteResult, "delete", "", false, "Delete all found ACLs")
	idmCmd.AddCommand(listAcls)
}
