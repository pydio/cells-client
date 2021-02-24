package rest

import (
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"

	cells_sdk "github.com/pydio/cells-sdk-go"

	"github.com/pydio/cells-client/v2/common"
)

const keyringService = "com.pydio.cells-client"

// ConfigToKeyring stores sensitive information in local keyring if any and removes it from current SDK config.
func ConfigToKeyring(conf *CecConfig) error {

	currKey := key(conf.Url, conf.User)

	switch conf.AuthType {
	case common.PatType:
		if e := keyring.Set(keyringService, currKey, conf.IdToken); e != nil {
			return e
		}
		conf.IdToken = ""
	case common.OAuthType:
		value := value(conf.IdToken, conf.RefreshToken)
		if e := keyring.Set(keyringService, currKey, value); e != nil {
			return e
		}
		conf.IdToken = ""
		conf.RefreshToken = ""
	case common.ClientAuthType:
		value := value(conf.ClientSecret, conf.Password)
		if e := keyring.Set(keyringService, currKey, value); e != nil {
			return e
		}
		conf.ClientSecret = ""
		conf.Password = ""
	}
	return nil
}

// ConfigFromKeyring tries to find sensitive info inside local keychain and feed the conf.
func ConfigFromKeyring(conf *CecConfig) error {
	value, err := keyring.Get(keyringService, key(conf.Url, conf.User))
	if err != nil {
		return err
	}

	switch conf.AuthType {
	case common.OAuthType:
		parts := splitValue(value)
		conf.IdToken = parts[0]
		conf.RefreshToken = parts[1]
	case common.ClientAuthType:
		parts := splitValue(value)
		conf.ClientSecret = parts[0]
		conf.Password = parts[1]
	case common.PatType:
		conf.IdToken = value
	default:
		// default case is intended for backwards compatibility
		// TODO manage this cleanly
		if conf.ClientKey != "" && conf.ClientSecret == "" && conf.User != "" && conf.Password == "" { // client auth
			if value, e := keyring.Get(keyringService, key(conf.Url, "ClientCredentials")); e == nil {
				parts := splitValue(value)
				conf.ClientSecret = parts[0]
				conf.Password = parts[1]
			} else {
				return e
			}
		} else if conf.IdToken == "" && conf.RefreshToken == "" && conf.User == "" && conf.Password == "" { // oauth
			if value, e := keyring.Get(keyringService, key(conf.Url, "IdToken")); e == nil {
				parts := splitValue(value)
				conf.IdToken = parts[0]
				conf.RefreshToken = parts[1]
			} else {
				return e
			}
		}
	}
	return nil
}

const (
	keySep   = "::"
	valueSep = "__//__"
)

func key(prefix, suffix string) string {
	return fmt.Sprintf("%s%s%s", prefix, keySep, suffix)
}

func value(prefix, suffix string) string {
	return fmt.Sprintf("%s%s%s", prefix, valueSep, suffix)
}

func splitValue(value string) []string {
	return strings.Split(value, valueSep)
}

// ClearKeyring removes sensitive info from local keychain, if they are present.
func ClearKeyring(c *cells_sdk.SdkConfig) error {
	// Best effort to remove known keys from keyring
	if err := keyring.Delete(keyringService, key(c.Url, c.User)); err != nil {
		if err.Error() != "secret not found in keyring" {
			return err
		}
	}

	// Legacy keys
	// TODO maybe check if at least one of the two has been found and deleted and otherwise print at least a warning
	if err := keyring.Delete(keyringService, key(c.Url, "ClientCredentials")); err != nil {
		if err.Error() != "secret not found in keyring" {
			return err
		}
	}
	if err := keyring.Delete(keyringService, key(c.Url, "IdToken")); err != nil {
		if err.Error() != "secret not found in keyring" {
			return err
		}
	}
	return nil
}
