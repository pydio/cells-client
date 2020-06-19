package rest

import (
	"strings"

	"github.com/pydio/cells-sdk-go"
	"github.com/zalando/go-keyring"
)

var (
	keyringService = "com.pydio.cells-client"
)

// ConfigToKeyring tries to store tokens in local keychain and remove them from the conf
func ConfigToKeyring(conf *cells_sdk.SdkConfig) error {
	if conf.IdToken != "" && conf.RefreshToken != "" {
		key := conf.Url + "::IdToken"
		value := conf.IdToken + "__//__" + conf.RefreshToken
		if e := keyring.Set(keyringService, key, value); e != nil {
			return e
		}
		conf.IdToken = ""
		conf.RefreshToken = ""
	}
	return nil
}

// ConfigFromKeyring tries to find tokens inside local keychain and feed the conf with them
func ConfigFromKeyring(conf *cells_sdk.SdkConfig) error {
	// If nothing is provided, consider it is stored in keyring
	if conf.IdToken == "" && conf.RefreshToken == "" && conf.User == "" && conf.Password == "" {
		if value, e := keyring.Get(keyringService, conf.Url+"::IdToken"); e == nil {
			parts := strings.Split(value, "__//__")
			conf.IdToken = parts[0]
			conf.RefreshToken = parts[1]
		} else {
			return e
		}
	}
	return nil
}

// ClearKeyring removes tokens from local keychain, if they are present
func ClearKeyring(c *cells_sdk.SdkConfig) error {
	// Try to delete creds from keyring
	return keyring.Delete(keyringService, c.Url+"::IdToken")
}
