package rest

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-openapi/strfmt"

	cellsSdk "github.com/pydio/cells-sdk-go/v5"
	"github.com/pydio/cells-sdk-go/v5/client"
	sdkHttp "github.com/pydio/cells-sdk-go/v5/transport/http"
	sdkRest "github.com/pydio/cells-sdk-go/v5/transport/rest"
	sdkS3 "github.com/pydio/cells-sdk-go/v5/transport/s3"

	"github.com/pydio/cells-client/v4/common"
)

var (
	// defaultCellsStore holds a static singleton that ensure we only have *one* source of truth
	// to trigger OAuth refresh
	// TODO make it more clever to be able to launch more than one command in parallel from the same machine.
	defaultCellsStore cellsSdk.ConfigRefresher
	cellsStoreInit    = &sync.Once{}
)

func cellsStore() cellsSdk.ConfigRefresher {
	cellsStoreInit.Do(func() {
		defaultCellsStore = &CellsConfigStore{}
	})
	return defaultCellsStore
}

// SdkClient wraps the APi Client and exposes convenient methods to be called by implementing layers.
type SdkClient struct {
	currentConfig *cellsSdk.SdkConfig
	configStore   cellsSdk.ConfigRefresher
	apiClient     *client.PydioCellsRestAPI
	s3Client      *s3.Client
}

// NewSdkClient creates a new client based on the given config.
// TODO It has the responsibility to do the token refresh procedure when needed in case of OAuth credentials.
func NewSdkClient(ctx context.Context, sdkConfig *cellsSdk.SdkConfig) (*SdkClient, error) {

	t, err := sdkRest.GetApiTransport(sdkConfig, false)
	if err != nil {
		return nil, err
	}

	store := cellsStore()
	apiClient := client.New(t, strfmt.Default)
	s3Client, err := doGetS3Client(ctx, store, sdkConfig)
	if err != nil {
		return nil, err
	}

	return &SdkClient{
		currentConfig: sdkConfig,
		configStore:   store,
		apiClient:     apiClient,
		s3Client:      s3Client,
	}, nil
}

// GetConfig simply exposes the current SdkConfig
func (fx *SdkClient) GetConfig() *cellsSdk.SdkConfig {
	return fx.currentConfig
}

// GetStore simply exposes the store that centralize credentials (and performs OAuth refresh).
func (fx *SdkClient) GetStore() cellsSdk.ConfigRefresher {
	return fx.configStore
}

// GetApiClient simply exposes the Cells REST API client that is hold by the current SDKClient.
func (fx *SdkClient) GetApiClient() *client.PydioCellsRestAPI {
	return fx.apiClient
}

// GetS3Client simply exposes the S3 client that is hold by the current SDKClient.
func (fx *SdkClient) GetS3Client() *s3.Client {
	return fx.s3Client
}

// GetBucketName returns the default buck name to be used with the s3 client.
func (fx *SdkClient) GetBucketName() string {
	return cellsSdk.DefaultS3Bucket
}

// doGetS3Client creates a new S3 client based on the given config to transfer files to/from a distant Cells server.
func doGetS3Client(ctx context.Context, configStore cellsSdk.ConfigRefresher, conf *cellsSdk.SdkConfig) (*s3.Client, error) {
	var options []interface{}
	options = append(options, sdkS3.WithCellsConfigStore(configStore))

	if int(common.S3RequestTimeout) > 0 {
		to := time.Duration(int(common.S3RequestTimeout)) * time.Second
		options = append(options, sdkHttp.WithTimout(to))
	}

	if logOption := configureLogMode(); logOption != nil {
		options = append(options, logOption)
	}

	if common.TransferRetryMaxBackoff != common.TransferRetryMaxBackoffDefault ||
		common.TransferRetryMaxAttempts != common.TransferRetryMaxAttemptsDefault {
		// TODO finalize addition of extra error codes that must be seen as "retry-able"
		options = append(
			options,
			sdkS3.WithCustomRetry(
				common.TransferRetryMaxAttempts,
				common.TransferRetryMaxBackoff,
				"ClientDisconnected",
			),
		)
	}

	cfg, e := sdkS3.LoadConfig(ctx, conf, options...)
	if e != nil {
		return nil, e
	}

	return sdkS3.NewClientFromConfig(cfg, conf.Url), nil
}

//// GetS3Client creates a new default S3 client based on current active config
//// to transfer files to/from a distant Cells server.
//func (fx *SdkClient) GetS3Client(ctx context.Context) *s3.Client {
//
//	var err error
//	fx.initApiClientLock.Do(func() {
//		currentClient, err = fx.doGetS3Client(ctx, DefaultConfig.SdkConfig)
//	})
//	if err != nil {
//		fmt.Println("Unexpected error:", err.Error())
//		fmt.Println(" -> Could not initialise s3 client, aborting.")
//		os.Exit(1)
//	}
//
//	// For the time being, we assume that the bucket used is always the same
//	return s3Client, cellsSdk.DefaultS3Bucket, e
//}
//
//
//s3Client *s3.Client,
//bucketName string,
