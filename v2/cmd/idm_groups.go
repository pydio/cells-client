package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v3/client/user_service"

	"github.com/pydio/cells-client/v2/rest"
)

var listGroups = &cobra.Command{
	Use:   "list-groups",
	Short: "List groups",
	Long: `
DESCRIPION

  List user groups that are defined in your Pydio Cells instance.
`,
	Run: func(cmd *cobra.Command, args []string) {

		ctx, apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err)
		}

		params := &user_service.SearchUsersParams{
			//Body:       &models.RestSearchUserRequest{},
			Context: ctx,
		}

		result, err := apiClient.UserService.SearchUsers(params)
		if err != nil {
			fmt.Printf("could not list groups %s\n", err.Error())
			log.Fatal(err)
		}

		if len(result.Payload.Groups) > 0 {
			fmt.Printf("Found %d groups\n", len(result.Payload.Groups))
			for _, u := range result.Payload.Groups {
				fmt.Println("  - " + u.GroupLabel)
			}
		}
	},
}

func init() {
	idmCmd.AddCommand(listGroups)
}
