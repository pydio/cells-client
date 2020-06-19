package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/pydio/cells-client/rest"
	cells_sdk "github.com/pydio/cells-sdk-go"
	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear current configuration",
	Long:  "Clear current authentication data from your local keyring",
	Run: func(cmd *cobra.Command, args []string) {
		filePath := rest.DefaultConfigFilePath()
		if s, err := ioutil.ReadFile(filePath); err == nil {
			var c cells_sdk.SdkConfig
			if err = json.Unmarshal(s, &c); err == nil {
				if err := rest.ClearKeyring(&c); err == nil {
					fmt.Println(promptui.IconGood + " Removed tokens from keychain")
				} else {
					fmt.Println(promptui.IconBad + " Error while removing token from keyring " + err.Error())
				}
			}
		}
		if err := os.Remove(filePath); err != nil {
			log.Fatal(err)
		}
		fmt.Println(promptui.IconGood + " Successfully removed config file")
	},
}

func init() {
	RootCmd.AddCommand(clearCmd)
}
