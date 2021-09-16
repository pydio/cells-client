package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v3/client/role_service"
	"github.com/pydio/cells-sdk-go/v3/models"

	"github.com/pydio/cells-client/v2/rest"
)

var listRoles = &cobra.Command{
	Use:   "list-roles",
	Short: "List roles",
	Long: `
DESCRIPTION
	
  List the roles defined in your Pydio Cells instance, including technical roles 
  that are implicitely created upon user or group creation.
`,
	Run: func(cm *cobra.Command, args []string) {

		ctx, apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err)
		}

		params := &role_service.SearchRolesParams{
			Body:    &models.RestSearchRoleRequest{},
			Context: ctx,
		}

		result, err := apiClient.RoleService.SearchRoles(params)
		if err != nil {
			fmt.Printf("could not list roles: %s\n", err.Error())
			log.Fatal(err)
		}

		if len(result.Payload.Roles) > 0 {
			fmt.Printf("Found %d roles\n", len(result.Payload.Roles))
			for _, u := range result.Payload.Roles {
				//fmt.Println("  -- " + u.Label)
				if u.GroupRole == true {
					fmt.Printf(" -- %s __ GROUP ROLE \n", u.Label)
				} else if u.UserRole == true {
					fmt.Printf(" -- %s __ USER ROLE \n", u.Label)
				} else {
					fmt.Println(" -- " + u.Label)

				}
			}
		}
	},
}

func init() {
	idmCmd.AddCommand(listRoles)
}
