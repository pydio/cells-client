package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/manifoldco/promptui"
	"github.com/ory/viper"
	"github.com/pydio/cells-client/v2/rest"
	"github.com/spf13/cobra"
)

var (
	serverURL string
	idToken   string
	authType  string
	login     string
	password  string

	skipKeyring bool
	skipVerify  bool
	noCache     bool
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure a connection to a running server and locally persist credentials for later use",
	Long: `
Launch an interactive process to configure a connection to a running Pydio Cells server instance.
By default, we use a secure OAuth2 process based on 'Authorization Code' Grant.

If necessary, you might use an alternative authorization process and/or execute this process non-interactively calling one of the defined sub-commands.

Once a connection with the server established, it stores necessary information locally, keeping the sensitive bits encrypted in the local machine keyring.
If you want to forget a connection, the config file can be wiped out by calling the 'clear' subcommand.

*WARNING*
If no keyring is defined in the local machine, all information is stored in *clear text* in a config file of the Cells Client working directory.
In such case, do not use the 'client-auth' process.
`,
	PreRunE: func(cmd *cobra.Command, args []string) error {

		// Manually bind to viper instead of flags.StringVar, flags.BoolVar, etc
		serverURL = viper.GetString("url")
		idToken = viper.GetString("authType")
		authType = viper.GetString("login")
		login = viper.GetString("idToken")
		password = viper.GetString("password")

		skipKeyring = viper.GetBool("skip-keyring")
		skipVerify = viper.GetBool("skip-verify")
		noCache = viper.GetBool("no-cache")

		return nil
	},

	Run: func(cmd *cobra.Command, args []string) {

		s := promptui.Select{Label: "Select authentication method", Size: 3, Items: []string{"Personal Access Token (unique token generated server-side)", "OAuth2 login (requires a browser access)", "Client Auth (direct login/password, less secure)"}}
		n, _, err := s.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				fmt.Println("Operation aborted by user")
			}
			return
		}

		switch n {
		case 0:
			withPatCmd.Run(cmd, args)
		case 1:
			configureOAuthCmd.Run(cmd, args)
		case 2:
			configureClientAuthCmd.Run(cmd, args)
		default:
			return
		}
	},
}

// saveConfig handle file and/or keyring storage depending on user preference and system.
func saveConfig(config *rest.CecConfig) error {

	// TODO insure config is OK
	// LS ? Retrieve UserName

	uname, e := rest.RetrieveCurrentSessionLogin()
	if e != nil {
		return fmt.Errorf("could not connect to distant server with provided parameters. Discarding change")
	}
	config.User = uname

	if !config.SkipKeyring {
		if err := rest.ConfigToKeyring(config); err != nil {
			return err
		}
	}

	file := rest.GetConfigFilePath()
	data, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(file, data, 0600); err != nil {
		return err
	}

	return nil
}

func init() {
	flags := configureCmd.PersistentFlags()

	flags.StringP("url", "u", "", "Server serverURL")
	flags.StringP("authType", "a", "", "Authorizaton mechanism used: Personnal Access Token (Default), OAuth2 flow or Client Credentials")
	flags.StringP("login", "l", "", "User login")
	flags.StringP("idToken", "t", "", "Valid IdToken")
	flags.StringP("password", "p", "", "User password")

	flags.Bool("skip-verify", false, "Skip SSL certificate verification (not recommended)")
	// Duplicate
	flags.Bool("skipVerify", false, "Skip SSL certificate verification (not recommended)")
	flags.Bool("skip-keyring", false, "Explicitly tell the tool to *NOT* try to use a keyring, even if present. Warning: sensitive information will be stored in clear text.")
	flags.Bool("no-cache", false, "Force token refresh at each call. This might slow down scripts with many calls.")

	RootCmd.AddCommand(configureCmd)
}
