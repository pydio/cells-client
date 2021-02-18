package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

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

		if token != "" && serverURL != "" {
			// if the flags are set jump to the saving process
			newConf.IdToken = token
			newConf.Url = serverURL
			goto save
		}

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

	save:
		// TODO handle skipKeyring
		if skipKeyring {
			newConf.SkipKeyring = skipKeyring
		}
		err = saveConfig(newConf)
		if err != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration file! " + err.Error())
		} else {
			fmt.Printf("%s Configuration saved, you can now use the client to interract with %s.\n", promptui.IconGood, newConf.Url)
		}
	},
}

func saveConfig(config *rest.CecConfig) error {
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
	configureCmd.AddCommand(configureTokenAuthCmd)
	configureTokenAuthCmd.Flags().StringVarP(&token, "token", "t", "", "personal token")
	configureTokenAuthCmd.Flags().StringVarP(&serverURL, "url", "u", "", "Server serverURL")
}
