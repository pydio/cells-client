package rest

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/shibukawa/configdir"

	cells_sdk "github.com/pydio/cells-sdk-go"
	"github.com/pydio/cells-sdk-go/client"
	"github.com/pydio/cells-sdk-go/transport"
)

var (
	DefaultConfig   *cells_sdk.SdkConfig
	DefaultS3Config *cells_sdk.S3Config

	// Keys to retrieve configuration via environment variables
	KeyURL, KeyClientKey, KeyClientSecret, KeyUser, KeyPassword, KeySkipVerify = "TARGET_URL", "TARGET_CLIENT_KEY", "TARGET_CLIENT_SECRET", "TARGET_USER", "TARGET_PASSWORD", "TARGET_SKIP_VERIFY"

	// Keys to retrieve environment variables to configure a connection to Pydio Cells S3 API
	KeyS3Endpoint, KeyS3Region, KeyS3Bucket, KeyS3ApiKey, KeyS3ApiSecret, KeyS3UsePydioSpecificHeader, KeyS3IsDebug = "TARGET_S3_ENDPOINT", "TARGET_S3_REGION", "TARGET_S3_BUCKET", "TARGET_S3_API_KEY", "TARGET_S3_API_SECRET", "TARGET_S3_USE_PYDIO_SPECIFIC_HEADER", "TARGET_S3_IS_DEBUG"
)

func DefaultConfigFilePath() string {

	vendor := "Pydio"
	if runtime.GOOS == "linux" {
		vendor = "pydio"
	}
	appName := "cells-client"
	configDirs := configdir.New(vendor, appName)
	folders := configDirs.QueryFolders(configdir.Global)
	if len(folders) == 0 {
		folders = configDirs.QueryFolders(configdir.Local)
	}
	f := folders[0].Path
	if err := os.MkdirAll(f, 0777); err != nil {
		log.Fatal("Could not create local data dir - please check that you have the correct permissions for the folder -", f)
	}

	f = filepath.Join(f, "config.json")

	return f
}

// GetApiClient connects to the Pydio Cells server defined by this config, by sending an authentication
// request to the OIDC service to get a valid JWT (or taking the JWT from cache).
// It also returns a context to be used in subsequent requests.
func GetApiClient(anonymous ...bool) (context.Context, *client.PydioCellsRest, error) {

	anon := false
	if len(anonymous) > 0 && anonymous[0] {
		anon = true
	}

	c, t, e := transport.GetRestClientTransport(DefaultConfig, anon)
	if e != nil {
		return nil, nil, e
	}
	cl := client.New(t, strfmt.Default)
	return c, cl, nil

}

// SetUpEnvironment retrieves parameters and stores them in the DefaultConfig of the SDK.
// configFilePath and s3ConfigFilePath can be <nil> if the parameters are defined via env variables.
func SetUpEnvironment(configFilePath string, s3ConfigFilePath ...string) error {

	c, err := getSdkConfigFromEnv()
	if err != nil {
		return err
	}

	if c.Url == "" {

		if configFilePath == "" {
			configFilePath = DefaultConfigFilePath()
		}
		s, err := ioutil.ReadFile(configFilePath)
		if err != nil {
			return err
		}
		err = json.Unmarshal(s, &c)
		if err != nil {
			return err
		}
	}

	// Store the retrieved parameters in a public static singleton
	DefaultConfig = &c

	cs3, err := getS3ConfigFromEnv()
	if err != nil {
		return err
	}

	if cs3.Bucket == "" && len(s3ConfigFilePath) > 0 {
		s, err := ioutil.ReadFile(s3ConfigFilePath[0])
		if err == nil {
			json.Unmarshal(s, &cs3)
		}
	}

	// Build S3 config directly
	if cs3.Bucket == "" {
		cs3 = getS3ConfigFromSdkConfig(c)
	}

	// Store the retrieved parameters in a public static singleton
	DefaultS3Config = &cs3

	return nil
}

func getSdkConfigFromEnv() (cells_sdk.SdkConfig, error) {

	var c cells_sdk.SdkConfig

	// Check presence of environment variables
	url := os.Getenv(KeyURL)
	clientKey := os.Getenv(KeyClientKey)
	clientSecret := os.Getenv(KeyClientSecret)
	user := os.Getenv(KeyUser)
	password := os.Getenv(KeyPassword)
	skipVerifyStr := os.Getenv(KeySkipVerify)
	if skipVerifyStr == "" {
		skipVerifyStr = "false"
	}
	skipVerify, err := strconv.ParseBool(skipVerifyStr)
	if err != nil {
		return c, err
	}

	if !(len(url) > 0 && len(clientKey) > 0 && len(clientSecret) > 0 && len(user) > 0 && len(password) > 0) {
		return c, nil
	}

	c.Url = url
	c.ClientKey = clientKey
	c.ClientSecret = clientSecret
	c.User = user
	c.Password = password
	c.SkipVerify = skipVerify

	// Note: this cannot be set via env variable. Enhance?
	c.UseTokenCache = true

	return c, nil
}

func getS3ConfigFromSdkConfig(sConf cells_sdk.SdkConfig) cells_sdk.S3Config {
	var c cells_sdk.S3Config
	c.Bucket = "io"
	c.ApiKey = "gateway"
	c.ApiSecret = "gatewaysecret"
	c.UsePydioSpecificHeader = false
	c.IsDebug = false
	c.Region = "us-east-1"
	c.Endpoint = sConf.Url
	return c
}

func getS3ConfigFromEnv() (cells_sdk.S3Config, error) {

	var c cells_sdk.S3Config

	// check presence of Env variable
	endpoint := os.Getenv(KeyS3Endpoint)
	region := os.Getenv(KeyS3Region)
	bucket := os.Getenv(KeyS3Bucket)
	apiKey := os.Getenv(KeyS3ApiKey)
	apiSecret := os.Getenv(KeyS3ApiSecret)
	usePSHStr := os.Getenv(KeyS3UsePydioSpecificHeader)
	if usePSHStr == "" {
		usePSHStr = "false"
	}
	usePSH, err := strconv.ParseBool(usePSHStr)
	if err != nil {
		return c, err
	}

	isDebugStr := os.Getenv(KeyS3IsDebug)
	if isDebugStr == "" {
		isDebugStr = "false"
	}
	isDebug, err := strconv.ParseBool(isDebugStr)
	if err != nil {
		return c, err
	}

	if !(len(endpoint) > 0 && len(region) > 0 && len(bucket) > 0 && len(apiKey) > 0 && len(apiSecret) > 0) {
		return c, nil
	}

	c.Endpoint = endpoint
	c.Region = region
	c.Bucket = bucket
	c.ApiKey = apiKey
	c.ApiSecret = apiSecret
	c.UsePydioSpecificHeader = usePSH
	c.IsDebug = isDebug

	return c, nil
}

func RetryCallback(callback func() error, number int, interval time.Duration) error {

	var e error
	for i := 0; i < number; i++ {
		if e = callback(); e == nil {
			break
		}
		if i < number-1 {
			<-time.After(interval)
		}
	}

	return e
}
