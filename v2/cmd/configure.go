package cmd

import (
	"fmt"
	"os"

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

	RunE: func(cmd *cobra.Command, args []string) error {
		items := []string{"OAuth2 login (requires a browser access)", "Personal Access Token (unique token generated server-side)", "Client Auth (direct login/password, less secure)"}
		s := promptui.Select{Label: "Select authentication method", Size: 3, Items: items}
		n, _, err := s.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				return fmt.Errorf("operation aborted by user")
			}
			return err
		}

		switch n {
		case 0:
			configureOAuthCmd.Run(cmd, args)
		case 1:
			if err := configurePersonalAccessTokenCmd.RunE(cmd, args); err != nil {
				return err
			}
		case 2:
			configureClientAuthCmd.Run(cmd, args)
		default:
			return fmt.Errorf("no authentication method was selected")
		}
		return nil
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

func notEmpty(input string) error {
	if len(input) == 0 {
		return fmt.Errorf("Field cannot be empty")
	}
	return nil
}

// TODO methode pour faire label
func getIDFromConfig(conf *rest.CecConfig) (id, defaultLabel string) {
	// u, _ := url.Parse()
	// u.Port()
	// u.Scheme http/https 80 / 443
	//user + host + port
	id = ""

	// label lisible

	return id, defaultLabel
}
