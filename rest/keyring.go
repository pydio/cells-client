package rest

import (
	"context"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"

	"github.com/pydio/cells-client/v4/common"
	cellsSdk "github.com/pydio/cells-sdk-go/v5"
)

const (
	// NoKeyringMsg warns end user when no keyring is found
	NoKeyringMsg = "Could not access local keyring: sensitive information like token or password will end up stored in clear text in the client machine."

	keySep   = "::"
	valueSep = "__//__"
)

func getKeyringServiceName() string {
	return "com.pydio." + common.AppName
}

// ConfigToKeyring stores sensitive information in local keyring if any and removes it from current SDK config.
func ConfigToKeyring(conf *CecConfig) error {

	currKey := key(conf.Url, conf.User)
	switch conf.AuthType {
	case cellsSdk.AuthTypePat:
		if e := keyring.Set(getKeyringServiceName(), currKey, conf.IdToken); e != nil {
			return e
		}
		conf.IdToken = ""
	case cellsSdk.AuthTypeOAuth:
		value := value(conf.IdToken, conf.RefreshToken)
		if e := keyring.Set(getKeyringServiceName(), currKey, value); e != nil {
			return e
		}
		conf.IdToken = ""
		conf.RefreshToken = ""
	case cellsSdk.AuthTypeClientAuth:
		if e := keyring.Set(getKeyringServiceName(), currKey, conf.Password); e != nil {
			return e
		}
		conf.Password = ""
	}
	return nil
}

// ConfigFromKeyring tries to find sensitive info inside local keychain and feed the conf.
func ConfigFromKeyring(ctx context.Context, conf *CecConfig) error {
	value, err := keyring.Get(getKeyringServiceName(), key(conf.Url, conf.User))
	if err != nil {
		// Best effort to retrieve legacy conf
		err = retrieveLegacyKey(ctx, conf)
		if err != nil {
			return err
		}
		value, err = keyring.Get(getKeyringServiceName(), key(conf.Url, conf.User))
		if err != nil {
			return err
		}
	}

	switch conf.AuthType {
	case cellsSdk.AuthTypeOAuth:
		parts := splitValue(value)
		conf.IdToken = parts[0]
		conf.RefreshToken = parts[1]
	case cellsSdk.AuthTypeClientAuth, common.LegacyCecConfigAuthTypeBasic:
		conf.Password = value
	case cellsSdk.AuthTypePat, common.LegacyCecConfigAuthTypePat:
		conf.IdToken = value
	}
	return nil
}

// CheckKeyring simply tries a write followed by a read in the local keyring and
// returns nothing if it works or an error otherwise.
func CheckKeyring() error {

	fmt.Println("Checking keyring service for", getKeyringServiceName())

	testKey := key("https://test.example.com", "john.doe")
	testValue := "A very complicated value !!#%<{}//\\q"

	if e := keyring.Set(getKeyringServiceName(), testKey, testValue); e != nil {
		return e
	}

	defer func() {
		// Best effort to remove the test key from the keyring => ignore error
		_ = keyring.Delete(getKeyringServiceName(), testKey)
	}()

	value, err := keyring.Get(getKeyringServiceName(), testKey)
	if err != nil {
		return err
	}

	if value != testValue {
		return fmt.Errorf("keyring seems to be broken in this machine, retrieved value (%s) differs from the one we stored (%s)", value, testValue)
	}

	return nil
}

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
	if err := keyring.Delete(getKeyringServiceName(), key(c.Url, c.User)); err != nil {
		if err.Error() != "secret not found in keyring" {
			return err
		}
	}
	return nil
}

func retrieveLegacyKey(_ context.Context, conf *CecConfig) error {
	if conf.User != "" && conf.Password == "" { // client auth
		if value, e := keyring.Get(getKeyringServiceName(), key(conf.Url, "ClientCredentials")); e == nil {
			parts := splitValue(value)
			//conf.ClientSecret = parts[0]
			conf.Password = parts[1]
			conf.AuthType = cellsSdk.AuthTypeClientAuth
			// Leave the keyring in a clean state
			_ = keyring.Delete(getKeyringServiceName(), key(conf.Url, "ClientCredentials"))
		} else {
			return e
		}
	} else if conf.IdToken == "" && conf.RefreshToken == "" && conf.Password == "" { // oauth
		if value, e := keyring.Get(getKeyringServiceName(), key(conf.Url, "IdToken")); e == nil {
			parts := splitValue(value)
			conf.IdToken = parts[0]
			conf.RefreshToken = parts[1]
			conf.AuthType = cellsSdk.AuthTypeOAuth
			//if _, err2 := CellsStore().RefreshIfRequired(ctx, conf.SdkConfig); err2 != nil {
			//	return err2
			//}
			_ = keyring.Delete(getKeyringServiceName(), key(conf.Url, "IdToken"))
		} else {
			return e
		}
	}

	return UpdateConfig(conf)
}
