// +build ignore

package rest

import (
	"strings"

	"github.com/99designs/keyring"
	"github.com/pydio/cells-sdk-go"
)

func ConfigToKeyring(conf *cells_sdk.SdkConfig) error {
	if conf.IdToken != "" && conf.RefreshToken != "" {
		ring, err := keyring.Open(keyring.Config{
			ServiceName:              "com.pydio.cells-client",
			KeychainTrustApplication: true,
		})
		if err != nil {
			return err
		}
		key := conf.Url + "::IdToken"
		value := conf.IdToken + "__//__" + conf.RefreshToken
		if e := ring.Set(keyring.Item{
			Key:         key,
			Data:        []byte(value),
			Label:       "Connection Token",
			Description: "Identity Token used to access the server",
		}); e != nil {
			return e
		}
		conf.IdToken = ""
		conf.RefreshToken = ""
	}
	return nil
}

func ConfigFromKeyring(conf *cells_sdk.SdkConfig) error {
	// If nothing is provided, consider it is stored in keyring
	if conf.IdToken == "" && conf.RefreshToken == "" && conf.User == "" && conf.Password == "" {
		ring, err := keyring.Open(keyring.Config{
			ServiceName: "com.pydio.cells-client",
		})
		if err != nil {
			return err
		}
		if id, e := ring.Get(conf.Url + "::IdToken"); e == nil {
			value := string(id.Data)
			parts := strings.Split(value, "__//__")
			conf.IdToken = parts[0]
			conf.RefreshToken = parts[1]
		} else {
			return e
		}
	}
	return nil
}

func ClearKeyring(c *cells_sdk.SdkConfig) error {
	// Try to delete creds from keyring
	if ring, er := keyring.Open(keyring.Config{
		ServiceName: "com.pydio.cells-client",
	}); er == nil {
		return ring.Remove(c.Url + "::IdToken")
	}
	return nil
}
