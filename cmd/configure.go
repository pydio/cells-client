package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/manifoldco/promptui"
	"github.com/micro/go-log"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
	cells_sdk "github.com/pydio/cells-sdk-go"
)

var (
	configHost       string
	configKey        string
	configSecret     string
	configUser       string
	configPwd        string
	configSkipVerify bool
)

func notEmpty(input string) error {
	if len(input) > 0 {
		return nil
	} else {
		return fmt.Errorf("Field cannot be empty!")
	}
}

func validUrl(input string) error {
	if len(input) == 0 {
		return fmt.Errorf("Field cannot be empty!")
	}
	u, e := url.Parse(input)
	if e != nil || u == nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("Provide a valid URL")
	}
	return nil
}

func interractive(newConf *cells_sdk.SdkConfig) error {
	var e error
	// PROMPT URL
	p := promptui.Prompt{Label: "Server Address (provide a valid URL)", Validate: validUrl}
	if newConf.Url, e = p.Run(); e != nil {
		return e
	}
	u, e := url.Parse(newConf.Url)
	if e != nil {
		return e
	}
	if u.Scheme == "https" {
		// PROMPT SKIP VERIFY
		p2 := promptui.Select{Label: "Skip SSL Verification? (not recommended)", Items: []string{"Yes", "No"}}
		if _, y, e := p2.Run(); y == "Yes" && e != nil {
			newConf.SkipVerify = true
		}
	}

	// PROMPT CLIENT ID
	p = promptui.Prompt{Label: "Client ID (found in your server pydio.json)", Validate: notEmpty, Default: "cells-front"}
	if newConf.ClientKey, e = p.Run(); e != nil {
		return e
	}

	// PROMPT CLIENT SECRET
	p = promptui.Prompt{Label: "Client Secret (found in your server pydio.json)", Validate: notEmpty}
	if newConf.ClientSecret, e = p.Run(); e != nil {
		return e
	}

	// PROMPT LOGIN
	p = promptui.Prompt{Label: "User Login", Validate: notEmpty}
	if newConf.User, e = p.Run(); e != nil {
		return e
	}

	// PROMPT PASSWORD
	p = promptui.Prompt{Label: "User Password", Mask: '*', Validate: notEmpty}
	if newConf.Password, e = p.Run(); e != nil {
		return e
	}

	// Test a simple PING with this config before saving!
	fmt.Println(promptui.IconWarn + " Testing this configuration before saving")
	rest.DefaultConfig = newConf
	if _, _, e := rest.GetApiClient(); e != nil {
		fmt.Println("\r" + promptui.IconBad + " Could not connect to server, please recheck your configuration")
		fmt.Println("   Error was " + e.Error())
		return fmt.Errorf("Test connection failed.")
	}
	fmt.Println("\r" + promptui.IconGood + " Successfully logged to server")
	return nil
}

func noninterractive(conf *cells_sdk.SdkConfig) error {

	conf.Url = configHost
	conf.ClientKey = configKey
	conf.ClientSecret = configSecret
	conf.User = configUser
	conf.Password = configPwd
	conf.SkipVerify = configSkipVerify

	// Insure values are legal
	if err := validUrl(conf.Url); err != nil {
		return fmt.Errorf("URL %s is not valid: %s", conf.Url, err.Error())
	}

	// Test a simple PING with this config before saving!
	rest.DefaultConfig = conf
	if _, _, e := rest.GetApiClient(); e != nil {
		return fmt.Errorf("Test connection to newly configured server failed.")
	}

	return nil
}

var configureCmd = &cobra.Command{
	Use:        "configure",
	Short:      "Retrieve token using Grant Credentials",
	Deprecated: "use oauth command instead",
	Long:       `Retrieve token using Grant Credentials`,
	Run: func(cm *cobra.Command, args []string) {

		var err error
		newConf := &cells_sdk.SdkConfig{}

		if notEmpty(configHost) == nil && notEmpty(configKey) == nil && notEmpty(configSecret) == nil && notEmpty(configUser) == nil && notEmpty(configPwd) == nil {
			err = noninterractive(newConf)
		} else {
			err = interractive(newConf)
		}
		if err != nil {
			log.Fatal(err)
		}

		// Now save config!
		filePath := rest.DefaultConfigFilePath()
		data, _ := json.Marshal(newConf)
		err = ioutil.WriteFile(filePath, data, 0755)
		if err != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration file! " + err.Error())
		} else {
			fmt.Printf("%s Configuration saved, you can now use the client to interract with %s.\n", promptui.IconGood, newConf.Url)
		}
	},
}

func init() {

	flags := configureCmd.PersistentFlags()

	flags.StringVarP(&configHost, "url", "u", "", "HTTP URL to server")
	flags.StringVarP(&configKey, "apiKey", "k", "", "OIDC Client ID")
	flags.StringVarP(&configSecret, "apiSecret", "s", "", "OIDC Client Secret")
	flags.StringVarP(&configUser, "login", "l", "", "User login")
	flags.StringVarP(&configPwd, "password", "p", "", "User password")
	flags.BoolVar(&configSkipVerify, "skipVerify", false, "Skip SSL certificate verification (not recommended)")

	RootCmd.AddCommand(configureCmd)
}
