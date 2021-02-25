package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
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
	// PreRunE: func(cmd *cobra.Command, args []string) error {

	// 	fmt.Println("[DEBUG] flags: ")
	// 	fmt.Printf("- serverURL: %s\n", serverURL)
	// 	fmt.Printf("- authType: %s\n", authType)
	// 	fmt.Printf("- idToken: %s\n", idToken)
	// 	fmt.Printf("- login: %s\n", login)
	// 	fmt.Printf("- password: %s\n", password)
	// 	fmt.Printf("- noCache: %v\n", noCache)
	// 	fmt.Printf("- skipKeyring: %v\n", skipKeyring)
	// 	fmt.Printf("- skipVerify: %v\n", skipVerify)

	// 	return nil
	// },

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

	// Legacy flags - TODO: finalise handling of retrocompatibility
	flags.Bool("skipVerify", false, "Skip SSL certificate verification (not recommended)")
	flags.String("idToken", "", "Valid IdToken")

	bindViperFlags(flags, map[string]string{})

	RootCmd.AddCommand(configureCmd)
}
