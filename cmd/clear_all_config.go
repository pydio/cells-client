package cmd

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/common"
	"github.com/pydio/cells-client/v4/rest"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all configuration",
	Long: `
DESCRIPTION

	Clear all authentication data from your client machine.
	
	It deletes the ` + common.DefaultConfigFileName + ` from Cells Client working directory.
	It also removes the sensitive data that has been stored in the keyring, if present.
`,
	Run: func(cmd *cobra.Command, args []string) {

		if !rmForce {
			prompt := promptui.Prompt{
				Label:     "Are you sure you wish to erase the configuration",
				IsConfirm: true,
			}

			_, err := prompt.Run()
			if err != nil {
				cmd.Println("Operation aborted by user, nothing has been removed.")
				return
			}
		}

		if err := ClearConfig(); err != nil {
			rest.Log.Fatalln(err)
		}

		fmt.Println(promptui.IconGood + " All defined accounts have been erased.")
	},
}

func ClearConfig() error {
	filePath := rest.GetConfigFilePath()
	configs, err := rest.GetConfigList()
	if err != nil {
		return fmt.Errorf("could not retrieve config list, aborting: %s", err)
	}

	for id, conf := range configs.Configs {
		if !conf.SkipKeyring {
			err = rest.ClearKeyring(conf)
			if err != nil {
				return fmt.Errorf("could not clear keyring for %s: %s \n ==> Aborting... ", id, err.Error())
			}
		}
	}
	if err := os.Remove(filePath); err != nil {
		return err
	}
	return nil
}

func init() {
	RootCmd.AddCommand(clearCmd)
	clearCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Non interactive way to clear")
}
