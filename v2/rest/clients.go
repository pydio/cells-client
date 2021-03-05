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
	"sync"

	"github.com/go-openapi/strfmt"
	"github.com/shibukawa/configdir"

	cells_sdk "github.com/pydio/cells-sdk-go"
	"github.com/pydio/cells-sdk-go/client"
	"github.com/pydio/cells-sdk-go/transport"
	sdk_rest "github.com/pydio/cells-sdk-go/transport/rest"

	"github.com/pydio/cells-client/v2/common"
)

var (
	// DefaultConfig  stores the current active config.
	DefaultConfig  *CecConfig
	configFilePath string
)

// CecConfig extends the default SdkConfig with custom parameters.
type CecConfig struct {
	cells_sdk.SdkConfig
	SkipKeyring      bool   `json:"skipKeyring"`
	AuthType         string `json:"authType"`
	CreatedAtVersion string `json:"createdAtVersion"`
}

// GetApiClient connects to the Pydio Cells server defined by this config, by sending an authentication
// request to the OIDC service to get a valid JWT (or taking the JWT from cache).
// It also returns a context to be used in subsequent requests.
func GetApiClient(anonymous ...bool) (context.Context, *client.PydioCellsRest, error) {

	anon := false
	if len(anonymous) > 0 && anonymous[0] {
		anon = true
	}
	DefaultConfig.CustomHeaders = map[string]string{"User-Agent": common.AppName + "/" + common.Version}
	c, t, e := sdk_rest.GetClientTransport(&DefaultConfig.SdkConfig, anon)
	if e != nil {
		return nil, nil, e
	}
	cl := client.New(t, strfmt.Default)
	return c, cl, nil

}

// AuthenticatedGet performs an authenticated GET request for the passed URI (that must start with a '/').
func AuthenticatedGet(uri string) (*http.Response, error) {

	currURL := DefaultConfig.Url + uri
	req, err := http.NewRequest("GET", currURL, nil)
	if err != nil {
		return nil, err
	}
	return AuthenticatedRequest(req, &DefaultConfig.SdkConfig)
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

	return httpClient.Do(req)
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
	return filepath.Join(f, "config.json")
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
		if !c.SkipKeyring {
			if err := ConfigToKeyring(&storeConfig); err != nil {
				return false
			}
		}
		// Save config to renew TokenExpireAt
		confData, _ := json.MarshalIndent(&storeConfig, "", "\t")
		ioutil.WriteFile(GetConfigFilePath(), confData, 0600)
	}

	return refreshed
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
