package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/micro/go-log"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
	cells_sdk "github.com/pydio/cells-sdk-go"
)

const authTypeClientAuth = "client-auth"

var (
	configHost       string
	configKey        string
	configSecret     string
	configUser       string
	configPwd        string
	configSkipVerify bool
)

var configureClientAuthCmd = &cobra.Command{
	Use:   authTypeClientAuth,
	Short: "Connect to the server directly using the Client Credentials",
	Long: `
Launch an interractive process to gather necessary client information to configure a connection to a running Pydio Cells server instance.

You can typically use the static credentials (Id and Secret) defined in the "services"."pydio.grpc.auth"."staticClients" section of your server's "pydio.json" config file, 
and a valid userName/password with enough permissions to achieve what you want on the server.

Please beware that this sentitive information will be stored in clear text if you do not have a *correctly configured and running* keyring on your client machine.

You can also go through the whole process in a non-interractive manner by using the provided flags.
`,
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
		if err := rest.ConfigToKeyring(newConf); err != nil {
			fmt.Println(promptui.IconWarn + " Cannot save token in keyring! " + err.Error())
		}
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

func interractive(newConf *cells_sdk.SdkConfig) error {
	var e error
	// PROMPT URL
	p := promptui.Prompt{Label: "Server Address (provide a valid URL)", Validate: validUrl}
	if newConf.Url, e = p.Run(); e != nil {
		return e
	} else {
		newConf.Url = strings.Trim(newConf.Url, " ")
	}

	u, e := url.Parse(newConf.Url)
	if e != nil {
		return e
	}
	if u.Scheme == "https" {
		// PROMPT SKIP VERIFY
		p2 := promptui.Select{Label: "Skip SSL Verification? (not recommended)", Items: []string{"No", "Yes"}}
		if _, y, e := p2.Run(); y == "Yes" && e == nil {
			newConf.SkipVerify = true
		}
	}

	// PROMPT CLIENT ID
	p = promptui.Prompt{
		Label:     "Client ID (found in your server pydio.json)",
		Validate:  notEmpty,
		Default:   "cells-front",
		AllowEdit: true,
	}
	if newConf.ClientKey, e = p.Run(); e != nil {
		return e
	}

	// PROMPT CLIENT SECRET
	p = promptui.Prompt{
		Label:    "Client Secret (found in your server pydio.json)",
		Validate: notEmpty,
	}
	if newConf.ClientSecret, e = p.Run(); e != nil {
		return e
	}

	// PROMPT LOGIN
	p = promptui.Prompt{
		Label:    "User Login",
		Validate: notEmpty,
	}
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
		fmt.Println("Cause: " + e.Error())
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
		return fmt.Errorf("Could not connect to newly configured server failed, cause: ", e.Error())
	}

	return nil
}

func validUrl(input string) error {
	// Warning: trim must also be performed when retrieving the final value.
	// Here we only validate that the trimed input is valid, but do not modify it.
	input = strings.Trim(input, " ")
	if len(input) == 0 {
		return fmt.Errorf("Field cannot be empty!")
	}
	u, e := url.Parse(input)
	if e != nil || u == nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("Provide a valid URL")
	}
	return nil
}

func notEmpty(input string) error {
	if len(input) == 0 {
		return fmt.Errorf("Field cannot be empty!")
	}
	return nil
}

func init() {

	flags := configureClientAuthCmd.PersistentFlags()

	flags.StringVarP(&configHost, "url", "u", "", "HTTP URL to server")
	flags.StringVarP(&configKey, "apiKey", "k", "", "OIDC Client ID")
	flags.StringVarP(&configSecret, "apiSecret", "s", "", "OIDC Client Secret")
	flags.StringVarP(&configUser, "login", "l", "", "User login")
	flags.StringVarP(&configPwd, "password", "p", "", "User password")
	flags.BoolVar(&configSkipVerify, "skipVerify", false, "Skip SSL certificate verification (not recommended)")

	configureCmd.AddCommand(configureClientAuthCmd)
}
