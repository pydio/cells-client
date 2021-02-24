package cmd

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

var withPatCmd = &cobra.Command{
	Use:   "token",
	Short: "Configure Authentication using the personal token",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var p promptui.Prompt
		newConf := &rest.CecConfig{
			SkipKeyring: skipKeyring,
			AuthType:    common.PatType,
		}

		// non interactive
		if idToken != "" && serverURL != "" {
			newConf.IdToken = idToken
			newConf.Url = serverURL
		} else { // interactive

			p = promptui.Prompt{Label: "Server URL", Validate: validUrl}
			newConf.Url, err = p.Run()
			// clean spaces in the URL
			newConf.Url = strings.TrimSpace(newConf.Url)
			if err != nil {
				if err == promptui.ErrInterrupt {
					fmt.Println("Operation aborted by user")
					return
				}
				fmt.Println(promptui.IconBad + "URL is not valid" + err.Error())
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

		err = saveConfig(newConf)
		if err != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration file! " + err.Error())
		} else {
			fmt.Printf("%s Configuration saved, you can now use the client to interract with %s.\n", promptui.IconGood, newConf.Url)
		}
	},
}

func init() {
	configureCmd.AddCommand(withPatCmd)
}
