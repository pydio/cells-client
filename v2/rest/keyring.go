package rest

import (
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"

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
		if e := keyring.Set(keyringService, currKey, conf.Password); e != nil {
			return e
		}
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
		conf.Password = value
	case common.PatType:
		conf.IdToken = value
	default:
		// default case is intended for backwards compatibility
		// TODO manage this cleanly
		if conf.User != "" && conf.Password == "" { // client auth
			if value, e := keyring.Get(keyringService, key(conf.Url, "ClientCredentials")); e == nil {
				parts := splitValue(value)
				//conf.ClientSecret = parts[0]
				conf.Password = parts[1]
			} else {
				return e
			}
		} else if conf.IdToken == "" && conf.RefreshToken == "" && conf.Password == "" { // oauth
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

// CheckKeyring simply tries a write followed by a read in the local keyring and
// returns nothing if it works or an error otherwise.
func CheckKeyring() error {

	testKey := key("https://test.example.com", "john.doe")
	testValue := "A very complicated value !!#%<{}//\\q"

	if e := keyring.Set(keyringService, testKey, testValue); e != nil {
		return e
	}

	defer func() {
		// Best effort to remove the test key from the keyring => ignore error
		_ = keyring.Delete(keyringService, testKey)
	}()

	value, err := keyring.Get(keyringService, testKey)
	if err != nil {
		return err
	}

	if value != testValue {
		return fmt.Errorf("Keyring seems to be broken in this machine, retrieved value (%s) differs from the one we stored (%s)", value, testValue)
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

// ClearKeyring removes sensitive info from the local keychain, if they are present.
func ClearKeyring(c *CecConfig) error {
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
