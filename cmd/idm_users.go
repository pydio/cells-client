package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v4/client/user_service"
	"github.com/pydio/cells-sdk-go/v4/models"
)

var listUsers = &cobra.Command{
	Use:   "list-users",
	Short: "List users",
	Long: `
DESCRIPTION	

  List the users defined in your Pydio Cells instance.
`,
	Run: func(cmd *cobra.Command, args []string) {

		ctx := cmd.Context()
		apiClient := sdkClient.GetApiClient()

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
			fmt.Println("Listing users")

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Login", "Type"})

			for _, u := range result.Payload.Users {
				profile := ""
				if u.Attributes != nil {
					if p, ok := u.Attributes["hidden"]; ok {
						if p == "true" {
							continue
						}
					}

					if p, ok := u.Attributes["profile"]; ok {
						if p == "anon" {
							continue
						}
						profile = p
					}
				}
				table.Append([]string{u.Login, profile})
			}
			table.Render()
		} else {
			fmt.Println("No user found.")
		}
	},
}

func init() {
	idmCmd.AddCommand(listUsers)
}
