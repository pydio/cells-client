package rest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	openapiruntime "github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/shibukawa/configdir"

	cells_sdk "github.com/pydio/cells-sdk-go/v5"
	"github.com/pydio/cells-sdk-go/v5/client"
	"github.com/pydio/cells-sdk-go/v5/transport"
	sdk_http "github.com/pydio/cells-sdk-go/v5/transport/http"
	sdk_rest "github.com/pydio/cells-sdk-go/v5/transport/rest"
	sdk_s3 "github.com/pydio/cells-sdk-go/v5/transport/s3"

	"github.com/pydio/cells-client/v4/common"
)

var (
	// DefaultConfig  stores the current active config, we must initialise it to avoid nil panic dereference
	DefaultConfig    *CecConfig
	DefaultTransport openapiruntime.ClientTransport
	configFilePath   string
	once             = &sync.Once{}
)

// CecConfig extends the default SdkConfig with custom parameters.
type CecConfig struct {
	*cells_sdk.SdkConfig
	Label            string `json:"label"`
	SkipKeyring      bool   `json:"skipKeyring"`
	CreatedAtVersion string `json:"createdAtVersion"`
}

// DefaultCecConfig simply creates a new configuration struct.
func DefaultCecConfig() *CecConfig {
	return &CecConfig{
		SdkConfig: &cells_sdk.SdkConfig{
			UseTokenCache: true,
			AuthType:      cells_sdk.AuthTypePat,
		},
		SkipKeyring: false,
	}
}

// GetApiClient returns a client to directly communicate with the Pydio Cells REST API.
// Requests are anonymous when corresponding flag is set. Otherwise, the authentication is managed
// by the client, using the current active SDKConfig to provide valid credentials.
func GetApiClient(anonymous ...bool) (*client.PydioCellsRestAPI, error) {

	anon := false
	if len(anonymous) > 0 && anonymous[0] {
		anon = true
	}
	DefaultConfig.CustomHeaders = map[string]string{transport.UserAgentKey: userAgent()}
	var err error
	once.Do(func() {
		currConf := DefaultConfig.SdkConfig
		DefaultTransport, err = sdk_rest.GetApiTransport(currConf, anon)
	})
	if err != nil {
		return nil, err
	}

	return client.New(DefaultTransport, strfmt.Default), nil
}

// GetS3Client creates a new default S3 client based on current active config
// to transfer files to/from a distant Cells server.
func GetS3Client(ctx context.Context) (*s3.Client, string, error) {

	DefaultConfig.CustomHeaders = map[string]string{
		transport.UserAgentKey: userAgent(),
	}

	//s3Conf := getS3ConfigFromSdkConfig(DefaultConfig)

	var options []interface{}

	if CellsStore == nil {
		fmt.Println("[WARNING] could not found a cells store")
	} else {
		options = append(options, sdk_s3.WithCellsConfigStore(CellsStore))
	}

	if int(common.S3RequestTimeout) > 0 {
		to := time.Duration(int(common.S3RequestTimeout)) * time.Second
		options = append(options, sdk_http.WithTimout(to))
	}

	if logOption := configureLogMode(); logOption != nil {
		options = append(options, logOption)
	}

	cfg, e := sdk_s3.LoadConfig(ctx, DefaultConfig.SdkConfig, options...)
	if e != nil {
		return nil, "", e
	}

	s3Client := sdk_s3.NewClientFromConfig(cfg, DefaultConfig.Url)

	// For the time being, we assume that the bucket used is always the same
	return s3Client, cells_sdk.DefaultS3Bucket, e
}

func GetConfigFilePath() string {
	if configFilePath != "" {
		return configFilePath
	}
	return DefaultConfigFilePath()
}

func SetConfigFilePath(confPath string) {
	configFilePath = confPath
}

func DefaultConfigDirPath() string {
	vendor := "Pydio"
	if runtime.GOOS == "linux" {
		vendor = "pydio"
	}
	configDirs := configdir.New(vendor, common.AppName)
	folders := configDirs.QueryFolders(configdir.Global)
	if len(folders) == 0 {
		folders = configDirs.QueryFolders(configdir.Local)
	}
	return folders[0].Path
}

func DefaultConfigFilePath() string {
	f := DefaultConfigDirPath()
	if err := os.MkdirAll(f, 0755); err != nil {
		log.Fatal("Could not create local data dir - please check that you have the correct permissions for the folder -", f)
	}
	return filepath.Join(f, common.DefaultConfigFileName)
}

func CloneConfig(from *CecConfig) *CecConfig {
	sdkClone := *from.SdkConfig
	conClone := *from
	conClone.SdkConfig = &sdkClone
	return &conClone
}

//func getS3ConfigFromSdkConfig(sConf *CecConfig) *cells_sdk.S3Config {
//	conf := cells_sdk.NewS3Config()
//	conf.Endpoint = sConf.Url
//	conf.RequestTimout = int(common.S3RequestTimeout)
//	return conf
//}

func userAgent() string {
	return common.AppName + "/" + common.Version
}

// getFrom performs an authenticated GET request for the passed URI (that must start with a '/').
func getFrom(config *CecConfig, uri string) (*http.Response, error) {
	currURL := config.Url + uri
	req, err := http.NewRequest("GET", currURL, nil)
	if err != nil {
		return nil, err
	}
	return authenticatedRequest(req, config.SdkConfig)
}

// authenticatedGet performs an authenticated GET request for the passed URI (that must start with a '/').
func authenticatedGet(uri string) (*http.Response, error) {
	currURL := DefaultConfig.Url + uri
	req, err := http.NewRequest("GET", currURL, nil)
	if err != nil {
		return nil, err
	}
	return authenticatedRequest(req, DefaultConfig.SdkConfig)
}

// authenticatedRequest performs the passed request after adding an authorization Header.
func authenticatedRequest(req *http.Request, sdkConfig *cells_sdk.SdkConfig) (*http.Response, error) {

	tp, e := transport.TokenProviderFromConfig(sdkConfig)
	if e != nil {
		return nil, e
	}

	httpClient := &http.Client{Transport: transport.New(
		sdk_http.WithCustomHeaders(sdkConfig.CustomHeaders),
		sdk_http.WithBearer(tp),
		sdk_http.WithSkipVerify(sdkConfig.SkipVerify),
	)}

	resp, e := httpClient.Do(req)
	if e != nil {
		log.Println("... Authenticated request failed, cause:", e)
		return nil, e
	}
	return resp, nil
}

// TODO WiP: finalize and clean

func configureLogMode() cells_sdk.AwsConfigOption {
	switch common.CurrentLogLevel {
	case common.Info:
		return nil
	case common.Debug:
		logMode := aws.LogSigning | aws.LogRetries
		return sdk_s3.WithLogger(printLnWriter{}, logMode)
	case common.Trace:
		logMode := aws.LogSigning | aws.LogRetries | aws.LogRequest | aws.LogResponse | aws.LogDeprecatedUsage | aws.LogRequestEventMessage | aws.LogResponseEventMessage
		return sdk_s3.WithLogger(printLnWriter{}, logMode)
	default:
		log.Fatal("unsupported log level:", common.CurrentLogLevel)
	}
	return nil
}

type printLnWriter struct{}

func (p printLnWriter) Write(data []byte) (n int, err error) {
	fmt.Println(string(data))
	return len(data), nil
}

func (p printLnWriter) Println(msg string) {
	fmt.Println(msg)
}
