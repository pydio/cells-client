package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	configAuthType string
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure a connection to a running server and locally persist credentials for later use",
	Long: `
Launch an interractive process to configure a connection to a running Pydio Cells server instance.
By default, we use a secure OAuth2 process based on 'Authorization Code' Grant.

If necessary, you might use an alternative authorization process and/or execute this process non-interactively calling one of the defined sub-commands.

Once a connection with the server established, it stores necessary information locally, keeping the sensitive bits encrypted in the local machine keyring.
If you want to forget a connection, the config file can be wiped out by calling the 'clear' subcommand.

*WARNING*
If no keyring is defined in the local machine, all information is stored in *clear text* in a config file of the Cells Client working directory.
In such case, do not use the 'client-auth' process.
`,
	Run: func(cm *cobra.Command, args []string) {
		switch configAuthType {
		case authTypeClientAuth:
			configureClientAuthCmd.Run(cm, args)
			break
		case authTypeOAuth:
		default:
			configureOAuthCmd.Run(cm, args)
		}
	},
}

func init() {

	flags := configureCmd.PersistentFlags()
	helpMsg := fmt.Sprintf("Choose the authentication process you want to use: %s (default) or %s", authTypeOAuth, authTypeClientAuth)
	flags.StringVar(&configHost, "auth-type", "", helpMsg)
	RootCmd.AddCommand(configureCmd)
}
