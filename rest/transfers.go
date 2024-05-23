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
	"github.com/gosuri/uiprogress"

	sdkS3 "github.com/pydio/cells-sdk-go/v5/transport/s3"

	"github.com/pydio/cells-client/v4/common"
)

// GetFile retrieves a file from the server in one big download (**no** multipart download for the time being).
func GetFile(ctx context.Context,
	s3Client *s3.Client,
	bucketName string,
	pathToFile string,
) (io.Reader, int, error) {
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

// PutFile upload a local file to the server without using multipart upload.
func PutFile(
	ctx context.Context,
	s3Client *s3.Client,
	bucketName string,
	pathToFile string,
	content io.ReadSeeker,
	checkExists bool,
	errChan ...chan error,
) (*s3.PutObjectOutput, error) {

	key := pathToFile
	var obj *s3.PutObjectOutput
	err := RetryCallback(func() error {
		var tmpErr error
		obj, tmpErr = s3Client.PutObject(
			ctx,
			&s3.PutObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(pathToFile),
				Body:   content,
			},
		)
		return tmpErr
	}, 5, 2*time.Second)
	if err != nil {
		errMsg := fmt.Errorf("could not put object in bucket %s with key %s, \ncause: %s", bucketName, key, err.Error())
		if len(errChan) > 0 {
			errChan[0] <- errMsg
		}
		return nil, errMsg
	}

	if checkExists {
		fmt.Println(" ## Waiting for file to be indexed...")
		// Now stat Node to make sure it is indexed
		err = RetryCallback(func() error {
			_, ok := StatNode(ctx, pathToFile)
			if !ok {
				return fmt.Errorf("could not stat node after PutFile operation")
			}
			return nil
		}, 5, 3*time.Second)
		if err != nil {
			errMsg := fmt.Errorf("existence check failed for %s in bucket %s\ntimeout after 15s, last error: %s", key, bucketName, err.Error())
			if len(errChan) > 0 {
				errChan[0] <- errMsg
			}
			return nil, errMsg
		}
		fmt.Println(" ## File has been indexed")
	}
	return obj, nil
}

type BarsPool struct {
	*uiprogress.Progress
	showGlobal bool
	nodesBar   *uiprogress.Bar
}

func NewBarsPool(showGlobal bool, totalNodes int, refreshInterval time.Duration) *BarsPool {
	b := &BarsPool{}
	b.Progress = uiprogress.New()
	b.Progress.SetRefreshInterval(refreshInterval)
	b.showGlobal = showGlobal
	if showGlobal { // we are transferring more than one file
		b.nodesBar = b.AddBar(totalNodes)
		b.nodesBar.PrependCompleted()
		b.nodesBar.AppendFunc(func(b *uiprogress.Bar) string {
			if b.Current() == b.Total {
				return fmt.Sprintf("Transferred %d/%d files and folders in %s.", b.Current(), b.Total, b.TimeElapsedString())
			} else {
				return fmt.Sprintf("Copying folders and files since %s: %d/%d", b.TimeElapsedString(), b.Current(), b.Total)
			}
		})
	}
	return b
}

func (b *BarsPool) Done() {
	if !b.showGlobal {
		return
	}
	b.nodesBar.Incr()
	if b.nodesBar.Current() == b.nodesBar.Total {
		// Finished, remove all bars
		b.Bars = []*uiprogress.Bar{b.nodesBar}
	}
}

func (b *BarsPool) Get(i int, total int, name string) *uiprogress.Bar {
	idx := i % PoolSize
	var nBars []*uiprogress.Bar
	if b.showGlobal {
		idx++
		nBars = append(nBars, b.nodesBar)
	}
	// Remove old bar
	for k, bar := range b.Bars {
		if k == idx || (b.showGlobal && bar == b.nodesBar) {
			continue
		}
		nBars = append(nBars, bar)
	}
	b.Bars = nBars
	bar := b.AddBar(total)
	bar.PrependCompleted()
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprint(name)
	})
	return bar
}

func s3Upload(ctx context.Context, s3Client *s3.Client, bucketName string, path string,
	content io.ReadSeeker, fSize int64, verbose bool, errChan ...chan error) error {

	ps, err := sdkS3.ComputePartSize(fSize, common.UploadDefaultPartSize, common.UploadMaxPartsNumber)
	if err != nil {
		if errChan != nil {
			errChan[0] <- err
		}
		return err
	}
	if verbose {
		fmt.Println("... Launching upload for", path)
		numParts := math.Ceil(float64(fSize) / float64(ps))
		fmt.Println("\tSize:", humanize.Bytes(uint64(fSize)))
		fmt.Println("\tPart Size:", humanize.Bytes(uint64(ps)))
		fmt.Println("\tNumber of parts:", numParts)
	}

	uploader := manager.NewUploader(s3Client,
		func(u *manager.Uploader) {
			u.Concurrency = common.UploadPartsConcurrency
			u.PartSize = ps
		},
	)

	// Adds a callback entry point so that we can follow the effective part upload.
	uploader.BufferProvider = sdkS3.NewCallbackTransferProvider(path, fSize, ps)

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
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
