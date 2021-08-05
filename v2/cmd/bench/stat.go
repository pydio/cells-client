package bench

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
	benchResourcePath string
)

var statCmd = &cobra.Command{
	Use:   "stat",
	Short: "Perform a set of stats calls in concurrency",
	Long:  "This command creates a simple resource (a folder) and then sends tons of stats on this resource in parallel.",
	Run: func(cmd *cobra.Command, args []string) {
		// Connect to the Pydio API via the sdkConfig

		if benchResourcePath == "" {
			benchResourcePath = "common-files/test-bench-dir-" + rest.Unique(4)
		}

		if !benchSkipCreate {
			ctx, apiClient, err := rest.GetApiClient()
			if err != nil {
				log.Fatal(err)
			}
			_, err = apiClient.TreeService.CreateNodes(&tree_service.CreateNodesParams{
				Body: &models.RestCreateNodesRequest{
					Nodes: []*models.TreeNode{{
						Path: benchResourcePath,
					}},
				},
				Context: ctx,
			})
			if err != nil {
				log.Fatal(err)
			}

			exists := false
			for {
				_, exists = rest.StatNode(benchResourcePath)
				if exists {
					break
				}
				log.Printf("No node found at %s, wait for a second before retry\n", benchResourcePath)
				time.Sleep(1 * time.Second)
			}
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

		if !benchSkipCreate && !benchSkipClean {
			rest.DeleteNode([]string{benchResourcePath})
		}
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
	res := time.Since(s)
	if err != nil {
		log.Println(i, res, "error", err.Error())
	} else {
		log.Println(i, res, "ok")
	}

	return err
}

func init() {
	benchCmd.AddCommand(statCmd)
	statCmd.Flags().StringVarP(&benchResourcePath, "resource", "r", "common-files/test-bench-dir", "Folder created that will be stated")
}
