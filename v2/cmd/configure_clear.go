package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var noKeyringDefined bool

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all configuration",
	Long: `
DESCRIPTION

	Clear all authentication data from your client machine.
	
	It deletes the ` + confFileName + ` from Cells Client working directory.
	It also removes the sensitive data that has been stored in the keyring, if present.
`,
	Run: func(cmd *cobra.Command, args []string) {

		prompt := promptui.Prompt{
			Label:     "Are you sure you wish to erase the configuration ?",
			IsConfirm: true,
		}

		_, err := prompt.Run()
		if err != nil {
			cmd.Println("Operation aborted nothing was removed")
			return
		}

		filePath := rest.DefaultConfigFilePath()
		configs, err := rest.GetConfigList()
		if err != nil {
			log.Fatal("could not retrieve config list, aborting: ", err)
		}

		for id, conf := range configs.Configs {
			if !conf.SkipKeyring {
				err = rest.ClearKeyring(conf)
				if err != nil {
					log.Fatalf("could not clear keyring for %s: %s \n ==> Aborting...", id, err.Error())
				}
			}
		}
		if err := os.Remove(filePath); err != nil {
			log.Fatal(err)
		}
		fmt.Println(promptui.IconGood + " All defined accounts have been erased.")
	},
}

func init() {
	RootCmd.AddCommand(clearCmd)
}
