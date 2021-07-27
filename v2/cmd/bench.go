package cmd

import (
	"log"
	"sync"
	"time"

	"github.com/pydio/cells-client/v2/rest"
	"github.com/pydio/cells-sdk-go/v2/client/tree_service"
	"github.com/pydio/cells-sdk-go/v2/models"
	"github.com/spf13/cobra"
)

var (
	benchPoolSize     int
	benchMaxRequests  int
	benchSkipCreate   bool
	benchResourcePath string
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Perform a set of stats calls in concurrency",
	Long:  "This command creates a simple resource (a folder) and then sends tons of stats on this resource in parallel.",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to the Pydio API via the sdkConfig
		if !benchSkipCreate {
			ctx, apiClient, err := rest.GetApiClient()
			if err != nil {
				log.Fatal(err)
			}
			_, err = apiClient.TreeService.CreateNodes(&tree_service.CreateNodesParams{
				Body: &models.RestCreateNodesRequest{
					Nodes: []*models.TreeNode{{
						Path: "common-files/test-bench-dir",
					}},
				},
				Context: ctx,
			})
		}

		wg := &sync.WaitGroup{}
		wg.Add(benchMaxRequests)
		throttle := make(chan struct{}, benchPoolSize)
		for i := 0; i < benchMaxRequests; i++ {
			throttle <- struct{}{}
			go func(id int) {
				benchStat(id, "common-files/test-bench-dir")
				wg.Done()
				<-throttle
			}(i)
		}
		wg.Wait()
	},
}

func benchStat(i int, node string) error {
	s := time.Now()
	ctx, apiClient, err := rest.GetApiClient()
	if err != nil {
		return err
	}
	_, err = apiClient.TreeService.HeadNode(&tree_service.HeadNodeParams{
		Node:    node,
		Context: ctx,
	})
	res := time.Now().Sub(s)
	if err != nil {
		log.Println(i, res, "error", err.Error())
	} else {
		log.Println(i, res, "ok")
	}

	return err
}

func init() {
	RootCmd.AddCommand(benchCmd)
	benchCmd.Flags().StringVarP(&benchResourcePath, "resource", "r", "common-files/test-bench-dir", "Folder created that will be stated")
	benchCmd.Flags().IntVarP(&benchPoolSize, "pool", "p", 1, "Pool size (number of parallel requests)")
	benchCmd.Flags().IntVarP(&benchMaxRequests, "max", "m", 100, "Total number of Stat requests sent")
	benchCmd.Flags().BoolVarP(&benchSkipCreate, "no-create", "n", false, "Skip test resource creation (if it is already existing)")
}
