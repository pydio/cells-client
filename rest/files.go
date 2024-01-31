package rest

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"

	"github.com/pydio/cells-sdk-go/v5/client/tree_service"
	"github.com/pydio/cells-sdk-go/v5/models"
	sdk_s3 "github.com/pydio/cells-sdk-go/v5/transport/s3"

	"github.com/pydio/cells-client/v4/common"
)

func StatNode(ctx context.Context, pathToFile string) (*models.TreeNode, bool) {
	client, e := GetApiClient()
	if e != nil {
		return nil, false
	}
	params := &tree_service.HeadNodeParams{}
	params.SetNode(pathToFile)
	params.SetContext(ctx)
	resp, err := client.TreeService.HeadNode(params)
	if err == nil && resp.Payload.Node != nil {
		return resp.Payload.Node, true
	} else {
		return nil, false
	}
}

func ListNodesPath(ctx context.Context, path string) ([]string, error) {
	client, err := GetApiClient()
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

func DeleteNode(ctx context.Context, paths []string) (jobUUIDs []string, e error) {
	if len(paths) == 0 {
		e = fmt.Errorf("no paths found to delete")
		return
	}
	client, err := GetApiClient()
	if err != nil {
		e = err
		return
	}
	var nn []*models.TreeNode
	for _, p := range paths {
		nn = append(nn, &models.TreeNode{Path: p})
	}

	params := tree_service.NewDeleteNodesParamsWithContext(ctx)
	params.Body = &models.RestDeleteNodesRequest{
		Nodes: nn,
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

const pageSize = 100

func GetAllBulkMeta(ctx context.Context, path string) (nodes []*models.TreeNode, err error) {
	client, err := GetApiClient()
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
			// params = tree_service.NewBulkStatNodesParams()
			// params.Body = &models.RestGetBulkMetaRequest{
			// 	Limit:     pageSize,
			// 	NodePaths: []string{path},
			// 	Offset:    int32(i),
			// }
			params.Body.Offset = int32(i)
			res, err = client.TreeService.BulkStatNodes(params)
			if err != nil {
				return
			}
			nodes = append(nodes, res.Payload.Nodes...)
			pg = res.Payload.Pagination
			fmt.Println("#", i, "Current page:", pg.CurrentPage, "CurrentOffset:", pg.CurrentOffset, "TotalPages: ", pg.TotalPages)
			fmt.Println(" Found:", len(res.Payload.Nodes), "nodes in page ", pg.CurrentOffset, "- TotalPages:", pg.TotalPages)
			fmt.Println(" Length after append:", len(nodes))
		}

	}
	return nodes, nil
}

func TreeCreateNodes(nodes []*models.TreeNode) error {
	client, err := GetApiClient()
	if err != nil {
		return err

	}
	params := tree_service.NewCreateNodesParams()
	params.Body = &models.RestCreateNodesRequest{
		Nodes:     nodes,
		Recursive: false,
	}

	_, e := client.TreeService.CreateNodes(params)
	if e != nil {
		return e
	}
	// TODO monitor jobs to wait for the index
	return nil
}

func GetFile(ctx context.Context, pathToFile string) (io.Reader, int, error) {

	s3Client, bucketName, e := GetS3Client(ctx)
	if e != nil {
		return nil, 0, e
	}
	hO, err := s3Client.HeadObject(
		ctx,
		&s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(pathToFile),
		},
	)
	if err != nil {
		return nil, 0, err
	}

	obj, err := s3Client.GetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(pathToFile),
		},
	)
	if err != nil {
		return nil, 0, err
	}
	return obj.Body, int(*hO.ContentLength), nil
}

func PutFile(ctx context.Context, pathToFile string, content io.ReadSeeker, checkExists bool, errChan ...chan error) (*s3.PutObjectOutput, error) {

	s3Client, bucketName, e := GetS3Client(ctx)
	if e != nil {
		return nil, e
	}

	key := pathToFile
	var obj *s3.PutObjectOutput
	e = RetryCallback(func() error {
		var err error
		obj, err = s3Client.PutObject(
			ctx,
			&s3.PutObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(pathToFile),
				Body:   content,
			},
		)
		if err != nil {
			if len(errChan) > 0 {
				errChan[0] <- err
			} else {
				fmt.Printf(" ## Could not upload file %s, cause: %s\n", key, err.Error())
			}
		}
		return err
	}, 3, 2*time.Second)
	if e != nil {
		return nil, fmt.Errorf("could not put object in bucket %s with key %s, \ncause: %s", bucketName, key, e.Error())
	}

	if checkExists {
		fmt.Println(" ## Waiting for file to be indexed...")
		// Now stat Node to make sure it is indexed
		e = RetryCallback(func() error {
			_, ok := StatNode(ctx, pathToFile)
			if !ok {
				return fmt.Errorf("cannot stat node just after PutFile operation")
			}
			return nil

		}, 3, 3*time.Second)
		if e != nil {
			return nil, e
		}
		fmt.Println(" ## File correctly indexed")
	}
	return obj, nil
}

func uploadManager(ctx context.Context, stats os.FileInfo, path string, content io.ReadSeeker, errChan ...chan error) error {

	s3Client, bucketName, err := GetS3Client(ctx)
	if err != nil {
		return err
	}

	fSize := stats.Size()
	ps, err := computePartSize(fSize)
	if err != nil {
		if errChan != nil {
			errChan[0] <- err
		}
		return err
	}

	uploader := manager.NewUploader(s3Client,
		func(u *manager.Uploader) {
			u.Concurrency = common.UploadPartsConcurrency
			u.PartSize = ps
		},
	)

	// Adds a callback entry point so that we can follow the effective part upload.
	uploader.BufferProvider = sdk_s3.NewCallbackTransferProvider(path, fSize, ps)

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(path),
		Body:   content,
	})

	if err != nil {
		if aerr, ok := err.(smithy.APIError); ok {
			// TODO better error handling
			if errChan != nil {
				errChan[0] <- aerr
			}
			return aerr
		}
		if errChan != nil {
			errChan[0] <- err
		}
		return err
	}

	return nil
}

func computePartSize(fileSize int64) (partSize int64, er error) {

	partSize = common.UploadDefaultPartSize * (1024 * 1024)
	maxNumberOfParts := common.UploadMaxPartsNumber
	steps := common.UploadPartsSteps
	if partSize%steps != 0 {
		return 0, fmt.Errorf("PartSize must be a multiple of 10MB")
	}

	if mnp := os.Getenv("CELLS_MAX_PARTS_NUMBER"); mnp != "" {
		if m, e := strconv.Atoi(mnp); e == nil {
			maxNumberOfParts = int64(m)
		}
	}
	if int64(float64(fileSize)/float64(partSize)) < maxNumberOfParts {
		return
	}
	partSize = int64(float64(fileSize) / float64(maxNumberOfParts))
	partSize = partSize + steps - partSize%steps
	return
}
