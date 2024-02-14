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
	"github.com/go-openapi/strfmt"
	"github.com/shibukawa/configdir"

	cellsSdk "github.com/pydio/cells-sdk-go/v5"
	"github.com/pydio/cells-sdk-go/v5/client"
	"github.com/pydio/cells-sdk-go/v5/transport"
	sdkHttp "github.com/pydio/cells-sdk-go/v5/transport/http"
	sdkRest "github.com/pydio/cells-sdk-go/v5/transport/rest"
	sdkS3 "github.com/pydio/cells-sdk-go/v5/transport/s3"

	"github.com/pydio/cells-client/v4/common"
)

var (
	// DefaultConfig  stores the current active config, we must initialise it to avoid nil panic dereference
	DefaultConfig *CecConfig
	// DefaultTransport openapiruntime.ClientTransport

	configFilePath string
	once           = &sync.Once{}
)

// CecConfig extends the default SdkConfig with custom parameters.
type CecConfig struct {
	*cellsSdk.SdkConfig
	Label            string `json:"label"`
	SkipKeyring      bool   `json:"skipKeyring"`
	CreatedAtVersion string `json:"createdAtVersion"`
}

// DefaultCecConfig simply creates a new configuration struct.
func DefaultCecConfig() *CecConfig {
	return &CecConfig{
		SdkConfig: &cellsSdk.SdkConfig{
			UseTokenCache: true,
			AuthType:      cellsSdk.AuthTypePat,
		},
		SkipKeyring: false,
	}
}

// GetApiClient returns a client to directly communicate with the Pydio Cells REST API.
// Requests are anonymous when corresponding flag is set. Otherwise, the authentication is managed
// by the client, using the current active SDKConfig to provide valid credentials.
func GetApiClient(customConf ...*cellsSdk.SdkConfig) (*client.PydioCellsRestAPI, error) {

	currConf := DefaultConfig.SdkConfig
	if len(customConf) == 1 {
		currConf = customConf[0]
	}
	return doGetApiClient(currConf, false)
}

func GetAnonymousApiClient(customConf ...*cellsSdk.SdkConfig) (*client.PydioCellsRestAPI, error) {
	currConf := DefaultConfig.SdkConfig
	if len(customConf) == 1 {
		currConf = customConf[0]
	}
	return doGetApiClient(currConf, true)
}

// by the client, using the current active SDKConfig to provide valid credentials.
func doGetApiClient(conf *cellsSdk.SdkConfig, anonymous bool) (*client.PydioCellsRestAPI, error) {
	t, err := sdkRest.GetApiTransport(conf, anonymous)
	if err != nil {
		return nil, err
	}
	return client.New(t, strfmt.Default), nil
}

// GetS3Client creates a new default S3 client based on current active config
// to transfer files to/from a distant Cells server.
func GetS3Client(ctx context.Context) (*s3.Client, string, error) {

	var options []interface{}

	if CellsStore == nil {
		fmt.Println("[WARNING] could not found a cells store")
	} else {
		options = append(options, sdkS3.WithCellsConfigStore(CellsStore))
	}

	if int(common.S3RequestTimeout) > 0 {
		to := time.Duration(int(common.S3RequestTimeout)) * time.Second
		options = append(options, sdkHttp.WithTimout(to))
	}

	if logOption := configureLogMode(); logOption != nil {
		options = append(options, logOption)
	}

	cfg, e := sdkS3.LoadConfig(ctx, DefaultConfig.SdkConfig, options...)
	if e != nil {
		return nil, "", e
	}

	s3Client := sdkS3.NewClientFromConfig(cfg, DefaultConfig.Url)

	// For the time being, we assume that the bucket used is always the same
	return s3Client, cellsSdk.DefaultS3Bucket, e
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

func UserAgent() string {
	osVersion := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	goVersion := fmt.Sprintf("Go_%s", runtime.Version())
	appVersion := fmt.Sprintf("%s/%s", common.AppName, common.Version)
	return fmt.Sprintf("%s; %s; %s", osVersion, goVersion, appVersion)
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
func authenticatedRequest(req *http.Request, sdkConfig *cellsSdk.SdkConfig) (*http.Response, error) {

	tp, e := transport.TokenProviderFromConfig(sdkConfig)
	if e != nil {
		return nil, e
	}

	httpClient := &http.Client{Transport: transport.New(
		sdkHttp.WithCustomHeaders(sdkConfig.CustomHeaders),
		sdkHttp.WithBearer(tp),
		sdkHttp.WithSkipVerify(sdkConfig.SkipVerify),
	)}

	resp, e := httpClient.Do(req)
	if e != nil {
		log.Println("... Authenticated request failed, cause:", e)
		return nil, e
	}
	return resp, nil
}

// TODO WiP: finalize and clean

func configureLogMode() cellsSdk.AwsConfigOption {
	switch common.CurrentLogLevel {
	case common.Info:
		return nil
	case common.Debug:
		logMode := aws.LogSigning | aws.LogRetries
		return sdkS3.WithLogger(printLnWriter{}, logMode)
	case common.Trace:
		logMode := aws.LogSigning | aws.LogRetries | aws.LogRequest | aws.LogResponse | aws.LogDeprecatedUsage | aws.LogRequestEventMessage | aws.LogResponseEventMessage
		return sdkS3.WithLogger(printLnWriter{}, logMode)
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
