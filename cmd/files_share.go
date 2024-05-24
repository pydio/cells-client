package cmd

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/rest"
)

var shareNode = &cobra.Command{
	Use:   "share",
	Short: "Share a single file or folder",
	Long: `
DESCRIPTION

  Create a public link that adds public access to the passed path on the server.

EXAMPLES

  1/ Create a link with a technical ID
  $ ` + os.Args[0] + ` share common-files/MyPublicImage.jpg
  Public link created at https://pydio.example.com/public/479cc5dbdf8b

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

		l, err := sdkClient.CreateSimpleFolderLink(ctx, node.UUID, path.Base(p))
		if err != nil {
			log.Fatal(err)
		}

		cmd.Println("Public link created at " + rest.StandardizeLink(sdkClient.GetConfig(), l.LinkURL))
		fmt.Println("") // Add a line to reduce glitches in the terminal
	},
}

func init() {
	RootCmd.AddCommand(shareNode)
}
