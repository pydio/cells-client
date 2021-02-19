package cmd

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

const authTypeToken = "token"

var (
	token     string
	serverURL string
)

var configureTokenAuthCmd = &cobra.Command{
	Use:   authTypeToken,
	Short: "Configure Authentication using the personal token",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		var p promptui.Prompt
		newConf := new(rest.CecConfig)

		newConf.SkipKeyring = skipKeyring

		if token != "" && serverURL != "" {

			newConf.IdToken = token
			newConf.Url = serverURL

		} else { // No Flags : prompt user

			p = promptui.Prompt{Label: "Server URL", Validate: validUrl}
			newConf.Url, err = p.Run()
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
	configureCmd.AddCommand(configureTokenAuthCmd)
	configureTokenAuthCmd.Flags().StringVarP(&token, "token", "t", "", "personal token")
	configureTokenAuthCmd.Flags().StringVarP(&serverURL, "url", "u", "", "Server serverURL")
}
