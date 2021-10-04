package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/manifoldco/promptui"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage authentication profiles.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configListCmd = &cobra.Command{
	Use:   "ls",
	Short: "List the current authentication profiles.",
	RunE: func(cmd *cobra.Command, args []string) error {
		list, err := rest.GetConfigList()
		if err != nil {
			return err
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"label", "user", "URL", "type", "Active"})
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetAutoWrapText(false)

		for id, v := range list.Configs {
			if list.ActiveConfigID == id {
				table.Append([]string{v.Label, v.User, v.Url, v.AuthType, "\u2713"})
			} else {
				table.Append([]string{v.Label, v.User, v.Url, v.AuthType, ""})
			}
		}
		table.Render()

		return nil
	},
}
var configUseCmd = &cobra.Command{
	Use:   "use",
	Short: "Define as active, one of the current authentication profiles.",
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

		sort.Strings(items)

		var initialCursor int
		for i, v := range items {
			if cl.ActiveConfigID == v {
				initialCursor = i
			}
		}

		if len(items) > 0 {
			pSelect := promptui.Select{Label: "Select the account you want to use", Items: items, Size: len(items), CursorPos: initialCursor}
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

		fmt.Printf("The active configuration is: %s\n", cl.ActiveConfigID)
		return nil
	},
}
var configRemoveCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove a profile from the cells-client authentication profiles.",
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

		var removed string
		var active string
		var index int
		if len(items) > 0 {
			pSelect := promptui.Select{Label: "Select a configuration to remove", Items: items, Size: len(items)}
			index, removed, err = pSelect.Run()
			if err != nil {
				return err
			}

			items = append(items[:index], items[index+1:]...)

			if err := cl.Remove(removed); err != nil {
				return err
			}

			if len(items) > 1 {
				pSelect2 := promptui.Select{Label: "Please select the new active configuration", Items: items, Size: len(items)}
				_, active, err = pSelect2.Run()
				if err != nil {
					return err
				}
			} else {
				active = items[0]
			}

			if err := cl.SetActiveConfig(active); err != nil {
				return err
			}

		}
		if err := cl.SaveConfigFile(); err != nil {
			return err
		}

		cmd.Printf("Removed the following configuration %s\n\n", removed)
		cmd.Printf("The new active configuration is: %s\n", active)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configUseCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configRemoveCmd)
	RootCmd.AddCommand(configCmd)

}
