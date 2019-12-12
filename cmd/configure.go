package cmd

import (
	"github.com/spf13/cobra"
)

var (
	skipKeyring bool
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

		// Call OAuth grant flow by default
		configureOAuthCmd.Run(cm, args)

		// switch configAuthType {
		// case authTypeClientAuth:
		// configureClientAuthCmd.Run(cm, args)
		// break
		// case authTypeOAuth:
		// default:
		// configureOAuthCmd.Run(cm, args)
		// }
	},
}

func init() {

	flags := configureCmd.PersistentFlags()
	helpMsg := "Explicitly tell the tool to *NOT* try to use a keyring. Only use this flag if you really know what your are doing: some sensitive information will end up stored on your file system in clear text."
	flags.BoolVar(&skipKeyring, "no-keyring", false, helpMsg)
	RootCmd.AddCommand(configureCmd)
}
