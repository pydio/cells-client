package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v3/client/user_service"

	"github.com/pydio/cells-client/v2/rest"
)

var listUsers = &cobra.Command{
	Use:   "list-users",
	Short: "List users",
	Long: `
DESCRIPTION	

  List the users defined in your Pydio Cells instance.
`,
	Run: func(cm *cobra.Command, args []string) {

		ctx, apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err)
		}

		// query := api.RestSearchUserRequest{}
		params := &user_service.SearchUsersParams{
			Context: ctx,
		}

		//assigns the users data retrieved above in the results variable
		result, err := apiClient.UserService.SearchUsers(params)
		if err != nil {
			fmt.Printf("could not list users: %s\n", err.Error())
			log.Fatal(err)
		}

		//prints the login of the users retrieved previously
		if len(result.Payload.Users) > 0 {
			fmt.Printf("Found %d users\n", len(result.Payload.Users))
			for _, u := range result.Payload.Users {
				fmt.Println("  - " + u.Login)
			}
		}

	},
}

func init() {
	idmCmd.AddCommand(listUsers)
}
