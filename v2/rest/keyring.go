package rest

import (
	"strings"

	"github.com/zalando/go-keyring"

	cells_sdk "github.com/pydio/cells-sdk-go"

	"github.com/pydio/cells-client/v2/common"
)

const (
	keyringService              = "com.pydio.cells-client"
	keyringIdTokenKey           = "IdToken"
	keyringClientCredentialsKey = "ClientCredentials"
	keyringPersonalToken        = "PersonalToken"
)

// ConfigToKeyring tries to store tokens in local keychain and remove them from the conf
func ConfigToKeyring(conf *CecConfig) error {
	switch conf.AuthType {
	case common.OAuthType:
		key := conf.Url + "::" + keyringIdTokenKey
		value := conf.IdToken + "__//__" + conf.RefreshToken
		if e := keyring.Set(keyringService, key, value); e != nil {
			return e
		}
		conf.IdToken = ""
		conf.RefreshToken = ""
	case common.ClientAuthType:
		key := conf.Url + "::" + keyringClientCredentialsKey
		value := conf.ClientSecret + "__//__" + conf.Password
		if e := keyring.Set(keyringService, key, value); e != nil {
			return e
		}
		conf.ClientSecret = ""
		conf.Password = ""
	case common.PersonalTokenType:
		key := conf.Url + "::" + keyringPersonalToken
		value := conf.IdToken
		if e := keyring.Set(keyringService, key, value); e != nil {
			return e
		}
		conf.IdToken = ""
	}
	return nil
}

// ConfigFromKeyring tries to find sensitive info inside local keychain and feed the conf.
func ConfigFromKeyring(conf *CecConfig) error {
	switch conf.AuthType {
	case common.OAuthType:
		if value, e := keyring.Get(keyringService, conf.Url+"::"+keyringIdTokenKey); e == nil {
			parts := strings.Split(value, "__//__")
			conf.IdToken = parts[0]
			conf.RefreshToken = parts[1]
		} else {
			return e
		}
	case common.ClientAuthType:
		if value, e := keyring.Get(keyringService, conf.Url+"::"+keyringClientCredentialsKey); e == nil {
			parts := strings.Split(value, "__//__")
			conf.ClientSecret = parts[0]
			conf.Password = parts[1]
		} else {
			return e
		}
	case common.PersonalTokenType:
		if value, e := keyring.Get(keyringService, conf.Url+"::"+keyringIdTokenKey); e == nil {
			conf.IdToken = value
		} else {
			return e
		}
	default:
		// default case is intended for backwards compatibility
		if conf.ClientKey != "" && conf.ClientSecret == "" && conf.User != "" && conf.Password == "" { // client auth
			if value, e := keyring.Get(keyringService, conf.Url+"::"+keyringClientCredentialsKey); e == nil {
				parts := strings.Split(value, "__//__")
				conf.ClientSecret = parts[0]
				conf.Password = parts[1]
			} else {
				return e
			}
		} else if conf.IdToken == "" && conf.RefreshToken == "" && conf.User == "" && conf.Password == "" { // oauth
			if value, e := keyring.Get(keyringService, conf.Url+"::"+keyringIdTokenKey); e == nil {
				parts := strings.Split(value, "__//__")
				conf.IdToken = parts[0]
				conf.RefreshToken = parts[1]
			} else {
				return e
			}
		}
	}
	return nil
}

// TODO create methods to properly concatenate or split the values inside the keyring

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
