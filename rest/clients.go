package rest

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	openapiruntime "github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/shibukawa/configdir"

	cells_sdk "github.com/pydio/cells-sdk-go/v5"
	"github.com/pydio/cells-sdk-go/v5/client"
	"github.com/pydio/cells-sdk-go/v5/transport"
	sdk_rest "github.com/pydio/cells-sdk-go/v5/transport/rest"
	sdk_s3 "github.com/pydio/cells-sdk-go/v5/transport/s3"

	"github.com/pydio/cells-client/v4/common"
)

var (
	// DefaultConfig  stores the current active config, we must initiliase it to avoid nil panic dereference
	DefaultConfig    *CecConfig
	DefaultContext   context.Context
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

func userAgent() string {
	return common.AppName + "/" + common.Version
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

func GetS3Client() (*s3.Client, string, error) {

	DefaultConfig.CustomHeaders = map[string]string{
		transport.UserAgentKey: userAgent(),
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

// GetFrom performs an authenticated GET request for the passed URI (that must start with a '/').
func GetFrom(config *CecConfig, uri string) (*http.Response, error) {
	currURL := config.Url + uri
	req, err := http.NewRequest("GET", currURL, nil)
	if err != nil {
		return nil, err
	}
	return AuthenticatedRequest(req, config.SdkConfig)
}

// AuthenticatedGet performs an authenticated GET request for the passed URI (that must start with a '/').
func AuthenticatedGet(uri string) (*http.Response, error) {

	currURL := DefaultConfig.Url + uri
	req, err := http.NewRequest("GET", currURL, nil)
	if err != nil {
		return nil, err
	}
	return AuthenticatedRequest(req, DefaultConfig.SdkConfig)
}

// AuthenticatedRequest performs the passed request after adding an authorization Header.
func AuthenticatedRequest(req *http.Request, sdkConfig *cells_sdk.SdkConfig) (*http.Response, error) {

	tp, e := transport.TokenProviderFromConfig(sdkConfig)
	if e != nil {
		return nil, e
	}

	httpClient := &http.Client{Transport: transport.New(
		transport.WithSkipVerify(sdkConfig.SkipVerify),
		transport.WithCustomHeaders(sdkConfig.CustomHeaders),
		transport.WithBearer(tp),
	)}

	resp, e := httpClient.Do(req)
	if e != nil {
		log.Println("... Authenticated request failed, cause:", e)
		return nil, e
	}
	return resp, nil
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

func getS3ConfigFromSdkConfig(sConf *CecConfig) *cells_sdk.S3Config {
	conf := cells_sdk.NewS3Config()
	conf.Endpoint = sConf.Url
	conf.RequestTimout = int(common.S3RequestTimeout)
	return conf
}
