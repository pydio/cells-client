package rest

import (
	"context"
	"time"

	"github.com/pydio/cells-sdk-go/v5/client/tree_service"
	"github.com/pydio/cells-sdk-go/v5/models"
)

const pageSize = 100

func (client *SdkClient) StatNode(ctx context.Context, pathToFile string) (*models.TreeNode, bool) {
	exists := false
	var node *models.TreeNode
	e := RetryCallback(func() error {
		params := &tree_service.HeadNodeParams{}
		params.SetNode(pathToFile)
		params.SetContext(ctx)
		resp, err := client.GetApiClient().TreeService.HeadNode(params)
		if err != nil {
			switch err.(type) {
			case *tree_service.HeadNodeNotFound:
				return nil
			}
			return err
		}
		if resp.IsSuccess() {
			exists = true
			node = resp.Payload.Node
		}
		return nil
	}, 5, 2*time.Second)

	if e != nil {
		Log.Debugf("Could not stat node at %s, cause: %s", pathToFile, e.Error())
		return nil, false
	}
	return node, exists
}

func (client *SdkClient) ListNodesPath(ctx context.Context, path string) ([]string, error) {
	params := tree_service.NewBulkStatNodesParamsWithContext(ctx)
	params.Body = &models.RestGetBulkMetaRequest{
		Limit:     100,
		NodePaths: []string{path},
	}
	res, e := client.GetApiClient().TreeService.BulkStatNodes(params)
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

func (client *SdkClient) DeleteNodes(ctx context.Context, paths []string, permanently ...bool) (jobUUIDs []string, e error) {
	if len(paths) == 0 { // List is empty
		Log.Warnln("called DeleteNodes with an empty list")
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
	res, err := client.GetApiClient().TreeService.DeleteNodes(params)
	if err != nil {
		e = err
		return
	}

	for _, job := range res.Payload.DeleteJobs {
		jobUUIDs = append(jobUUIDs, job.UUID)
	}
	return
}

func (client *SdkClient) GetAllBulkMeta(ctx context.Context, path string) (nodes []*models.TreeNode, err error) {
	params := tree_service.NewBulkStatNodesParamsWithContext(ctx)
	params.Body = &models.RestGetBulkMetaRequest{
		Limit:     pageSize,
		NodePaths: []string{path},
	}
	res, e := client.GetApiClient().TreeService.BulkStatNodes(params)
	if e != nil {
		return nil, e
	}

	nodes = append(nodes, res.Payload.Nodes...)

	if len(nodes) >= pageSize {
		pg := res.Payload.Pagination
		for i := pageSize; i <= int(pg.Total); i += pageSize {
			params.Body.Offset = int32(i)
			res, err = client.GetApiClient().TreeService.BulkStatNodes(params)
			if err != nil {
				return
			}
			nodes = append(nodes, res.Payload.Nodes...)
			pg = res.Payload.Pagination
		}
	}
	return nodes, nil
}
