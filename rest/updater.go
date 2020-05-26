package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/pydio/cells-client/common"
)

type UpdateRequest struct {
	// Channel name
	Channel string `json:"Channel,omitempty"`
	// Name of the currently running application
	PackageName string `json:"PackageName,omitempty"`
	// Current version of the application
	CurrentVersion string `json:"CurrentVersion,omitempty"`
	// Current GOOS
	GOOS string `json:"GOOS,omitempty"`
	// Current GOARCH
	GOARCH string `json:"GOARCH,omitempty"`
	// Not Used : specific service to get updates for
	ServiceName string `json:"ServiceName,omitempty"`
	// For enterprise version, info about the current license
	LicenseInfo map[string]string `json:"LicenseInfo,omitempty"`
}

type UpdatePackage struct {
	// Name of the application
	PackageName string `json:"PackageName,omitempty"`
	// Version of this new binary
	Version string `json:"Version,omitempty"`
	// Release date of the binary
	ReleaseDate int32 `json:"ReleaseDate,omitempty"`
	// Short human-readable description
	Label string `json:"Label,omitempty"`
	// Long human-readable description (markdown)
	Description string `json:"Description,omitempty"`
	// List or public URL of change logs
	ChangeLog string `json:"ChangeLog,omitempty"`
	// License of this package
	License string `json:"License,omitempty"`
	// Https URL where to download the binary
	BinaryURL string `json:"BinaryURL,omitempty"`
	// Checksum of the binary to verify its integrity
	BinaryChecksum string `json:"BinaryChecksum,omitempty"`
	// Signature of the binary
	BinarySignature string `json:"BinarySignature,omitempty"`
	// Hash type used for the signature
	BinaryHashType string `json:"BinaryHashType,omitempty"`
	// Size of the binary to download
	BinarySize int64 `json:"BinarySize,omitempty"`
	// GOOS value used at build time
	BinaryOS string `json:"BinaryOS,omitempty"`
	// GOARCH value used at build time
	BinaryArch string `json:"BinaryArch,omitempty"`
	// Not used : if binary is a patch
	IsPatch bool `json:"IsPatch,omitempty"`
	// Not used : if a patch, how to patch (bsdiff support)
	PatchAlgorithm string `json:"PatchAlgorithm,omitempty"`

	// ServiceName string                `json:"ServiceName,omitempty"`
	// Status      string `json:"Status,omitempty"`

}

type UpdateResponse struct {
	Channel string `json:"Channel,omitempty"`
	// List of available binaries
	AvailableBinaries []*UpdatePackage `json:"AvailableBinaries,omitempty"`
}

func LoadUpdates(ctx context.Context) ([]*UpdatePackage, error) {

	urlConf := common.UpdateServerUrl
	parsed, e := url.Parse(urlConf)
	if e != nil {
		return nil, e
	}
	if strings.Trim(parsed.Path, "/") == "" {
		parsed.Path = "/a/update-server"
	}

	jsonReq, _ := json.Marshal(&UpdateRequest{
		Channel:        common.UpdateChannel,
		PackageName:    common.UpdatePackageType,
		CurrentVersion: common.Version,
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
	})
	reader := bytes.NewReader(jsonReq)

	var response *http.Response
	var err error

	postRequest, err := http.NewRequest("POST", strings.TrimRight(parsed.String(), "/")+"/", reader)
	if err != nil {
		return nil, err
	}
	postRequest.Header.Add("Content-type", "application/json")
	myClient := http.DefaultClient

	if proxy := os.Getenv("CELLS_UPDATE_HTTP_PROXY"); proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return nil, err
		}
		myClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	}

	response, err = myClient.Do(postRequest)

	// if proxy := os.Getenv("CELLS_UPDATE_HTTP_PROXY"); proxy == "" {
	// 	response, err = http.Post(strings.TrimRight(parsed.String(), "/")+"/", "application/json", reader)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// } else {
	// 	postRequest, err := http.NewRequest("POST", strings.TrimRight(parsed.String(), "/")+"/", reader)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	postRequest.Header.Add("Content-type", "application/json")
	//
	// 	proxyUrl, err := url.Parse(proxy)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	myClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
	// 	response, err = myClient.Do(postRequest)
	// }

	if response.StatusCode != 200 {
		rErr := fmt.Errorf("could not connect to the update server, error code was %d", response.StatusCode)
		if response.StatusCode == 500 {
			var jsonErr struct {
				Title  string
				Detail string
			}
			data, _ := ioutil.ReadAll(response.Body)
			if e := json.Unmarshal(data, &jsonErr); e == nil {
				rErr = fmt.Errorf("failed connecting to the update server (%s), error code %d", jsonErr.Title, response.StatusCode)
			}
		}
		return nil, rErr
	}
	var updateResponse UpdateResponse
	data, _ := ioutil.ReadAll(response.Body)
	if e := json.Unmarshal(data, &updateResponse); e != nil {
		return nil, e
	}

	// Sort by version using hashicorp sorting (X.X.X-rc should appear before X.X.X)
	sort.Slice(updateResponse.AvailableBinaries, func(i, j int) bool {
		va, _ := version.NewVersion(updateResponse.AvailableBinaries[i].Version)
		vb, _ := version.NewVersion(updateResponse.AvailableBinaries[j].Version)
		return va.LessThan(vb)
	})

	return updateResponse.AvailableBinaries, nil

}
