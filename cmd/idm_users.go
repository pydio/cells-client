package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v5/client/user_service"
	"github.com/pydio/cells-sdk-go/v5/models"

	"github.com/pydio/cells-client/v4/rest"
)

var listUsers = &cobra.Command{
	Use:   "list-users",
	Short: "List users",
	Long: `
DESCRIPTION	

  List the users defined in your Pydio Cells instance.
`,
	Run: func(cmd *cobra.Command, args []string) {

		apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err)
		}
		ctx := cmd.Context()
		q := &models.IdmUserSingleQuery{Login: "*"}
		r := &models.RestSearchUserRequest{Queries: []*models.IdmUserSingleQuery{q}}

		result, err := apiClient.UserService.SearchUsers(&user_service.SearchUsersParams{
			Body:    r,
			Context: ctx,
		})
		if err != nil {
			fmt.Printf("could not list users: %s\n", err.Error())
			log.Fatal(err)
		}

		if len(result.Payload.Users) > 0 {
			msg := fmt.Sprintf("Found %d users:", len(result.Payload.Users))
			if len(result.Payload.Users) == 1 {
				msg = "Found 1 user:"
			}
			fmt.Println(msg)
			for _, u := range result.Payload.Users {
				fmt.Println("  - " + u.Login)
			}
		} else {
			fmt.Println("No user found.")
		}
	},
}

func init() {
	idmCmd.AddCommand(listUsers)
}
