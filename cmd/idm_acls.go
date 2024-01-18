package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/olekukonko/tablewriter"
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
	Short: "List acls by node UUID",
	Long: `
DESCRIPTION	

	List ACLs attached to one or more given nodes, and optionally delete them afterward. 
	Can be handy for debugging purposes.

EXAMPLE

# Given this listing in "test" workspace:

$` + os.Args[0] + ` ls -d test  
Found 4 nodes at test:  
+--------+--------------------------------------+-----------+--------+-------------+  
|  TYPE  |                 UUID                 |   NAME    |  SIZE  |  MODIFIED   |  
+--------+--------------------------------------+-----------+--------+-------------+  
| Folder | 8ec79c1e-2464-40d0-a762-c36d8a9e5886 | .         | 2.6 MB | 2 years ago |  
| File   | 1c989848-5eff-49cf-8727-4db754e02c25 | buro4.jpg | 541 kB | 2 years ago |  
| File   | d796d7c5-dce9-4994-bca3-3cf03c27c39d | büro1.jpg | 1.0 MB | 2 years ago |  
| File   | 15f09f59-9171-4e25-809a-488e475dafa4 | büro2.jpg | 996 kB | 2 years ago |  
+--------+--------------------------------------+-----------+--------+-------------+  

# List ACLs for file "buro4.txt":  
` + os.Args[0] + ` idm list-acls --uuid 1c989848-5eff-49cf-8727-4db754e02c25
	
# List ACLs for multiple files:  
` + os.Args[0] + ` idm list-acls -n 1c989848-5eff-49cf-8727-4db754e02c25 -n d796d7c5-dce9-4994-bca3-3cf03c27c39d
  
# Delete all ACLs on a given node  
` + os.Args[0] + ` idm list-acls -n 1c989848-5eff-49cf-8727-4db754e02c25 --delete

`,
	Run: func(cm *cobra.Command, args []string) {

		ctx, apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err)
		}

		if len(listAclsByNodeIds) == 0 {
			log.Fatal("Cannot list ACLS. Please precise *at least* one node UUID")
		}

		params := &acl_service.SearchAclsParams{
			Body: &models.RestSearchACLRequest{
				Queries: []*models.IdmACLSingleQuery{{
					NodeIDs: listAclsByNodeIds,
				}},
			},
			Context: ctx,
		}

		result, err := apiClient.ACLService.SearchAcls(params)
		if err != nil {
			fmt.Printf("could not list acls: %s\n", err.Error())
			log.Fatal(err)
		}

		if len(result.Payload.ACLs) > 0 {

			fmt.Printf("Found %d ACLs:\n", len(result.Payload.ACLs))

			for _, u := range result.Payload.ACLs {
				fmt.Println("  - " + u.NodeID + " | " + u.RoleID + " | " + u.Action.Name + " | " + u.WorkspaceID)
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"UUID", "Role ID", "Action Name", "WS ID"})
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetAutoWrapText(false)

			for _, u := range result.Payload.ACLs {
				table.Append([]string{u.NodeID, u.RoleID, u.Action.Name, u.WorkspaceID})
			}
			table.Render()

		}

		if listAclsDeleteResult {
			pr := promptui.Prompt{Label: "Do you want to delete all these ACLs ?", IsConfirm: true}
			if _, e := pr.Run(); e != nil {
				fmt.Println("Aborting operation...")
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
	listAcls.Flags().StringSliceVarP(&listAclsByNodeIds, "uuid", "n", []string{}, "Search by node UUID, can be used multiple times")
	listAcls.Flags().BoolVarP(&listAclsDeleteResult, "delete", "", false, "Delete all found ACLs")
	idmCmd.AddCommand(listAcls)
}
