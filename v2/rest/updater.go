package rest

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
	update2 "github.com/inconshreveable/go-update"
	"github.com/kardianos/osext"
	"github.com/pydio/cells/common/utils/net"

	"github.com/pydio/cells-client/v2/common"
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
	// BinarySize int64 `json:"BinarySize,omitempty"`
	BinarySize string `json:"BinarySize,omitempty"`
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

func LoadUpdates(ctx context.Context, channel string) ([]*UpdatePackage, error) {

	urlConf := common.UpdateServerURL
	parsed, e := url.Parse(urlConf)
	if e != nil {
		return nil, e
	}
	if strings.Trim(parsed.Path, "/") == "" {
		parsed.Path = "/a/update-server"
	}

	jsonReq, _ := json.Marshal(&UpdateRequest{
		Channel:        channel,
		PackageName:    common.PackageType,
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
	if err != nil {
		return nil, err
	}
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

func ApplyUpdate(ctx context.Context, p *UpdatePackage, dryRun bool, pgChan chan float64, doneChan chan bool, errorChan chan error) {

	defer func() {
		close(doneChan)
	}()

	if resp, err := http.Get(p.BinaryURL); err != nil {
		errorChan <- err
		return
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			plain, _ := ioutil.ReadAll(resp.Body)
			errorChan <- fmt.Errorf("binary.download.error %s", string(plain))
			return
		}

		targetPath := ""
		if dryRun {
			targetPath = filepath.Join(os.TempDir(), "pydio-update")
		}
		if p.BinaryChecksum == "" || p.BinarySignature == "" {
			errorChan <- fmt.Errorf("missing checksum and signature infos")
			return
		}
		checksum, e := base64.StdEncoding.DecodeString(p.BinaryChecksum)
		if e != nil {
			errorChan <- e
			return
		}
		signature, e := base64.StdEncoding.DecodeString(p.BinarySignature)
		if e != nil {
			errorChan <- e
			return
		}

		pKey := common.UpdatePublicKey
		block, _ := pem.Decode([]byte(pKey))
		if block == nil || block.Type != "PUBLIC KEY" {
			log.Fatalf("failed to decode pubKey")
		}
		var pubKey rsa.PublicKey
		if _, err := asn1.Unmarshal(block.Bytes, &pubKey); err != nil {
			errorChan <- err
			return
		}

		// Write previous version inside the same folder
		if targetPath == "" {
			exe, er := osext.Executable()
			if er != nil {
				errorChan <- err
				return
			}
			targetPath = exe
		}
		// backupFile := targetPath + "-" + common.Version + "-rev-" + common.BuildStamp

		defaultConfPath := DefaultConfigFilePath()
		backupFile := filepath.Join(filepath.Dir(defaultConfPath), "cec-"+common.Version+"-"+common.BuildStamp)

		reader := net.BodyWithProgressMonitor(resp, pgChan, nil)

		er := update2.Apply(reader, update2.Options{
			Checksum:    checksum,
			Signature:   signature,
			TargetPath:  targetPath,
			OldSavePath: backupFile,
			Hash:        crypto.SHA256,
			PublicKey:   &pubKey,
			Verifier:    update2.NewRSAVerifier(),
		})
		if er != nil {
			errorChan <- er
		}

		// Now try to move previous version to the services folder. Do not break on error, just Warn in the logs.
		// dataDir, _ := config.ServiceDataDir(common.SERVICE_GRPC_NAMESPACE_ + common.SERVICE_UPDATE)

		// backupPath := filepath.Join("dataDir", filepath.Base(backupFile))
		// if err := filesystem.SafeRenameFile(backupFile, backupPath); err != nil {
		// 	// log.Logger(ctx).Warn("Update successfully applied but previous binary could not be moved to backup folder", zap.Error(err))
		// 	log.Println("Update successfully applied but previous binary could not be moved to backup folder")
		// }

		return
	}

}
