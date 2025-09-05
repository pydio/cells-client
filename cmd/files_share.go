package cmd

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/rest"
	"github.com/pydio/cells-sdk-go/v4/client/acl_service"
	"github.com/pydio/cells-sdk-go/v4/models"
	"github.com/pydio/cells-sdk-go/v4/client/user_service"
)

var shareNode = &cobra.Command{
	Use:   "share",
	Short: "Share a single file or folder",
	Long: `
DESCRIPTION

  Create a public link that adds public access to the passed path on the server
  or grant access to a specific role.

EXAMPLES

  1/ Create a link with a technical ID
  $ ` + os.Args[0] + ` share common-files/MyPublicImage.jpg
  Public link created at https://pydio.example.com/public/479cc5dbdf8b

  2/ Share a file with a specific user
  $ ` + os.Args[0] + ` share personal-files/SecretDoc.pdf --user john
  (no output, but user 'john' now has write access to the file)

  Note: you can use the 'idm list-users' command to get a list of usernames to share with.

  3/ Share a folder with a specific user
  $ ` + os.Args[0] + ` share personal-files/Projects --user alice
  (no output, but user 'alice' now has write access to the folder)
`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		p := args[0]
		ctx := cmd.Context()
		node, exists := sdkClient.StatNode(ctx, p)

		if !exists {
			// Avoid 404 errors
			cmd.Printf("Could create link, no node found at %s\n", p)
			return
		}

		if username != "" {
			apiClient := sdkClient.GetApiClient()
			// find the user
			user, err := apiClient.UserService.GetUser(&user_service.GetUserParams{
				Login:   username,
				Context: ctx,
			})
			if err != nil {
				log.Fatal(err)
			}

			actionValue := "false"
			if modify {
				actionValue = "true"
			}
			params := &acl_service.PutACLParams{
				Body: &models.IdmACL{
					Action: &models.IdmACLAction{
						Name:  "write",
						Value: actionValue,
					},
					NodeID: node.UUID,
					RoleID: user.Payload.UUID,
				},
				Context: ctx,
			}

			_, err = apiClient.ACLService.PutACL(params)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			l, err := sdkClient.CreateSimpleFolderLink(ctx, node.UUID, path.Base(p))
			if err != nil {
				log.Fatal(err)
			}

			cmd.Println("Public link created at " + rest.StandardizeLink(sdkClient.GetConfig(), l.LinkURL))
		}
		fmt.Println("") // Add a line to reduce glitches in the terminal
	},
}

var username string
var modify bool

func init() {
	shareNode.Flags().StringVarP(&username, "user", "", "", "Username to grant access to")
	shareNode.Flags().BoolVarP(&modify, "modify", "m", false, "Grant modify permission (only with --user)")
    RootCmd.AddCommand(shareNode)
}