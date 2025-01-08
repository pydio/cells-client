package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v4/client/workspace_service"
	"github.com/pydio/cells-sdk-go/v4/models"
)

var listWorkspaces = &cobra.Command{
	Use:   "list-workspaces",
	Short: "List workspaces",
	Long: `
DESCRIPTION	

  List all workspaces on which the current logged in user has *at least* Read Access.

`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		apiClient := sdkClient.GetApiClient()

		//retrieves the users using the searchWorkspacesParams function
		params := &workspace_service.SearchWorkspacesParams{
			Body:    &models.RestSearchWorkspaceRequest{CountOnly: true},
			Context: ctx,
		}

		//assigns the workspaces data retrieved above in the results variable
		result, err := apiClient.WorkspaceService.SearchWorkspaces(params)
		if err != nil {
			fmt.Printf("could not list workspaces: %s\n", err.Error())
			log.Fatal(err)
		}

		//prints the workspace label
		if len(result.Payload.Workspaces) > 0 {
			fmt.Printf("Found %d workspaces:\n", len(result.Payload.Workspaces))
			for _, u := range result.Payload.Workspaces {
				fmt.Println("\t- " + u.Label)
			}
		}

	},
}

func init() {
	idmCmd.AddCommand(listWorkspaces)
}
