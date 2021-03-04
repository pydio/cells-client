package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

const noKeyringMsg = "Could not access local keyring: sensitive information like token or password will end up stored in clear text in the client machine."

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure a connection to a running server and persist credentials locally for later use",
	Long: `
DESCRIPTION

  Launch an interactive process to configure a connection to a running Cells server.
  By default, we use a secure OAuth2 process based on 'Authorization Code' Grant.

  If necessary, you might use an alternative authorization process and/or execute this process non-interactively calling one of the defined sub-commands.

  Once a connection with the server established, it stores necessary information locally, keeping the sensitive bits encrypted in the client machine keyring.
  If you want to forget a connection, the config file and keyring can be cleant out by calling the 'clear' subcommand.

WARNING

If no keyring is defined in the client machine, all information is stored in *clear text* in a config file of the Cells Client working directory.
`,

	Run: func(cmd *cobra.Command, args []string) {

		s := promptui.Select{Label: "Select authentication method", Size: 3, Items: []string{"OAuth2 login (requires a browser access)", "Personal Access Token (unique token generated server-side)", "Client Auth (direct login/password, less secure)"}}
		n, _, err := s.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				fmt.Println("Operation aborted by user")
			}
			return
		}

		switch n {
		case 0:
			configureOAuthCmd.Run(cmd, args)
		case 1:
			withPatCmd.Run(cmd, args)
		case 2:
			configureClientAuthCmd.Run(cmd, args)
		default:
			return
		}
	},
}

func init() {
	configureCmd.AddCommand(checkKeyringCmd)
	RootCmd.AddCommand(configureCmd)
}

var checkKeyringCmd = &cobra.Command{
	Use:   "check-keyring",
	Short: "Try to store and retrieve a dummy value in local keyring to test it",
	Long: `
DESCRIPTION

  Helper command to check if a keyring is present and correctly configured 
  in the client machine by simply storing and retrieving a dummy password.
`,
	Run: func(cm *cobra.Command, args []string) {

		if err := rest.CheckKeyring(); err != nil {
			fmt.Println(promptui.IconWarn + " " + noKeyringMsg)
			os.Exit(1)
		} else {
			fmt.Println(promptui.IconGood + " Keyring seems to be here and working.")
		}
	},
}

// Local helpers

func validURL(input string) error {
	// Warning: trim must also be performed when retrieving the final value.
	// Here we only validate that the trimmed input is valid, but do not modify it.
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return fmt.Errorf("Field cannot be empty")
	}
	u, e := url.Parse(input)
	if e != nil || u == nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("Please, provide a valid URL")
	}
	return nil
}

func notEmpty(input string) error {
	if len(input) == 0 {
		return fmt.Errorf("Field cannot be empty")
	}
	return nil
}

// saveConfig handle file and/or keyring storage depending on user preference and system.
func saveConfig(config *rest.CecConfig) error {

	var err error
	oldConfig := rest.DefaultConfig
	defer func() {
		if err != nil {
			rest.DefaultConfig = oldConfig
		}
	}()

	rest.DefaultConfig = config

	uname, e := rest.RetrieveCurrentSessionLogin()
	if e != nil {
		err = e
		return fmt.Errorf("could not connect to distant server with provided parameters. Discarding change")
	}
	config.User = uname

	if !config.SkipKeyring {
		if err = rest.ConfigToKeyring(config); err != nil {
			// We still save info in clear text but warn the user
			fmt.Println(promptui.IconWarn + " " + noKeyringMsg)
			// Force skip keyring flag in the config file to be explicit
			config.SkipKeyring = true
		}
	}

	file := rest.GetConfigFilePath()

	// Add version before saving the config
	config.CreatedAtVersion = common.Version

	data, e := json.MarshalIndent(config, "", "\t")
	if e != nil {
		err = e
		return e
	}
	if err = ioutil.WriteFile(file, data, 0600); err != nil {
		return err
	}

	fmt.Printf("%s Configuration saved. You can now use the Cells Client to interact as %s with %s\n", promptui.IconGood, config.User, config.Url)

	return nil
}
