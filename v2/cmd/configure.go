package cmd

import (
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	skipKeyring bool
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
	Run: func(cmd *cobra.Command, args []string) {

		s := promptui.Select{Label: "Select authentication method", Size: 3, Items: []string{"Personal Access Token (unique token generated server-side)", "OAuth2 login (requires a browser access)", "Client Auth (direct login/password, less secure)"}}
		n, _, err := s.Run()
		if err != nil {
			return
		}

		switch n {
		case 0:
			configureTokenAuthCmd.Run(cmd, args)
		case 1:
			configureOAuthCmd.Run(cmd, args)
		case 2:
			configureClientAuthCmd.Run(cmd, args)
		default:
			return
		}
	},
}

func init() {

	flags := configureCmd.PersistentFlags()
	helpMsg := "Explicitly tell the tool to *NOT* try to use a keyring. Only use this flag if you really know what your are doing: some sensitive information will end up stored on your file system in clear text."
	flags.BoolVar(&skipKeyring, "no-keyring", false, helpMsg)
	RootCmd.AddCommand(configureCmd)
}
