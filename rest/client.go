package rest

import (
	"context"
	"fmt"
	"runtime"
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

const (
	TransferRetryMaxAttemptsDefault = 3
	TransferRetryMaxBackoffDefault  = time.Second * 3
)

var (
	UploadSwitchMultipart = int64(100)
	UploadDefaultPartSize = int64(50)
	UploadMaxPartsNumber  = int64(5000)

	TransferRetryMaxAttempts = TransferRetryMaxAttemptsDefault
	TransferRetryMaxBackoff  = TransferRetryMaxBackoffDefault

	UploadPartsSteps       = int64(10 * 1024 * 1024)
	UploadPartsConcurrency = 3
	UploadSkipMD5          = false
	S3RequestTimeout       = int64(-1)

	// defaultCellsStore holds a static singleton that ensure we only have *one* source of truth
	// to trigger OAuth refresh
	// TODO Make the cells store more clever to be able to launch more than one command in parallel from the same machine.
	//  In current state we might get issues with the refresh procedure
	defaultCellsStore cellsSdk.ConfigRefresher
	cellsStoreInit    = &sync.Once{}
)

func cellsStore() cellsSdk.ConfigRefresher {
	cellsStoreInit.Do(func() {
		defaultCellsStore = &CellsConfigStore{}
	})
	return defaultCellsStore
}

func UserAgent() string {
	osVersion := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	goVersion := runtime.Version()
	appVersion := fmt.Sprintf("github.com/pydio/%s@v%s", common.AppName, common.Version)
	return fmt.Sprintf("%s %s %s", osVersion, goVersion, appVersion)
}

// SdkClient wraps the APi Client and exposes convenient methods to be called by implementing layers.
type SdkClient struct {
	currentConfig *CecConfig
	configStore   cellsSdk.ConfigRefresher
	apiClient     *client.PydioCellsRestAPI
	s3Client      *s3.Client

	// stopRefreshChan enable stopping the OAuth auto refresh mechanism at teardown
	stopRefreshChan chan struct{}
}

// NewSdkClient creates a new client based on the given config.
// TODO It has the responsibility to do the token refresh procedure when needed in case of OAuth credentials.
func NewSdkClient(ctx context.Context, config *CecConfig) (*SdkClient, error) {

	t, err := sdkRest.GetApiTransport(config.SdkConfig, false)
	if err != nil {
		return nil, err
	}

	store := cellsStore()
	apiClient := client.New(t, strfmt.Default)
	s3Client, err := doGetS3Client(ctx, store, config.SdkConfig)
	if err != nil {
		return nil, err
	}

	return &SdkClient{
		currentConfig: config,
		configStore:   store,
		apiClient:     apiClient,
		s3Client:      s3Client,
	}, nil
}

// ConfigureS3Logger reset the current s3 client that is hold in the SDK client, adding a logger that is configured via the passed flags.
func (client *SdkClient) ConfigureS3Logger(ctx context.Context, s3Flags string) error {

	options := buildS3Options(client.configStore)
	if logOption := optionFromS3Flags(s3Flags); logOption != nil {
		options = append(options, logOption)
		// TODO handle error
	}

	if cfg, e := sdkS3.LoadConfig(ctx, client.currentConfig.SdkConfig, options...); e != nil {
		return e
	} else {
		client.s3Client = sdkS3.NewClientFromConfig(cfg, client.currentConfig.Url)
	}
	return nil
}

// Setup prepare the client after it has been created, especially refreshes the token in case of OAuth
func (client *SdkClient) Setup(ctx context.Context) {

	// Launch a "background thread" that call the refresh if and when necessary as long as the command runs.
	if client.currentConfig.AuthType == cellsSdk.AuthTypeOAuth {
		client.stopRefreshChan = make(chan struct{})
		// First call the refresh synchronously
		if _, err := client.GetStore().RefreshIfRequired(ctx, client.currentConfig.SdkConfig); err != nil {
			Log.Fatalf("login failed for %s, cause: %s", id(client.currentConfig.SdkConfig), err)
		}
		go func() {
			ticker := time.NewTicker(20 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					Log.Debugf("About to check refreshment for %s", id(client.currentConfig.SdkConfig))
					if _, err := client.GetStore().RefreshIfRequired(ctx, client.currentConfig.SdkConfig); err != nil {
						Log.Errorf("could not refresh authentication token: %s", err)
						close(client.stopRefreshChan)
					}
				case <-client.stopRefreshChan:
					Log.Debugln("Stopping refresh daemon")
					return
				}
			}
		}()
	}
}

// Teardown clean resources before terminating.
func (client *SdkClient) Teardown() {
	if client.stopRefreshChan != nil {
		close(client.stopRefreshChan)
	}
}

// GetConfig simply exposes the current SdkConfig
func (client *SdkClient) GetConfig() *cellsSdk.SdkConfig {
	return client.currentConfig.SdkConfig
}

// GetCecConfig simply exposes the current CecConfig
func (client *SdkClient) GetCecConfig() *CecConfig {
	return client.currentConfig
}

// GetStore simply exposes the store that centralize credentials (and performs OAuth refresh).
func (client *SdkClient) GetStore() cellsSdk.ConfigRefresher {
	return client.configStore
}

// GetApiClient simply exposes the Cells REST API client that is hold by the current SDKClient.
func (client *SdkClient) GetApiClient() *client.PydioCellsRestAPI {
	return client.apiClient
}

// GetS3Client simply exposes the S3 client that is hold by the current SDKClient.
func (client *SdkClient) GetS3Client() *s3.Client {
	return client.s3Client
}

// GetBucketName returns the default buck name to be used with the s3 client.
func (client *SdkClient) GetBucketName() string {
	return cellsSdk.DefaultS3Bucket
}

// doGetS3Client creates a new S3 client based on the given config to transfer files to/from a distant Cells server.
func doGetS3Client(ctx context.Context, configStore cellsSdk.ConfigRefresher, conf *cellsSdk.SdkConfig) (*s3.Client, error) {
	options := buildS3Options(configStore)
	//if logOption := configureLogMode(); logOption != nil {
	//	options = append(options, logOption)
	//}
	if cfg, e := sdkS3.LoadConfig(ctx, conf, options...); e != nil {
		return nil, e
	} else {
		return sdkS3.NewClientFromConfig(cfg, conf.Url), nil
	}
}

func buildS3Options(configStore cellsSdk.ConfigRefresher) []interface{} {
	var options []interface{}
	options = append(options, sdkS3.WithCellsConfigStore(configStore))

	if int(S3RequestTimeout) > 0 {
		to := time.Duration(int(S3RequestTimeout)) * time.Second
		options = append(options, sdkHttp.WithTimout(to))
	}

	if TransferRetryMaxBackoff != TransferRetryMaxBackoffDefault ||
		TransferRetryMaxAttempts != TransferRetryMaxAttemptsDefault {
		// TODO finalize addition of extra error codes that must be seen as "retry-able"
		options = append(
			options,
			sdkS3.WithCustomRetry(
				TransferRetryMaxAttempts,
				TransferRetryMaxBackoff,
				"ClientDisconnected",
			),
		)
	}
	return options
}

func optionFromS3Flags(s3Flags string) cellsSdk.AwsConfigOption {

	// input := "Retries | Signing | Response | "
	// For the record was:
	// - verbose: 	aws.LogRetries | aws.LogRequest | aws.LogSigning
	//	- very verbose: aws.LogSigning | aws.LogResponseEventMessage |
	//			aws.LogRetries | aws.LogRequest | aws.LogResponse |
	//			aws.LogDeprecatedUsage | aws.LogRequestEventMessage
	// 2 more: aws.LogRequestWithBody | aws.LogResponseWithBody

	if s3Flags == "" {
		return nil
	}
	logMode := getLogMode(s3Flags)
	return sdkS3.WithLogger(printLnWriter{}, logMode)
}

//// TODO Work in progress: finalize and clean
//func configureLogMode() cellsSdk.AwsConfigOption {
//	switch common.CurrentLogLevel {
//	case common.Info:
//		return nil
//	case common.Debug:
//		logMode := aws.LogRetries | aws.LogRequest // | aws.LogSigning
//		return sdkS3.WithLogger(printLnWriter{}, logMode)
//	case common.Trace:
//		logMode := aws.LogSigning |
//			aws.LogRetries | aws.LogRequest | aws.LogResponse |
//			aws.LogDeprecatedUsage | aws.LogRequestEventMessage | aws.LogResponseEventMessage
//		return sdkS3.WithLogger(printLnWriter{}, logMode)
//	default:
//		log.Fatal("unsupported log level:", common.CurrentLogLevel)
//	}
//	return nil
//}

type printLnWriter struct{}

func (p printLnWriter) Write(data []byte) (n int, err error) {
	fmt.Println(string(data))
	return len(data), nil
}

func (p printLnWriter) Println(msg string) {
	fmt.Println(msg)
}
