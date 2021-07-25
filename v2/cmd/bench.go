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
	benchPoolSize    int
	benchMaxRequests int
)

var benchCmd = &cobra.Command{
	Use: "bench",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to the Pydio API via the sdkConfig
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
	benchCmd.Flags().IntVarP(&benchPoolSize, "pool", "p", 1, "Pool Size")
	benchCmd.Flags().IntVarP(&benchMaxRequests, "max", "m", 100, "Max Requests")
}
