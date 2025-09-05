package cmd

import (
	//"fmt"
	"log"
	"os"

	//"github.com/olekukonko/tablewriter"
	"github.com/pydio/cells-sdk-go/v4/client/user_service"
	"github.com/pydio/cells-sdk-go/v4/models"
	"github.com/spf13/cobra"
	//"github.com/pydio/cells-sdk-go/v4/client/user_service"
	//"github.com/pydio/cells-sdk-go/v4/models"
)

var createSharedUser = &cobra.Command{
	Use:   "create-shared-user",
	Short: "Create a shared user",
	Long: `
DESCRIPTION	

  Create a shared user for external file sharing.

EXAMPLES
  1/ Create a shared user
  $ ` + os.Args[0] + ` idm create-shared-user --userlogin newguy --userpassword testing123 --display-name "New Guy" --email newguy@example.com
`,
	Run: func(cmd *cobra.Command, args []string) {

	ctx := cmd.Context()
	apiClient := sdkClient.GetApiClient()

	params := user_service.NewPutUserParams().
		WithLogin(userlogin).
		WithBody(user_service.PutUserBody{
			IsGroup:  false,
			Password: userpassword,
			Attributes: map[string]string{
				"displayName": displayName,
				"email":       email,
			},
			Roles: []*models.IdmRole{},
		}).
		WithContext(ctx)

		result, err := apiClient.UserService.PutUser(params)
		if err != nil {
			log.Fatal(err)
		}

		println("Shared user created with login:", result.Payload.Login)
	},
}

var userlogin string
var userpassword string
var displayName string
var email string

func init() {
	createSharedUser.Flags().StringVarP(&userlogin, "userlogin", "", "", "Shared User login to create")
	createSharedUser.Flags().StringVarP(&userpassword, "userpassword", "", "", "Shared User password to create")
	createSharedUser.Flags().StringVarP(&displayName, "display-name", "d", "", "Shared User display name")
	createSharedUser.Flags().StringVarP(&email, "email", "e", "", "Shared User email")
	idmCmd.AddCommand(createSharedUser)
}
