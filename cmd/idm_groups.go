package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v4/client/user_service"
	"github.com/pydio/cells-sdk-go/v4/models"
)

var listGroups = &cobra.Command{
	Use:   "list-groups",
	Short: "List groups",
	Long: `
DESCRIPTION

  List user groups that are defined in your Pydio Cells instance.
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
			fmt.Printf("could not list groups %s\n", err.Error())
			log.Fatal(err)
		}

		if len(result.Payload.Groups) > 0 {
			msg := fmt.Sprintf("Found %d groups:", len(result.Payload.Groups))
			if len(result.Payload.Groups) == 1 {
				msg = "Found 1 group:"
			}
			fmt.Println(msg)
			for _, u := range result.Payload.Groups {
				fmt.Println("  - " + u.GroupLabel)
			}
		} else {
			fmt.Println("No group found.")
		}
	},
}

func init() {
	idmCmd.AddCommand(listGroups)
}
