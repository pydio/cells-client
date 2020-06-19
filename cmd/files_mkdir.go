package cmd

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/micro/go-log"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
	"github.com/pydio/cells-sdk-go/client/tree_service"
	"github.com/pydio/cells-sdk-go/models"
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

		// Connect to the Pydio API via the sdkConfig
		ctx, apiClient, err := rest.GetApiClient()
		if err != nil {
			log.Fatal(err)
		}
		var dirs []*models.TreeNode
		var paths []string
		var crt = parts[0]
		for i := 1; i < len(parts); i++ {
			crt = path.Join(crt, parts[i])
			if _, e := apiClient.TreeService.HeadNode(&tree_service.HeadNodeParams{Node: crt, Context: ctx}); e != nil {
				dirs = append(dirs, &models.TreeNode{Path: crt})
				paths = append(paths, crt)
			}
		}
		if len(dirs) == 0 {
			fmt.Println("All dirs already exist, exiting")
			return
		}
		fmt.Printf("Creating folder(s) %s\n", strings.Join(paths, ","))
		_, err = apiClient.TreeService.CreateNodes(&tree_service.CreateNodesParams{
			Body: &models.RestCreateNodesRequest{
				Nodes: dirs,
			},
			Context: ctx,
		})
		if err != nil {
			log.Fatal("error while calling CreateNodes:", err)
		}
		// Wait that it is indexed
		e := rest.RetryCallback(func() error {
			_, e := apiClient.TreeService.HeadNode(&tree_service.HeadNodeParams{Node: dir, Context: ctx})
			if e != nil {
				fmt.Println("Waiting for folder to be correctly indexed...")
			}
			return e
		}, 10, 2*time.Second)

		if e != nil {
			log.Fatal(e)
		}
		fmt.Printf("SUCCESS: Dir %s created and indexed\n", dir)

	},
}

func init() {
	RootCmd.AddCommand(mkDir)
}
