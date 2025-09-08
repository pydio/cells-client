package cmd

import (
	"log"
	"github.com/go-openapi/strfmt"
	"github.com/pydio/cells-sdk-go/v4/client/share_service"
	"github.com/pydio/cells-sdk-go/v4/models"
	"github.com/pydio/cells-sdk-go/v4/client/user_service"

	"github.com/spf13/cobra"
)

var cellCmd = &cobra.Command{
	Use:   "create-cell",
	Short: "Create a cell",
	Long: `
DESCRIPTION

  Create a cell to collaborate with another user.

EXAMPLES

  1/ cec create-cell label --username username --write --invite
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		apiClient := sdkClient.GetApiClient()

		// find the user
		user, err := apiClient.UserService.GetUser(&user_service.GetUserParams{
			Login:   username,
			Context: ctx,
		})
		if err != nil {
			log.Fatal(err)
		}

		label := args[0]
		cellshare := share_service.New(apiClient.Transport, strfmt.Default)

		// create a cell
		result, err := cellshare.PutCell(&share_service.PutCellParams{
			Body: &models.RestPutCellRequest{
				Room: &models.RestCell{
					Label: label,
					ACLs: map[string]models.RestCellACL{
						user.Payload.UUID: {
							 RoleID: user.Payload.UUID,
							 User: user.Payload,
							 Actions: []*models.IdmACLAction{
								{
									Name:  "read",
									Value: "1",
								},
								{
									Name:  "write",
									Value: "1",
								},
							},
						 },
					},
				},
				CreateEmptyRoot: true,  // TODO: allow specifing a folder
			},
			Context: ctx,
		})
		if err != nil {
			log.Fatal(err)
		}
		println("Cell created with ID:", result.Payload.UUID)

		if invite {
			NewCellInvitation(ctx, username, label, label)   // TODO: does this wont work if nodes are specified?
		}
	},
}

var username string
var invite bool
var write bool

func init() {
	cellCmd.Flags().BoolVarP(&invite, "invite", "", false, "Specify to send a cell invitation (assumes they have an email address)")
	cellCmd.Flags().BoolVarP(&write, "write", "", false, "Grant user write (Modify) access to the Cell")
	cellCmd.Flags().StringVarP(&username, "username", "", "", "Specify the user to add to the Cell")   // TODO: csv list of users
	RootCmd.AddCommand(cellCmd)
}