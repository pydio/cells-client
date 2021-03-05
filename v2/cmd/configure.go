package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

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
			fmt.Println(promptui.IconWarn + " " + rest.NoKeyringMsg)
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
