package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/manifoldco/promptui"

	cellsSdk "github.com/pydio/cells-sdk-go/v5"
	sdkRest "github.com/pydio/cells-sdk-go/v5/transport/rest"

	"github.com/pydio/cells-client/v4/common"
)

type ConfigList struct {
	ActiveConfigID string
	Configs        map[string]*CecConfig
}

// GetConfigList retrieves configuration stored in the config.json file.
func GetConfigList() (*ConfigList, error) {

	data, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &ConfigList{Configs: make(map[string]*CecConfig)}, nil
		} else {
			return nil, err
		}
	}

	var tmp ConfigList
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal conf from %s, cause: %s", GetConfigFilePath(), err)
	}
	configList := &tmp

	// Double-check to detect and migrate legacy configs
	if configList.Configs == nil || len(configList.Configs) == 0 {

		configList, err = tryToGetLegacyConfig(data)
		if err != nil {
			return nil, err
		}

		err = configList.SaveConfigFile()
		if err != nil {
			return nil, fmt.Errorf("could not save after config migration: %s", err.Error())
		}
	} else {
		hasChanged, err := migrateAuthTypes(configList)
		if err != nil {
			return nil, err
		}
		if hasChanged {
			err := configList.SaveConfigFile()
			if err != nil {
				return nil, fmt.Errorf("could not save after config migration: %s", err.Error())
			}
		}
	}

	return configList, nil
}

// tryToGetLegacyConfig is best-effort to retrieve and migrate cec v2 configuration to the latest format at first use.
func tryToGetLegacyConfig(data []byte) (*ConfigList, error) {

	var oldConf *CecConfig
	if err := json.Unmarshal(data, &oldConf); err != nil {
		return nil, fmt.Errorf("unknown config format: %s", err)
	}
	id := createID(oldConf)
	oldConf.Label = createLabel(oldConf)
	oldConf.CreatedAtVersion = common.Version
	configs := make(map[string]*CecConfig)
	configs[id] = oldConf

	configList := &ConfigList{
		Configs:        configs,
		ActiveConfigID: id,
	}
	_, err := migrateAuthTypes(configList)
	if err != nil {
		return nil, err
	}
	err = configList.SaveConfigFile()
	if err != nil {
		return nil, fmt.Errorf("could not save after config migration: %s", err.Error())
	}
	return configList, nil
}

// migrateAuthTypes simply replaces AuthType values in the given structure to use SDK v5 standard values.
// The resulting config is **not** saved to disk / keyring
func migrateAuthTypes(configList *ConfigList) (bool, error) {

	hasChanged := false
	for _, v := range configList.Configs {
		switch v.AuthType {
		case common.LegacyCecConfigAuthTypeBasic:
			v.AuthType = cellsSdk.AuthTypeClientAuth
			v.CreatedAtVersion = common.Version
			hasChanged = true
		case common.LegacyCecConfigAuthTypePat:
			v.AuthType = cellsSdk.AuthTypePat
			v.CreatedAtVersion = common.Version
			hasChanged = true
		case common.LegacyCecConfigAuthTypeOAuth:
			v.AuthType = cellsSdk.AuthTypeOAuth
			v.CreatedAtVersion = common.Version
			hasChanged = true
		}
	}
	return hasChanged, nil
}

// Remove unregisters a config from the list of available configurations by its ID.
func (list *ConfigList) Remove(id string) error {
	if _, ok := list.Configs[id]; !ok {
		return fmt.Errorf("config not found, ID is not valid [%s]", id)
	}
	if list.ActiveConfigID == id {
		list.ActiveConfigID = ""
	}
	delete(list.Configs, id)
	return nil
}

func (list *ConfigList) SetActiveConfig(id string) error {
	if _, ok := list.Configs[id]; !ok {
		return fmt.Errorf("this ID does not exist %s", id)
	}
	list.ActiveConfigID = id
	return nil
}

func (list *ConfigList) GetActiveConfig(ctx context.Context) (*CecConfig, error) {
	activeConfig := list.Configs[list.ActiveConfigID]
	if activeConfig == nil {
		return nil, fmt.Errorf("active config not found")
	}
	if !activeConfig.SkipKeyring {
		if err := ConfigFromKeyring(ctx, activeConfig); err != nil {
			return nil, err
		}
	}
	return activeConfig, nil
}

func (list *ConfigList) GetStoredConfig(ctx context.Context, id string) (*CecConfig, error) {
	c := list.Configs[id]
	if c == nil {
		return nil, fmt.Errorf("no config found for %s", id)
	}
	if !c.SkipKeyring {
		if err := ConfigFromKeyring(ctx, c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// SaveConfigFile saves inside the config file.
func (list *ConfigList) SaveConfigFile() error {
	confData, _ := json.MarshalIndent(&list, "", "\t")
	if err := os.WriteFile(GetConfigFilePath(), confData, 0666); err != nil {
		return fmt.Errorf("could not save the config file, cause: %s", err)
	}
	return nil
}

// CellsConfigStore implements a Cells Client specific ConfigRefresher, that also securely stores credentials:
// It wraps a keyring if such a tool is correctly configured and can be reached by the client.
type CellsConfigStore struct {
	refreshLock sync.Mutex
}

func (store *CellsConfigStore) RefreshIfRequired(ctx context.Context, sdkConfig *cellsSdk.SdkConfig) (bool, error) {

	// No token to refresh
	if sdkConfig.IdToken == "" || sdkConfig.RefreshToken == "" || sdkConfig.TokenExpiresAt == 0 {
		return false, nil
	}

	configId := id(sdkConfig)

	// We can only launch *one* refresh token procedure at a time (and consume the refresh only once)
	store.refreshLock.Lock()
	defer store.refreshLock.Unlock()

	list, err := GetConfigList()
	if err != nil {
		return false, fmt.Errorf("could not refresh retrieve stored config list to update, cause: %s", err.Error())
	}
	storedConf, err := list.GetStoredConfig(ctx, configId)
	if err != nil {
		return false, err
	}
	updated, err := sdkRest.RefreshJwtToken(common.AppName, storedConf.SdkConfig)
	if err != nil {
		return false, fmt.Errorf("could not refresh JWT token for %s, cause: %s", configId, err.Error())
	}
	// Update values in the given struct (we have a pointer)
	sdkConfig.IdToken = storedConf.IdToken
	sdkConfig.User = storedConf.User
	sdkConfig.TokenExpiresAt = storedConf.TokenExpiresAt
	if !updated { // we yet have reloaded the token from the central store, in case it has been changed in another thread in the meantime.
		return false, nil
	}

	// Store the updated config

	//  Finally, if username has changed. Not sure if it is really relevant here.
	newId := id(sdkConfig)
	if newId != configId {
		// // Set new active config
		// list.SetActiveConfig(newId)
		// Delete old config (ignoring any error while deleting)
		_ = list.Remove(configId)
	}

	err = UpdateConfig(storedConf)
	if err != nil {
		return true, fmt.Errorf("could not store updated conf for %s, cause: %s", newId, err.Error())
	}
	return true, nil
}

func UpdateConfig(newConf *CecConfig) error {

	var err error

	//// Failsafe if an error is thrown at save time
	//if DefaultConfig != nil && DefaultConfig.SdkConfig != nil {
	//	oldConfig := CloneConfig(DefaultConfig)
	//	defer func() {
	//		if err != nil {
	//			DefaultConfig = CloneConfig(oldConfig)
	//		}
	//	}()
	//}

	uname, e := RetrieveSessionLogin(newConf)
	if e != nil {
		return fmt.Errorf("could not connect to distant server with provided parameters. Discarding change")
	}
	newConf.SdkConfig.User = uname
	id := createID(newConf)
	newConf.Label = createLabel(newConf)
	newConf.CreatedAtVersion = common.Version
	// DefaultConfig = newConf

	// We create a clone that will be persisted without sensitive info
	persistedConf := CloneConfig(newConf)
	if err = ConfigToKeyring(persistedConf); err != nil {
		// Could not save credentials in the keyring: sensitive information are still in clear text.
		// We warn the user but do not abort the process.
		fmt.Println(promptui.IconWarn + " " + NoKeyringMsg)
		// We also force the "Skip Keyring" flag in the config file to be explicit
		persistedConf.SkipKeyring = true
	}

	cl, err := GetConfigList()
	if err != nil {
		return err
	}

	cl.Configs[id] = persistedConf
	cl.ActiveConfigID = id
	return cl.SaveConfigFile()
}

// Helpers

func createID(c *CecConfig) string {
	return id(c.SdkConfig)
}

func id(conf *cellsSdk.SdkConfig) string {
	var port string
	u, _ := url.Parse(conf.Url)
	port = u.Port()
	if port == "" {
		switch u.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}
	return fmt.Sprintf("%s@%s:%s", conf.User, u.Hostname(), port)
}

func createLabel(c *CecConfig) string {
	u, _ := url.Parse(c.SdkConfig.Url)
	return fmt.Sprintf("%s@%s", c.SdkConfig.User, u.Hostname())
}
