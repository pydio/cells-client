package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/micro/go-log"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
	"github.com/pydio/cells-sdk-go/client/tree_service"
	"github.com/pydio/cells-sdk-go/models"
	"github.com/pydio/cells/common/service"
)

var mkDir = &cobra.Command{
	Use:   "mkdir",
	Short: `Create folder on remote server`,
	Long: `Create a folder on remote Cells server

Use path including workspace slug 
`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 1 {
			log.Fatal(fmt.Errorf("please provide the target path"))
		}
		dir := args[0]
		parts := strings.Split(dir, "/")
		if len(parts) < 2 {
			log.Fatal("Please provide at least a workspace segment in the path")
		}

		fmt.Printf("Creating folder %s\n", dir)
		//connects to the pydio api via the sdkConfig
		ctx, apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err)
		}
		_, err = apiClient.TreeService.CreateNodes(&tree_service.CreateNodesParams{
			Body: &models.RestCreateNodesRequest{
				Nodes:     []*models.TreeNode{{Path: dir}},
				Recursive: true,
			},
			Context: ctx,
		})
		if err != nil {
			log.Fatal(err)
		}
		// Wait that it's indexed
		e := service.Retry(func() error {
			_, e := apiClient.TreeService.HeadNode(&tree_service.HeadNodeParams{Node: dir, Context: ctx})
			if e != nil {
				log.Log("Waiting for folder to be correctly indexed...")
			}
			return e
		}, 2*time.Second, 10*time.Second)

		if e != nil {
			log.Fatal(e)
		}
		log.Logf("SUCCESS: Dir %s created and indexed", dir)

	},
}

func init() {
	RootCmd.AddCommand(mkDir)
}
