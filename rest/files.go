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
	"github.com/pydio/cells-sdk-go/v5/transport"
	sdk_s3 "github.com/pydio/cells-sdk-go/v5/transport/s3"

	"github.com/pydio/cells-client/v4/common"
)

func StatNode(pathToFile string) (*models.TreeNode, bool) {
	ctx, client, e := GetApiClient()
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

func ListNodesPath(path string) ([]string, error) {
	_, client, err := GetApiClient()
	if err != nil {
		return nil, err
	}
	params := tree_service.NewBulkStatNodesParams()
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

func DeleteNode(paths []string) (jobUUIDs []string, e error) {
	if len(paths) == 0 {
		e = fmt.Errorf("no paths found to delete")
		return
	}
	_, client, err := GetApiClient()
	if err != nil {
		e = err
		return
	}
	var nn []*models.TreeNode
	for _, p := range paths {
		nn = append(nn, &models.TreeNode{Path: p})
	}

	params := tree_service.NewDeleteNodesParams()
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

func GetBulkMetaNode(path string) ([]*models.TreeNode, error) {
	_, client, err := GetApiClient()
	if err != nil {
		return nil, err
	}
	params := tree_service.NewBulkStatNodesParams()
	params.Body = &models.RestGetBulkMetaRequest{
		Limit:     100,
		NodePaths: []string{path},
	}
	res, e := client.TreeService.BulkStatNodes(params)
	if e != nil {
		return nil, e
	}
	return res.Payload.Nodes, nil
}

func TreeCreateNodes(nodes []*models.TreeNode) error {
	_, client, err := GetApiClient()
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

func GetFile(pathToFile string) (io.Reader, int, error) {

	s3Client, bucketName, e := getS3Client()
	if e != nil {
		return nil, 0, e
	}
	hO, err := s3Client.HeadObject(
		context.TODO(),
		&s3.HeadObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(pathToFile),
		},
	)
	if err != nil {
		return nil, 0, err
	}
	size := int(*hO.ContentLength)

	obj, err := s3Client.GetObject(
		context.TODO(),
		&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(pathToFile),
		},
	)
	if err != nil {
		return nil, 0, err
	}
	return obj.Body, size, nil
}

func PutFile(pathToFile string, content io.ReadSeeker, checkExists bool, errChan ...chan error) (*s3.PutObjectOutput, error) {

	s3Client, bucketName, e := getS3Client()
	if e != nil {
		return nil, e
	}

	key := pathToFile
	var obj *s3.PutObjectOutput
	e = RetryCallback(func() error {
		var err error
		obj, err = s3Client.PutObject(
			context.TODO(),
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
				fmt.Println(" ## Trying to Put file:", key, "Error:", err.Error())
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
			_, ok := StatNode(pathToFile)
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

func getS3Client() (*s3.Client, string, error) {

	// FIXME enrich User-Agent
	DefaultConfig.CustomHeaders = map[string]string{
		transport.UserAgentKey: common.AppName + "/" + common.Version,
	}

	// TODO this must be done before
	s3Config := getS3ConfigFromSdkConfig(DefaultConfig)
	bucketName := s3Config.Bucket

	s3Client, e := sdk_s3.GetClient(CellsStore, DefaultConfig.SdkConfig, s3Config)
	if e != nil {
		return nil, "", e
	}
	// s3Client.Config.S3DisableContentMD5Validation = aws.Bool(true)
	return s3Client, bucketName, e
}

func uploadManager(stats os.FileInfo, path string, content io.ReadSeeker, errChan ...chan error) error {

	s3Client, bucketName, err := getS3Client()
	if err != nil {
		return err
	}

	ps, err := computePartSize(stats.Size())
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

	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(path),
		Body:   content,
	})

	if err != nil {
		if aerr, ok := err.(smithy.APIError); ok {
			// TODO better error handling
			errChan[0] <- aerr
			return aerr
		}
		errChan[0] <- err
		return err
	}

	return nil

	// sess.Config.S3DisableContentMD5Validation = aws.Bool(true)
	// ps, e := computePartSize(stats.Size())
	// if e != nil {
	// 	if errChan != nil {
	// 		errChan[0] <- e
	// 	}
	// 	return e
	// }

	// uploader := s3manager.NewUploader(sess, func(u *s3manager.Uploader) {
	// 	u.PartSize = ps
	// 	u.Concurrency = common.UploadPartsConcurrency
	// 	u.RequestOptions = []request.Option{func(r *request.Request) {

	// 		if RefreshAndStoreIfRequired(DefaultConfig) {
	// 			// We must explicitely tell the uploader that the token has been refreshed
	// 			sess.Config.Credentials.Expire()
	// 			// s3Config := getS3ConfigFromSdkConfig(DefaultConfig)
	// 			// if testClient, e := s3transport.GetClient(DefaultConfig.SdkConfig, &s3Config); e == nil {
	// 			// 	r.Config.WithCredentials(testClient.Config.Credentials)
	// 			// }
	// 		}
	// 	}}
	// })

	// input := &s3manager.UploadInput{
	// 	Body:   aws.ReadSeekCloser(content),
	// 	Bucket: aws.String(bucketName),
	// 	Key:    aws.String(path),
	// }

	// if !common.UploadSkipMD5 && stats.Size() > (5*1024*1024*1024) {
	// 	h := md5.New()
	// 	if _, err := io.Copy(h, content); err != nil {
	// 		return fmt.Errorf("could not copy md5: %v", err)
	// 	}
	// 	input.Metadata = map[string]*string{"content-md5": aws.String(fmt.Sprintf("%x", h.Sum(nil)))}
	// }

	// _, _ = content.Seek(0, io.SeekStart)

	// _, err = uploader.Upload(input)
	// if err != nil {

	// 	// FIXME
	// 	// if aerr, ok := err.(awserr.Error); ok {
	// 	// 	errChan[0] <- aerr
	// 	// 	return aerr
	// 	// }
	// 	errChan[0] <- err
	// 	return err
	// }
	// return nil
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
