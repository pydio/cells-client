package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/rest"
	cellsSdk "github.com/pydio/cells-sdk-go/v4"
)

func init() {
	configureCmd.AddCommand(legacyCheckKeyringCmd)
	configCmd.AddCommand(configAddCmd)
	// Legacy will be soon removed
	RootCmd.AddCommand(configureCmd)
}

var configAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Configure a new connection to a running server and persist credentials locally",
	Long: `
DESCRIPTION

  Launch an interactiveClientAuth process to configure a connection to a running Cells server.
  By default, we use a secure OAuth2 process based on 'Authorization Code' Grant.

  If necessary, you might use an alternative authorization process and/or execute this process non-interactively calling one of the defined sub-commands.

  Once a connection with the server is established, it stores necessary information locally, keeping the sensitive bits encrypted in the client machine keyring.
  If you want to forget a connection, the config file and keyring can be cleant out by calling the 'config rm' subcommand.

WARNING

If no keyring is defined in the client machine, all information is stored in *clear text* in a config file of the working directory.
`,

	Run: func(cmd *cobra.Command, args []string) {
		items := []string{"OAuth2 login (requires a browser access)", "Personal Access Token (unique token generated server-side)", "Client Auth (direct login/password, less secure)"}
		s := promptui.Select{Label: "Select authentication method", Size: 3, Items: items}
		n, _, err := s.Run()
		if err != nil {
			if errors.Is(err, promptui.ErrInterrupt) {
				log.Fatal("operation aborted by user")
			}
			log.Fatal(err)
		}

		switch n {
		case 0:
			configureOAuthCmd.Run(cmd, args)
		case 1:
			configurePersonalAccessTokenCmd.Run(cmd, args)
		case 2:
			configureClientAuthCmd.Run(cmd, args)
		default:
			log.Fatal("no authentication method was selected")
		}
	},
}

var configureCmd = &cobra.Command{
	Use:    "configure",
	Hidden: true,
	Short:  "[Deprecated] Configure a connection to a running server and persist credentials locally for later use",
	Long: `
DESCRIPTION

  Launch an interactiveClientAuth process to configure a connection to a running Cells server.
  By default, we use a secure OAuth2 process based on 'Authorization Code' Grant.

  If necessary, you might use an alternative authorization process and/or execute this process non-interactively calling one of the defined sub-commands.

  Once a connection with the server established, it stores necessary information locally, keeping the sensitive bits encrypted in the client machine keyring.
  If you want to forget a connection, the config file and keyring can be cleant out by calling the 'clear' subcommand.

WARNING

This command has been deprecated in favor of '` + os.Args[0] + ` config add' command and will be removed in the future major version.
Please update your scripts to be ready.

`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[Warning] this command has been deprecated and will be removed in next major version.")
		fmt.Println("Please rather use '" + os.Args[0] + " config add' that is its new name.")
		fmt.Println("")

		items := []string{"OAuth2 login (requires a browser access)", "Personal Access Token (unique token generated server-side)", "Client Auth (direct login/password, less secure)"}
		s := promptui.Select{Label: "Select authentication method", Size: 3, Items: items}
		n, _, err := s.Run()
		if err != nil {
			if errors.Is(err, promptui.ErrInterrupt) {
				log.Fatal("operation aborted by user")
			}
			log.Fatal(err)
		}

		switch n {
		case 0:
			configureOAuthCmd.Run(cmd, args)
		case 1:
			configurePersonalAccessTokenCmd.Run(cmd, args)
		case 2:
			configureClientAuthCmd.Run(cmd, args)
		default:
			log.Fatal("no authentication method was selected")
		}
	},
}

var legacyCheckKeyringCmd = &cobra.Command{
	Use:    "check-keyring",
	Short:  "[Deprecated] Rather use '" + os.Args[0] + " config check-keyring'",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("[Warning] this command has been deprecated and will be removed in next major version.")
		fmt.Println("Please rather use '" + os.Args[0] + " config check-keyring' that is its new name. Yet, launching the check...")
		fmt.Println("")
		checkKeyringCmd.Run(cmd, args)
	},
}

// Local helpers

func notEmpty(input string) error {
	if len(input) == 0 {
		return fmt.Errorf("field cannot be empty")
	}
	return nil
}

func persistConfig(newConf *rest.CecConfig) error {

	err := rest.UpdateConfig(newConf)
	if err != nil {
		return fmt.Errorf(promptui.IconBad + " could not save configuration: " + err.Error())
	}
	fmt.Printf("%s Configuration saved. You can now use the Cells Client to interact as %s with %s\n", promptui.IconGood, newConf.User, newConf.Url)
	return nil
}

var configRelogCmd = &cobra.Command{
	Use:   "relog",
	Short: "Re-run the authentication process for an existing connection",
	RunE: func(cmd *cobra.Command, args []string) error {
		cl, err := rest.GetConfigList()
		if err != nil {
			return err
		}

		var items []string
		for k := range cl.Configs {
			items = append(items, k)
		}

		sort.Strings(items)

		var initialCursor int
		for i, v := range items {
			if cl.ActiveConfigID == v {
				initialCursor = i
			}
		}

		if len(items) > 0 {
			pSelect := promptui.Select{Label: "Select the account you want to auth against anew", Items: items, Size: len(items), CursorPos: initialCursor}
			_, accountId, err := pSelect.Run()
			if err != nil {
				return err
			}
			conf, err := cl.GetStoredConfig(cmd.Context(), accountId)
			if err != nil {
				return err
			}

			fmt.Printf("Re-run the authentication process for account %s\n", accountId)

			err = relog(conf)
			if err != nil {
				return err
			}

			// TODO  Do we also want to set the newly re-authenticated connection as the current
			//  and to save thus the configs file
			// if err := cl.SetActiveConfig(accountId); err != nil {
			// 				return err
			// 		}
			// fmt.Printf("The active configuration is: %s\n", cl.ActiveConfigID)
			// if err := cl.SaveConfigFile(); err != nil {
			// 	return err
			// }
		}
		return nil
	},
}

func relog(oldConf *rest.CecConfig) error {

	var err error
	switch oldConf.AuthType {
	case cellsSdk.AuthTypeOAuth:
		err = oAuthInteractive(oldConf)
	case cellsSdk.AuthTypePat:
		err = interractiveTokenAuth(oldConf)
	case cellsSdk.AuthTypeClientAuth:
		err = interactiveClientAuth(oldConf)
	default:
		log.Fatalf("Cannot relog for authentication type: %s", oldConf.AuthType)
	}
	err = persistConfig(oldConf)
	if err != nil {
		log.Fatal(err.Error())
	}
	return err
}
