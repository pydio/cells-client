package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/manifoldco/promptui"

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

	var configList ConfigList
	err = json.Unmarshal(data, &configList)
	if err != nil {
		return nil, fmt.Errorf("unknown config format: %s", err)
	}

	// Double-check to detect and migrate legacy configs
	if configList.Configs == nil || len(configList.Configs) == 0 {
		var oldConf *CecConfig
		if err = json.Unmarshal(data, &oldConf); err != nil {
			return nil, fmt.Errorf("unknown config format: %s", err)
		}

		id := createID(oldConf)
		oldConf.Label = createLabel(oldConf)
		oldConf.CreatedAtVersion = common.Version
		configs := make(map[string]*CecConfig)
		configs[id] = oldConf

		configList = ConfigList{
			Configs:        configs,
			ActiveConfigID: id,
		}
		err = configList.SaveConfigFile()
		if err != nil {
			return nil, fmt.Errorf("could not save after config migration: %s", err.Error())
		}
	}

	return &configList, nil
}

func UpdateConfig(newConf *CecConfig) error {

	var err error
	oldConfig := DefaultConfig
	defer func() {
		if err != nil {
			DefaultConfig = oldConfig
		}
	}()

	uname, e := RetrieveSessionLogin(newConf)
	if e != nil {
		return fmt.Errorf("could not connect to distant server with provided parameters. Discarding change")
	}
	newConf.SdkConfig.User = uname
	id := createID(newConf)
	newConf.Label = createLabel(newConf)
	newConf.CreatedAtVersion = common.Version
	DefaultConfig = newConf

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

// Remove removes a config from the list of available configurations by its ID.
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

func (list *ConfigList) GetActiveConfig() (*CecConfig, error) {
	c := list.Configs[list.ActiveConfigID]
	if c == nil {
		return nil, fmt.Errorf("active config not found")
	}
	if !c.SkipKeyring {
		if err := ConfigFromKeyring(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func createID(c *CecConfig) string {
	var port string
	u, _ := url.Parse(c.SdkConfig.Url)
	port = u.Port()
	if port == "" {
		switch u.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}

	return fmt.Sprintf("%s@%s:%s", c.User, u.Hostname(), port)
}

func createLabel(c *CecConfig) string {
	u, _ := url.Parse(c.SdkConfig.Url)
	return fmt.Sprintf("%s@%s", c.SdkConfig.User, u.Hostname())
}

// SaveConfigFile saves inside the config file.
func (list *ConfigList) SaveConfigFile() error {
	confData, _ := json.MarshalIndent(&list, "", "\t")
	if err := os.WriteFile(GetConfigFilePath(), confData, 0666); err != nil {
		return fmt.Errorf("could not save the config file, cause: %s", err)
	}
	return nil
}
