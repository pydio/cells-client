package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/pydio/cells-client/v2/rest"
	cells_sdk "github.com/pydio/cells-sdk-go"
	"github.com/spf13/cobra"
)

var noKeyringDefined bool

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear current configuration",
	Long:  "Clear current authentication data from your local keyring",
	Run: func(cmd *cobra.Command, args []string) {
		filePath := rest.GetConfigFilePath()
		if s, err := ioutil.ReadFile(filePath); err == nil {
			var c cells_sdk.SdkConfig
			if err = json.Unmarshal(s, &c); err == nil {
				if !noKeyringDefined {
					// First clean the keyring
					if err := rest.ClearKeyring(&c); err == nil {
						fmt.Println(promptui.IconGood + " Removed tokens from keychain")
					} else {
						fmt.Println(promptui.IconBad + " Error while removing token from keyring: " + err.Error())
					}
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
	flags := clearCmd.PersistentFlags()
	helpMsg := "Explicitly tell the tool to *NOT* try to use a keyring. Only use this flag if you really know what your are doing: some sensitive information will end up stored on your file system in clear text."
	flags.BoolVar(&noKeyringDefined, "no-keyring", false, helpMsg)
	RootCmd.AddCommand(clearCmd)
}
