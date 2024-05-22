package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v5/client/user_service"
	"github.com/pydio/cells-sdk-go/v5/models"

	"github.com/pydio/cells-client/v4/rest"
)

var listGroups = &cobra.Command{
	Use:   "list-groups",
	Short: "List groups",
	Long: `
DESCRIPTION

  List user groups that are defined in your Pydio Cells instance.
`,
	Run: func(cmd *cobra.Command, args []string) {

		apiClient, err := rest.GetApiClient(cmd.Context())
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
