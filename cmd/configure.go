package cmd

import (
	"fmt"

	"net/url"

	"encoding/json"
	"io/ioutil"

	"github.com/manifoldco/promptui"
	"github.com/micro/go-log"
	"github.com/pydio/cells-client/rest"
	"github.com/pydio/cells-sdk-go"
	"github.com/spf13/cobra"
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

var ConfigureCmd = &cobra.Command{
	Use:  "configure",
	Long: `DataSources / Workspaces`,
	Run: func(cm *cobra.Command, args []string) {

		newConf := &cells_sdk.SdkConfig{}

		var e error
		// PROMPT URL
		p := promptui.Prompt{Label: "Server Address (provide a valid URL)", Validate: validUrl}
		if newConf.Url, e = p.Run(); e != nil {
			log.Fatal(e)
		}
		u, e := url.Parse(newConf.Url)
		if e != nil {
			log.Fatal(e)
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
			log.Fatal(e)
		}

		// PROMPT CLIENT SECRET
		p = promptui.Prompt{Label: "Client Secret (found in your server pydio.json)", Validate: notEmpty}
		if newConf.ClientSecret, e = p.Run(); e != nil {
			log.Fatal(e)
		}

		// PROMPT LOGIN
		p = promptui.Prompt{Label: "User Login", Validate: notEmpty}
		if newConf.User, e = p.Run(); e != nil {
			log.Fatal(e)
		}

		// PROMPT PASSWORD
		p = promptui.Prompt{Label: "User Password", Mask: '*', Validate: notEmpty}
		if newConf.Password, e = p.Run(); e != nil {
			log.Fatal(e)
		}

		// Test a simple PING with this config before saving!
		fmt.Println(promptui.IconWarn + " Testing this configuration before saving")
		rest.DefaultConfig = newConf
		if _, _, e := rest.GetApiClient(); e != nil {
			fmt.Println("\r" + promptui.IconBad + " Could not connect to server, please recheck your configuration")
			fmt.Println("   Error was " + e.Error())
			return
		} else {
			fmt.Println("\r" + promptui.IconGood + " Successfully logged to server")
		}

		// Now save config!
		filePath := rest.DefaultConfigFilePath()
		data, _ := json.Marshal(newConf)
		e = ioutil.WriteFile(filePath, data, 0755)
		if e != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration file! " + e.Error())
		} else {
			fmt.Println(promptui.IconGood + " Configuration saved, you can now use the client!")
		}

	},
}

func init() {

	flags := ConfigureCmd.PersistentFlags()

	flags.StringVarP(&configHost, "url", "u", "", "HTTP URL to server")
	flags.StringVarP(&configKey, "apiKey", "k", "", "OIDC Client ID")
	flags.StringVarP(&configSecret, "apiSecret", "s", "", "OIDC Client Secret")
	flags.StringVarP(&configUser, "login", "l", "", "User login")
	flags.StringVarP(&configPwd, "password", "p", "", "User password")
	flags.BoolVar(&configSkipVerify, "skipVerify", false, "Skip SSL certificate verification (not recommended)")

	RootCmd.AddCommand(ConfigureCmd)
}
