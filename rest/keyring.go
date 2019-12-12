package rest

import (
	"strings"

	"github.com/zalando/go-keyring"

	cells_sdk "github.com/pydio/cells-sdk-go"
)

const (
	keyringService              = "com.pydio.cells-client"
	keyringIdTokenKey           = "IdToken"
	keyringClientCredentialsKey = "ClientCredentials"
)

// ConfigToKeyring tries to store tokens in local keychain and remove them from the conf
func ConfigToKeyring(conf *cells_sdk.SdkConfig) error {

	// We use OAuth2 grant flow
	if conf.IdToken != "" && conf.RefreshToken != "" {
		key := conf.Url + "::" + keyringIdTokenKey
		value := conf.IdToken + "__//__" + conf.RefreshToken
		if e := keyring.Set(keyringService, key, value); e != nil {
			return e
		}
		conf.IdToken = ""
		conf.RefreshToken = ""
	}

	// We use client credentials
	if conf.ClientSecret != "" && conf.Password != "" {
		key := conf.Url + "::" + keyringClientCredentialsKey
		value := conf.ClientSecret + "__//__" + conf.Password
		if e := keyring.Set(keyringService, key, value); e != nil {
			return e
		}
		conf.ClientSecret = ""
		conf.Password = ""
	}

	return nil
}

// ConfigFromKeyring tries to find sensitive info inside local keychain and feed the conf.
func ConfigFromKeyring(conf *cells_sdk.SdkConfig) error {

	// If only client key and user name, consider Client Secret and password are in the keyring
	if conf.ClientKey != "" && conf.ClientSecret == "" && conf.User != "" && conf.Password == "" {
		if value, e := keyring.Get(keyringService, conf.Url+"::"+keyringClientCredentialsKey); e == nil {
			parts := strings.Split(value, "__//__")
			conf.ClientSecret = parts[0]
			conf.Password = parts[1]
		} else {
			return e
		}
	}

	// If no token, no user and no client key, consider tokens are stored in keyring
	if conf.IdToken == "" && conf.RefreshToken == "" && conf.User == "" && conf.Password == "" {
		if value, e := keyring.Get(keyringService, conf.Url+"::"+keyringIdTokenKey); e == nil {
			parts := strings.Split(value, "__//__")
			conf.IdToken = parts[0]
			conf.RefreshToken = parts[1]
		} else {
			return e
		}
	}
	return nil
}

// ClearKeyring removes sensitive info from local keychain, if they are present.
func ClearKeyring(c *cells_sdk.SdkConfig) error {
	// Best effort to remove known keys from keyring
	// TODO maybe check if at least one of the two has been found and deleted and otherwise print at least a warning
	if err := keyring.Delete(keyringService, c.Url+"::"+keyringClientCredentialsKey); err != nil {
		if err.Error() != "secret not found in keyring" {
			return err
		}
	}
	if err := keyring.Delete(keyringService, c.Url+"::"+keyringIdTokenKey); err != nil {
		if err.Error() != "secret not found in keyring" {
			return err
		}
	}
	return nil
}
