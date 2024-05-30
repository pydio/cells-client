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

	"github.com/pydio/cells-client/v4/common"
)

// GetFile retrieves a file from the server in one big download (**no** multipart download for the time being).
func (fx *SdkClient) GetFile(ctx context.Context, pathToFile string) (io.Reader, int, error) {
	hO, err := fx.GetS3Client().HeadObject(
		ctx,
		&s3.HeadObjectInput{
			Bucket: aws.String(fx.GetBucketName()),
			Key:    aws.String(pathToFile),
		},
	)
	if err != nil {
		return nil, 0, err
	}

	obj, err := fx.GetS3Client().GetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(fx.GetBucketName()),
			Key:    aws.String(pathToFile),
		},
	)
	if err != nil {
		return nil, 0, err
	}
	return obj.Body, int(*hO.ContentLength), nil
}

// PutFile upload a local file to the server without using multipart upload.
func (fx *SdkClient) PutFile(
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
		obj, tmpErr = fx.GetS3Client().PutObject(
			ctx,
			&s3.PutObjectInput{
				Bucket: aws.String(fx.GetBucketName()),
				Key:    aws.String(pathToFile),
				Body:   content,
			},
		)
		return tmpErr
	}, 5, 2*time.Second)
	if err != nil {
		errMsg := fmt.Errorf("could not put object in bucket %s with key %s, \ncause: %s", fx.GetBucketName(), key, err.Error())
		if len(errChan) > 0 {
			errChan[0] <- errMsg
		}
		return nil, errMsg
	}

	if checkExists {
		fmt.Println(" ## Waiting for file to be indexed...")
		// Now stat Node to make sure it is indexed
		err = RetryCallback(func() error {
			_, ok := fx.StatNode(ctx, pathToFile)
			if !ok {
				return fmt.Errorf("could not stat node after PutFile operation")
			}
			return nil
		}, 5, 3*time.Second)
		if err != nil {
			errMsg := fmt.Errorf("existence check failed for %s in bucket %s\ntimeout after 15s, last error: %s", key, fx.GetBucketName(), err.Error())
			if len(errChan) > 0 {
				errChan[0] <- errMsg
			}
			return nil, errMsg
		}
		fmt.Println(" ## File has been indexed")
	}
	return obj, nil
}

func (fx *SdkClient) s3Upload(ctx context.Context, path string,
	content io.ReadSeeker, fSize int64, verbose bool, errChan ...chan error) error {

	ps, err := sdkS3.ComputePartSize(fSize, common.UploadDefaultPartSize, common.UploadMaxPartsNumber)
	if err != nil {
		if errChan != nil {
			errChan[0] <- err
		}
		return err
	}
	if verbose {
		Log.Infof("... Launching upload for %s", path)
		numParts := math.Ceil(float64(fSize) / float64(ps))
		fmt.Println("\tSize:", humanize.Bytes(uint64(fSize)))
		fmt.Println("\tPart Size:", humanize.Bytes(uint64(ps)))
		fmt.Println("\tNumber of parts:", numParts)
	}

	uploader := manager.NewUploader(fx.GetS3Client(),
		func(u *manager.Uploader) {
			u.Concurrency = common.UploadPartsConcurrency
			u.PartSize = ps
		},
	)

	// Adds a callback entry point so that we can follow the effective part upload.
	uploader.BufferProvider = sdkS3.NewCallbackTransferProvider(path, fSize, ps)

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(fx.GetBucketName()),
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
