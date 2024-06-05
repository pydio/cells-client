package rest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/dustin/go-humanize"

	sdkS3 "github.com/pydio/cells-sdk-go/v5/transport/s3"
)

// GetFile retrieves a file from the server in one big download (**no** multipart download for the time being).
func (client *SdkClient) GetFile(ctx context.Context, pathToFile string) (io.Reader, int, error) {
	hO, err := client.GetS3Client().HeadObject(
		ctx,
		&s3.HeadObjectInput{
			Bucket: aws.String(client.GetBucketName()),
			Key:    aws.String(pathToFile),
		},
	)
	if err != nil {
		return nil, 0, err
	}

	obj, err := client.GetS3Client().GetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(client.GetBucketName()),
			Key:    aws.String(pathToFile),
		},
	)
	if err != nil {
		return nil, 0, err
	}
	return obj.Body, int(*hO.ContentLength), nil
}

// PutFile upload a local file to the server without using multipart upload.
func (client *SdkClient) PutFile(
	ctx context.Context,
	pathToFile string,
	content io.ReadSeeker,
	checkExists bool,
	errChan ...chan error,
) (*s3.PutObjectOutput, error) {

	key := pathToFile
	var obj *s3.PutObjectOutput
	err := RetryCallback(func() error {
		var tmpErr error
		obj, tmpErr = client.GetS3Client().PutObject(
			ctx,
			&s3.PutObjectInput{
				Bucket: aws.String(client.GetBucketName()),
				Key:    aws.String(pathToFile),
				Body:   content,
			},
		)
		return tmpErr
	}, 5, 2*time.Second)
	if err != nil {
		errMsg := fmt.Errorf("could not put object in bucket %s with key %s, \ncause: %s", client.GetBucketName(), key, err.Error())
		if len(errChan) > 0 {
			errChan[0] <- errMsg
		}
		return nil, errMsg
	}

	if checkExists {
		fmt.Println(" ## Waiting for file to be indexed...")
		// Now stat Node to make sure it is indexed
		err = RetryCallback(func() error {
			_, ok := client.StatNode(ctx, pathToFile)
			if !ok {
				return fmt.Errorf("could not stat node after PutFile operation")
			}
			return nil
		}, 5, 3*time.Second)
		if err != nil {
			errMsg := fmt.Errorf("existence check failed for %s in bucket %s\ntimeout after 15s, last error: %s", key, client.GetBucketName(), err.Error())
			if len(errChan) > 0 {
				errChan[0] <- errMsg
			}
			return nil, errMsg
		}
		fmt.Println(" ## File has been indexed")
	}
	return obj, nil
}

func (client *SdkClient) s3Upload(ctx context.Context, path string,
	content io.ReadSeeker, fSize int64, verbose bool, errChan ...chan error) error {

	ps, err := sdkS3.ComputePartSize(fSize, UploadDefaultPartSize, UploadMaxPartsNumber)
	if err != nil {
		if errChan != nil {
			errChan[0] <- err
		}
		return err
	}
	numParts := int(math.Ceil(float64(fSize) / float64(ps)))
	if verbose {
		Log.Infof("Multipart upload for %s", path)
		Log.Infof("\tSize: %s", humanize.IBytes(uint64(fSize)))
		Log.Infof("\tPart Size: %s", humanize.IBytes(uint64(ps)))
		Log.Infof("\tNumber of parts: %d", numParts)
	}

	uploader := manager.NewUploader(client.GetS3Client(),
		func(u *manager.Uploader) {
			u.Concurrency = UploadPartsConcurrency
			u.PartSize = ps
		},
	)

	// Adds a callback entry point so that we can follow the effective part upload.
	uploader.BufferProvider = sdkS3.NewCallbackTransferProvider(path, fSize, ps, numParts, verbose)

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(client.GetBucketName()),
		Key:    aws.String(path),
		Body:   content,
	})

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			// TODO better error handling
			if errChan != nil {
				errChan[0] <- apiErr
			}
			return apiErr
		}
		if errChan != nil {
			errChan[0] <- err
		}
		return err
	}
	return nil
}
