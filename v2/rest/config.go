package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/pydio/cells-client/v2/common"
	cells_sdk "github.com/pydio/cells-sdk-go/v3"
)

type ConfigList struct {
	ActiveConfigID string
	Configs        map[string]*CecConfig
}

// GetConfigList retrieves the current configurations stored in the config.json file.
func GetConfigList() (*ConfigList, error) {

	var configList ConfigList

	// TODO this assumes config are located in the default folder
	data, err := ioutil.ReadFile(GetConfigFilePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &ConfigList{Configs: make(map[string]*CecConfig)}, nil
		} else {
			return nil, err
		}
	}

	err = json.Unmarshal(data, &configList)
	if err == nil {
		return &configList, nil
	}

	// tries to unmarshall with the old format and migrate if necessary
	var oldConf *cells_sdk.SdkConfig
	if err = json.Unmarshal(data, &oldConf); err != nil {
		return nil, fmt.Errorf("unknown config format: %s", err)
	}

	defaultID := "default"
	return &ConfigList{
		Configs: map[string]*CecConfig{defaultID: {
			SdkConfig: *oldConf,
		}},
		ActiveConfigID: defaultID,
	}, nil
}

func UpdateConfig(newConf *CecConfig) error {

	var err error
	oldConfig := DefaultConfig
	defer func() {
		if err != nil {
			DefaultConfig = oldConfig
		}
	}()

	DefaultConfig = newConf
	uname, e := RetrieveCurrentSessionLogin()
	if e != nil {
		return fmt.Errorf("could not connect to distant server with provided parameters. Discarding change")
	}
	newConf.User = uname

	if err = ConfigToKeyring(newConf); err != nil {
		// We still save info in clear text but warn the user
		fmt.Println(promptui.IconWarn + " " + NoKeyringMsg)
		// Force skip keyring flag in the config file to be explicit
		newConf.SkipKeyring = true
	}

	cl, err := GetConfigList()
	if err != nil {
		return err
	}

	id := createID(newConf)
	newConf.Label = createLabel(newConf)
	newConf.CreatedAtVersion = common.Version

	cl.Configs[id] = newConf
	cl.ActiveConfigID = id

	return cl.SaveConfigFile()
}

// Add appends the new config to the list and set it as default.
func (list *ConfigList) Add(id string, config *CecConfig) error {
	// TODO push to keyring
	//if err := ConfigToKeyring(config); err != nil {
	//	return err
	//}
	_, ok := list.Configs[id]
	if ok {
		for i := 1; i < 255; i++ {
			id = fmt.Sprintf("%d-%s", i, id)
			if _, ok := list.Configs[id]; !ok {
				break
			}
		}
	}
	list.ActiveConfigID = id
	list.Configs[id] = config
	return nil
}

// Remove removes a config from the list of available configurations by its ID.
func (list *ConfigList) Remove(id string) error {
	if _, ok := list.Configs[id]; !ok {
		return fmt.Errorf("config not found, ID is not valid [%s]", id)
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
	if err := ConfigFromKeyring(c); err != nil {
		return nil, err
	}
	return c, nil
}

func createID(c *CecConfig) string {
	var port string
	u, _ := url.Parse(c.Url)
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
	u, _ := url.Parse(c.Url)
	return fmt.Sprintf("%s@%s", c.User, u.Hostname())
}

// SaveConfigFile saves inside the config file.
func (list *ConfigList) SaveConfigFile() error {
	confData, _ := json.MarshalIndent(&list, "", "\t")
	if err := ioutil.WriteFile(GetConfigFilePath(), confData, 0666); err != nil {
		return fmt.Errorf("could not save the config file, cause: %s", err)
	}
	return nil
}
