package cmd

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var configCmd = &cobra.Command{
	Use: "config",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configListCmd = &cobra.Command{
	Use: "ls",
	RunE: func(cmd *cobra.Command, args []string) error {
		list, err := rest.GetConfigList()
		if err != nil {
			return err
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"label", "user", "URL", "type"})
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetAutoWrapText(false)

		for _, v := range list.Configs {
			table.Append([]string{v.Label, v.User, v.Url, v.AuthType})
		}
		table.Render()

		return nil
	},
}
var configUseCmd = &cobra.Command{
	Use: "use",
	RunE: func(cmd *cobra.Command, args []string) error {
		cl, err := rest.GetConfigList()
		if err != nil {
			return err
		}

		// interactive mode with promptui
		var items []string
		for k := range cl.Configs {
			items = append(items, k)
		}

		if len(items) > 0 {
			pSelect := promptui.Select{Label: "Please select the active configuration", Items: items, Size: len(items)}
			_, result, err := pSelect.Run()
			if err != nil {
				return err
			}

			if err := cl.SetActiveConfig(result); err != nil {
				return err
			}
		}

		if err := cl.SaveConfigFile(); err != nil {
			return err
		}

		fmt.Printf("The active configuration is: %s\n", cl.ActiveConfig)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configUseCmd)
	configCmd.AddCommand(configListCmd)
	RootCmd.AddCommand(configCmd)

}
