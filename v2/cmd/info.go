package cmd

import (
	"os"

	"github.com/olekukonko/tablewriter"
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

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Login", "URL"})
		table.Append([]string{rest.DefaultConfig.User, rest.DefaultConfig.Url})
		table.Render()

	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
}
