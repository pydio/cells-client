package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var noKeyringDefined bool

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear current configuration",
	Long: `
DESCRIPTION

	Clear current authentication data from your client machine.
	
	It deletes the ` + confFileName + ` from Cells Client working directory.
	It also removes the sensitive data that has been stored in the keyring, if present.
`,
	Run: func(cmd *cobra.Command, args []string) {
		filePath := rest.DefaultConfigFilePath()
		if s, err := ioutil.ReadFile(filePath); err == nil {
			config := new(rest.CecConfig)
			if err = json.Unmarshal(s, &config); err == nil {
				if !config.SkipKeyring {
					// First clean the keyring
					if err := rest.CheckKeyring(); err != nil {
						fmt.Println(promptui.IconWarn + "No Keyring found on this system")
					} else if err := rest.ClearKeyring(config); err != nil {
						fmt.Println(promptui.IconBad + " Error while removing token from keyring: " + err.Error())
					} else {
						fmt.Println(promptui.IconGood + " Removed tokens from keychain")
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
