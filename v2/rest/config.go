package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	cells_sdk "github.com/pydio/cells-sdk-go/v3"
)

type ConfigList struct {
	ActiveConfigID string
	Configs        map[string]*CecConfig
}

// GetConfigList retrieves the current configurations stored in the config.json file.
func GetConfigList() (*ConfigList, error) {

	// assuming they are located in the default folder
	data, err := ioutil.ReadFile(GetConfigFilePath())
	if err != nil {
		return nil, err
	}

	cfg := &ConfigList{Configs: make(map[string]*CecConfig)}
	err = json.Unmarshal(data, cfg)
	if err == nil {
		return cfg, nil
	}

	var oldConf *cells_sdk.SdkConfig
	// tries to unmarshall with the old format and migrate if necessary
	if err = json.Unmarshal(data, &oldConf); err != nil {
		return nil, fmt.Errorf("unknown config format: %s", err)
	}

	defaultID := "default"
	cfg.ActiveConfigID = defaultID
	cfg = &ConfigList{
		Configs: map[string]*CecConfig{"default": {
			SdkConfig: *oldConf,
		}},
		ActiveConfigID: defaultID,
	}

	return cfg, nil
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
	//TODO retrieve data from keyring
	//if err := ConfigFromKeyring(list.Configs[list.ActiveConfig]); err != nil {
	//	return nil, err
	//}
	return list.Configs[list.ActiveConfigID], nil
}

func (list *ConfigList) updateActiveConfig(cf *CecConfig) error {
	// TODO retrieve from keyring update and push
	//if err := ConfigFromKeyring(list.Configs[list.ActiveConfig]); err != nil {
	//	return err
	//}
	list.Configs[list.ActiveConfigID] = cf
	//if err := ConfigToKeyring(list.Configs[list.ActiveConfig]); err != nil {
	//	return err
	//}
	return nil
}

func AddNewConfig(newConf *CecConfig) (string, error) {
	cl, err := GetConfigList()
	if errors.Is(err, os.ErrNotExist) {
		cl = &ConfigList{Configs: map[string]*CecConfig{}}
	} else {
		if err != nil {
			return "", err
		}
	}

	id := createID(newConf)
	newConf.Label = createLabel(newConf)
	if err := cl.Add(id, newConf); err != nil {
		return "", err
	}

	if err := cl.SaveConfigFile(); err != nil {
		return "", err
	}
	return id, nil
}

func createID(c *CecConfig) string {
	DefaultConfig = c
	uname, e := RetrieveCurrentSessionLogin()
	if e != nil {
		uname = "username_not_found"
	}

	// Also set the username
	c.User = uname

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

	return fmt.Sprintf("%s@%s:%s", uname, u.Hostname(), port)
}

func createLabel(c *CecConfig) string {
	DefaultConfig = c
	uname, e := RetrieveCurrentSessionLogin()
	if e != nil {
		uname = "username_not_found"
	}

	u, _ := url.Parse(c.Url)

	return fmt.Sprintf("%s@%s", uname, u.Hostname())
}

// SaveConfigFile saves inside the config file.
func (list *ConfigList) SaveConfigFile() error {
	confData, _ := json.MarshalIndent(&list, "", "\t")
	if err := ioutil.WriteFile(GetConfigFilePath(), confData, 0666); err != nil {
		return fmt.Errorf("could not save the config file, cause: %s", err)
	}
	return nil
}
