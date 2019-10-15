package rest

import (
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/pydio/cells-client/common"
	"github.com/pydio/cells-sdk-go/client/tree_service"
	"github.com/pydio/cells-sdk-go/models"
	awstransport "github.com/pydio/cells-sdk-go/transport/aws"
)

func GetS3Client() (*s3.S3, string, error) {
	DefaultConfig.CustomHeaders = map[string]string{"User-Agent": "cells-client/" + common.Version}
	s3Config := getS3ConfigFromSdkConfig(*DefaultConfig)
	bucketName := s3Config.Bucket
	s3Client, e := awstransport.GetS3CLient(DefaultConfig, &s3Config)
	return s3Client, bucketName, e
}

func GetFile(pathToFile string) (io.Reader, error) {

	s3Client, bucketName, e := GetS3Client()
	if e != nil {
		return nil, e
	}
	_, err := s3Client.HeadObject((&s3.HeadObjectInput{}).
		SetBucket(bucketName).
		SetKey(pathToFile),
	)
	if err != nil {
		return nil, err
	}

	obj, err := s3Client.GetObject((&s3.GetObjectInput{}).
		SetBucket(bucketName).
		SetKey(pathToFile),
	)
	if err != nil {
		return nil, err
	}
	return obj.Body, nil
}

func PutFile(pathToFile string, content io.ReadSeeker, checkExists bool) (*s3.PutObjectOutput, error) {
	s3Client, bucketName, e := GetS3Client()
	if e != nil {
		return nil, e
	}

	var err error
	key := pathToFile
	var obj *s3.PutObjectOutput
	e = RetryCallback(func() error {
		obj, err = s3Client.PutObject((&s3.PutObjectInput{}).
			SetBucket(bucketName).
			SetKey(key).
			SetBody(content),
		)
		if err != nil {
			fmt.Println(" ## Trying to Put file:", key)
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
