package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Displays current config",
	Long: `
Displays the current active config, show the users and the cells instance
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("You are currently logged on:", rest.DefaultConfig.Url)
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
}
