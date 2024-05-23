package rest

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/pydio/cells-sdk-go/v5/client/tree_service"
	"github.com/pydio/cells-sdk-go/v5/models"
)

const pageSize = 100

func StatNode(ctx context.Context, pathToFile string) (*models.TreeNode, bool) {
	client, e := GetApiClient(ctx)
	if e != nil {
		return nil, false
	}
	params := &tree_service.HeadNodeParams{}
	params.SetNode(pathToFile)
	params.SetContext(ctx)
	resp, err := client.TreeService.HeadNode(params)
	if err != nil {
		//if errors.As(err, &tree_service.HeadNodeNotFound{}) {
		//	return nil, false
		//}
		switch err.(type) {
		case *tree_service.HeadNodeNotFound:
			return nil, false
		}
		fmt.Println("#############")
		fmt.Println("#############")
		fmt.Printf("Could not stat %s: %s\n", pathToFile, err.Error())
		fmt.Printf("Could not stat %s: %s\n", pathToFile, err.Error())
		// sleep and retry
		time.Sleep(2000 * time.Millisecond)
		resp, err = client.TreeService.HeadNode(params)
		if err != nil {
			//if errors.As(err, &tree_service.HeadNodeNotFound{}) {
			//	return nil, false
			//}
			switch err.(type) {
			case *tree_service.HeadNodeNotFound:
				return nil, false
			}
			// Try to refresh
			refreshed, err2 := CellsStore().RefreshIfRequired(ctx, DefaultConfig.SdkConfig)
			if err2 != nil {
				fmt.Println("#############")
				fmt.Println("#############")
				fmt.Println("Could not refresh:", err.Error())
				return nil, false
			} else if refreshed {
				fmt.Println("#############")
				fmt.Println("#############")
				fmt.Println("#############")
				fmt.Println("Node: Token has been refreshed")
				fmt.Println("Node: Token has been refreshed")
			}
			client, err2 = GetApiClient(ctx)
			if err2 != nil {
				fmt.Println("#################")
				fmt.Println("[Error] Could not retrieve client after token refresh")
				return nil, false
			}
			resp, err = client.TreeService.HeadNode(params)
			if err != nil {
				switch err.(type) {
				case *tree_service.HeadNodeNotFound:
					return nil, false
				}
				fmt.Println("#############")
				fmt.Println("Abort the mission:", err.Error())
			}
		}
	}
	if err == nil && resp.Payload.Node != nil {
		return resp.Payload.Node, true
	} else {
		return nil, false
	}
}

func ListNodesPath(ctx context.Context, path string) ([]string, error) {
	client, err := GetApiClient(ctx)
	if err != nil {
		return nil, err
	}
	params := tree_service.NewBulkStatNodesParamsWithContext(ctx)
	params.Body = &models.RestGetBulkMetaRequest{
		Limit:     100,
		NodePaths: []string{path},
	}
	res, e := client.TreeService.BulkStatNodes(params)
	if e != nil {
		return nil, e
	}
	var nodes []string
	if len(res.Payload.Nodes) == 0 {
		return nil, nil
	}
	for _, node := range res.Payload.Nodes {
		nodes = append(nodes, node.Path)
	}
	return nodes, nil
}

func DeleteNode(ctx context.Context, paths []string, permanently ...bool) (jobUUIDs []string, e error) {
	if len(paths) == 0 {
		e = fmt.Errorf("no paths found to delete")
		return
	}
	client, err := GetApiClient(ctx)
	if err != nil {
		e = err
		return
	}
	var nn []*models.TreeNode
	for _, p := range paths {
		nn = append(nn, &models.TreeNode{Path: p})
	}

	var perm bool
	if len(permanently) > 0 && permanently[0] {
		perm = true
	}

	params := tree_service.NewDeleteNodesParamsWithContext(ctx)
	params.Body = &models.RestDeleteNodesRequest{
		Nodes:             nn,
		RemovePermanently: perm,
	}
	res, err := client.TreeService.DeleteNodes(params)
	if err != nil {
		e = err
		return
	}

	for _, job := range res.Payload.DeleteJobs {
		jobUUIDs = append(jobUUIDs, job.UUID)
	}
	return
}

func GetAllBulkMeta(ctx context.Context, path string) (nodes []*models.TreeNode, err error) {
	client, err := GetApiClient(ctx)
	if err != nil {
		return nil, err
	}
	params := tree_service.NewBulkStatNodesParamsWithContext(ctx)
	params.Body = &models.RestGetBulkMetaRequest{
		Limit:     pageSize,
		NodePaths: []string{path},
	}
	res, e := client.TreeService.BulkStatNodes(params)
	if e != nil {
		return nil, e
	}

	nodes = append(nodes, res.Payload.Nodes...)

	if len(nodes) >= pageSize {
		pg := res.Payload.Pagination
		for i := pageSize; i <= int(pg.Total); i += pageSize {
			params.Body.Offset = int32(i)
			res, err = client.TreeService.BulkStatNodes(params)
			if err != nil {
				return
			}
			nodes = append(nodes, res.Payload.Nodes...)
			pg = res.Payload.Pagination
		}
	}
	return nodes, nil
}

// createRemoteFolders creates necessary folders on the distant server.
func createRemoteFolders(ctx context.Context, mm []*models.TreeNode, pool *BarsPool) error {

	client, err := GetApiClient(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < len(mm); i += pageSize {
		end := i + pageSize
		if end > len(mm) {
			end = len(mm)
		}
		subArray := mm[i:end]

		params := tree_service.NewCreateNodesParams()
		params.Body = &models.RestCreateNodesRequest{
			Nodes:     subArray,
			Recursive: false,
		}
		_, err := client.TreeService.CreateNodes(params)
		if err != nil {
			return errors.Errorf("could not create folders: %s", err.Error())
		}
		// TODO:  Stat all folders to make sure they are indexed ?
		if pool != nil {
			for range subArray {
				pool.Done()
			}
		} else { // verbose mode
			fmt.Printf("... Created %d folders on remote server\n", end)
		}
	}
	return nil
}
