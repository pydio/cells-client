package rest

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/go-openapi/strfmt"
	"github.com/shibukawa/configdir"

	"github.com/pydio/cells-client/v2/common"
	cells_sdk "github.com/pydio/cells-sdk-go"
	"github.com/pydio/cells-sdk-go/client"
	"github.com/pydio/cells-sdk-go/transport"
	sdk_http "github.com/pydio/cells-sdk-go/transport/http"
	"github.com/pydio/cells-sdk-go/transport/oidc"
)

var (
	// DefaultConfig  stores the current active config.
	DefaultConfig  *CecConfig
	configFilePath string
)

// CecConfig extends the default SdkConfig with custom parameters.
type CecConfig struct {
	cells_sdk.SdkConfig
	SkipKeyring bool
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
	DefaultConfig.CustomHeaders = map[string]string{"User-Agent": "cells-client/" + common.Version}
	c, t, e := transport.GetRestClientTransport(&DefaultConfig.SdkConfig, anon)
	if e != nil {
		return nil, nil, e
	}
	cl := client.New(t, strfmt.Default)
	return c, cl, nil

}

// AuthenticatedGet performs an authenticated GET request for the passed URI (that must start with a '/')
func AuthenticatedGet(uri string) (*http.Response, error) {

	currURL := DefaultConfig.SdkConfig.Url + uri
	req, err := http.NewRequest("GET", currURL, nil)
	if err != nil {
		return nil, err
	}

	token, err := oidc.RetrieveToken(&DefaultConfig.SdkConfig)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	httpClient := sdk_http.GetHttpClient(&DefaultConfig.SdkConfig)
	return httpClient.Do(req)
}

// SetUpEnvironment retrieves parameters and stores them in the DefaultConfig of the SDK.
// It also puts the sensible info in the server keyring if one is present.
// Note the precedence order (for each start of the app):
//  1) environment variables,
//  2) config files whose path is passed as argument of the start command
//  3) local config file (that are generated at first start with one of the 2 options above OR by calling the configure command.
func SetUpEnvironment(confPath string) error {
	// Use a config file
	if confPath != "" {
		SetConfigFilePath(confPath)
	}

	// Get config params from environment variables
	c, err := getSdkConfigFromEnv()
	if err != nil {
		return err
	}

	if c.Url == "" {

		confPath = GetConfigFilePath()
		s, err := ioutil.ReadFile(confPath)
		if err != nil {
			return err
		}
		err = json.Unmarshal(s, &c)
		if err != nil {
			return err
		}
		// Retrieves sensible info from the keyring if one is present
		ConfigFromKeyring(&c)

		// Refresh token if required
		if refreshed, err := RefreshIfRequired(&c); refreshed {
			if err != nil {
				log.Fatal("Could not refresh authentication token:", err)
			}
			// Copy config as IdToken will be cleared
			storeConfig := c
			ConfigToKeyring(&storeConfig)
			// Save config to renew TokenExpireAt
			confData, _ := json.Marshal(&storeConfig)
			ioutil.WriteFile(confPath, confData, 0666)
		}
	}

	// Store the retrieved parameters in a public static singleton
	DefaultConfig = &c

	return nil
}

var refreshMux = &sync.Mutex{}

func RefreshAndStoreIfRequired(c *CecConfig) bool {
	refreshMux.Lock()
	defer refreshMux.Unlock()

	refreshed, err := RefreshIfRequired(c)
	if err != nil {
		log.Fatal("Could not refresh authentication token:", err)
	}
	if refreshed {
		// Copy config as IdToken will be cleared
		storeConfig := *c
		ConfigToKeyring(&storeConfig)
		// Save config to renew TokenExpireAt
		confData, _ := json.Marshal(&storeConfig)
		ioutil.WriteFile(GetConfigFilePath(), confData, 0600)
	}

	return refreshed
}

func getSdkConfigFromEnv() (CecConfig, error) {

	// var c CecConfig
	c := new(CecConfig)

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
		return *c, err
	}

	// Client Key and Client Secret are not used anymore
	// if !(len(url) > 0 && len(clientKey) > 0 && len(clientSecret) > 0 && len(user) > 0 && len(password) > 0) {
	if !(len(url) > 0 && len(user) > 0 && len(password) > 0) {
		return *c, nil
	}

	c.Url = url
	c.ClientKey = clientKey
	c.ClientSecret = clientSecret
	c.User = user
	c.Password = password
	c.SkipVerify = skipVerify

	// Note: this cannot be set via env variable. Enhance?
	c.UseTokenCache = true

	return *c, nil
}

func getS3ConfigFromSdkConfig(sConf *CecConfig) cells_sdk.S3Config {
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

// func getS3ConfigFromEnv() (cells_sdk.S3Config, error) {

// 	var c cells_sdk.S3Config

// 	// check presence of Env variable
// 	endpoint := os.Getenv(KeyS3Endpoint)
// 	region := os.Getenv(KeyS3Region)
// 	bucket := os.Getenv(KeyS3Bucket)
// 	apiKey := os.Getenv(KeyS3ApiKey)
// 	apiSecret := os.Getenv(KeyS3ApiSecret)
// 	usePSHStr := os.Getenv(KeyS3UsePydioSpecificHeader)
// 	if usePSHStr == "" {
// 		usePSHStr = "false"
// 	}
// 	usePSH, err := strconv.ParseBool(usePSHStr)
// 	if err != nil {
// 		return c, err
// 	}

// 	isDebugStr := os.Getenv(KeyS3IsDebug)
// 	if isDebugStr == "" {
// 		isDebugStr = "false"
// 	}
// 	isDebug, err := strconv.ParseBool(isDebugStr)
// 	if err != nil {
// 		return c, err
// 	}

// 	if !(len(endpoint) > 0 && len(region) > 0 && len(bucket) > 0 && len(apiKey) > 0 && len(apiSecret) > 0) {
// 		return c, nil
// 	}

// 	c.Endpoint = endpoint
// 	c.Region = region
// 	c.Bucket = bucket
// 	c.ApiKey = apiKey
// 	c.ApiSecret = apiSecret
// 	c.UsePydioSpecificHeader = usePSH
// 	c.IsDebug = isDebug

// 	return c, nil
// }
