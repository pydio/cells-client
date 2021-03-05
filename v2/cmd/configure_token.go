package cmd

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
	"github.com/spf13/cobra"
)

var withPatCmd = &cobra.Command{
	Use:   "token",
	Short: "Configure Authentication using a Personal Access Token",
	Long: `
DESCRIPTION

  Configure your Cells Client to connect to your distant server using a Personal Acces Token.
  A token can be generated with the 'token' command for a given user on the server side (not in this client),
  see 'cells admin user token --help' for further details.

  Please beware that the Personal Access Token will be stored in clear text if you do not have a **correctly configured and running** keyring on your client machine.

  This is the prefered way to handle authentication between Cells and your client.
`,

	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var p promptui.Prompt
		newConf := &rest.CecConfig{
			SkipKeyring: skipKeyring,
			AuthType:    common.PatType,
		}

		// non interactive
		if token != "" && serverURL != "" {
			newConf.IdToken = token
			newConf.Url = serverURL
		} else { // interactive

			p = promptui.Prompt{Label: "Server URL", Validate: validURL}
			newConf.Url, err = p.Run()
			if err != nil {
				if err == promptui.ErrInterrupt {
					fmt.Println("Operation aborted by user")
					return
				}
				fmt.Println(promptui.IconBad + "URL is not valid" + err.Error())
				return
			}
			newConf.Url, err = rest.CleanURL(newConf.Url)
			if err != nil {
				fmt.Println(promptui.IconBad + err.Error())
				return
			}

			p = promptui.Prompt{Label: "Token"}
			newConf.IdToken, err = p.Run()
			if err != nil {
				if err == promptui.ErrInterrupt {
					fmt.Println("Operation aborted by user")
				}
				return
			}
		}

		err = rest.SaveConfig(newConf)
		if err != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration, cause: " + err.Error())
		}
	},
}

func init() {
	configureCmd.AddCommand(withPatCmd)
}
