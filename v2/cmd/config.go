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
	Short: "Manage authentication profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configListCmd = &cobra.Command{
	Use:   "ls",
	Short: "List the current authentication profiles",
	RunE: func(cmd *cobra.Command, args []string) error {

		list, err := rest.GetConfigList()
		if err != nil {
			return err
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Active", "Label", "User", "URL", "Type"})
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetAutoWrapText(false)

		// Sorts the keys of the map
		var keys []string
		for k := range list.Configs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, val := range keys {
			if val == list.ActiveConfigID {
				table.Append([]string{"\u2713", list.Configs[val].Label, list.Configs[val].User, list.Configs[val].Url, list.Configs[val].AuthType})
			} else {
				table.Append([]string{"", list.Configs[val].Label, list.Configs[val].User, list.Configs[val].Url, list.Configs[val].AuthType})
			}
		}
		table.Render()

		return nil
	},
}
var configUseCmd = &cobra.Command{
	Use:   "use",
	Short: "Define as active, one of the current authentication profiles",
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
	Short: "Remove a profile from the cells-client authentication profiles",
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

		var removed string
		var index int
		var active string

		if len(items) == 1 {
			return ClearConfig()
		} else if len(items) > 1 {

			pSelect := promptui.Select{Label: "Select a configuration to remove", Items: items, Size: len(items)}
			index, removed, err = pSelect.Run()
			if err != nil {
				return err
			}
			items = append(items[:index], items[index+1:]...)

			// TODO also remove the key from the keyring
			if !cl.Configs[removed].SkipKeyring {
				err = rest.ClearKeyring(cl.Configs[removed])
				if err != nil {
					return fmt.Errorf("could not clear keyring for %s: %s \n ==> Aborting... ", removed, err.Error())
				}
			}

			if removed != cl.ActiveConfigID && len(items) > 1 {
				pSelect2 := promptui.Select{Label: "Please select the new active configuration", Items: items, Size: len(items)}
				_, active, err = pSelect2.Run()
				if err != nil {
					return err
				}
			} else if len(items) == 1 {
				active = items[0]
			}

		} else {
			return fmt.Errorf("configuration list is empty")
		}

		if !cl.Configs[removed].SkipKeyring {
			err = rest.ClearKeyring(cl.Configs[removed])
			if err != nil {
				return fmt.Errorf("could not clear keyring for %s: %s \n ==> Aborting... ", removed, err.Error())
			}
		}

		if err := cl.Remove(removed); err != nil {
			return err
		}

		if active != "" {
			if err := cl.SetActiveConfig(active); err != nil {
				return err
			}
		}

		if err := cl.SaveConfigFile(); err != nil {
			return err
		}

		cmd.Printf("Removed the following configuration %s\n\n", removed)
		if active != "" {
			cmd.Printf("The new active configuration is: %s\n", cl.ActiveConfigID)
		}

		return nil
	},
}

var checkKeyringCmd = &cobra.Command{
	Use:   "check-keyring",
	Short: "Try to store and retrieve a dummy value in the local keyring to test it",
	Long: `
DESCRIPTION

  Helper command to check if a keyring is present and correctly configured 
  in the client machine by simply storing and retrieving a dummy password.
`,
	Run: func(cm *cobra.Command, args []string) {

		if err := rest.CheckKeyring(); err != nil {
			fmt.Println(promptui.IconWarn + " " + rest.NoKeyringMsg)
			os.Exit(1)
		} else {
			fmt.Println(promptui.IconGood + " Keyring seems to be here and working.")
		}
	},
}

func init() {
	configCmd.AddCommand(configUseCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configRemoveCmd)
	configCmd.AddCommand(checkKeyringCmd)
	RootCmd.AddCommand(configCmd)
}
