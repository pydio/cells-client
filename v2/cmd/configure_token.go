package cmd

import (
	"errors"
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

var configurePersonalAccessTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Configure Authentication using a Personal Access Token",
	Long: `
DESCRIPTION

  Configure your Cells Client to connect to your distant server using a Personal Acces Token.
  A token can be generated with the 'token' command for a given user on the server side (not in this client),
  see 'cells admin user token --help' for further details.

  Please beware that the Personal Access Token will be stored in clear text if you do not have a **correctly configured and running** keyring on your client machine.

  This is the preferred way to handle authentication between Cells and your client.
`,
	RunE: func(cmd *cobra.Command, args []string) error {

		var err error
		var p promptui.Prompt
		newConf := &rest.CecConfig{
			AuthType:    common.PatType,
			SkipKeyring: true,
		}

		// non interactive
		if token != "" && serverURL != "" {
			newConf.IdToken = token
			newConf.Url = serverURL
		} else {

			// interactive
			p = promptui.Prompt{Label: "Server URL", Validate: rest.ValidURL}
			newConf.Url, err = p.Run()
			if err != nil {
				if errors.Is(err, promptui.ErrInterrupt) {
					return fmt.Errorf("operation aborted by user")
				}
				return fmt.Errorf("%s URL is not valid %s", promptui.IconBad, err.Error())
			}
			newConf.Url, err = rest.CleanURL(newConf.Url)
			if err != nil {
				return fmt.Errorf("%s %s", promptui.IconBad, err.Error())
			}

			p = promptui.Prompt{Label: "Token"}
			newConf.IdToken, err = p.Run()
			if err != nil {
				if errors.Is(err, promptui.ErrInterrupt) {
					return fmt.Errorf("operation aborted by user")
				}
				return err
			}
		}

		label, err := rest.AddNewConfig(newConf)
		if err != nil {
			return err
		}

		cmd.Println("Config saved under:", label)
		return nil
	},
}

func init() {
	configureCmd.AddCommand(configurePersonalAccessTokenCmd)
}
