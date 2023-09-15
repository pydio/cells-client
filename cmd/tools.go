package cmd

import "github.com/spf13/cobra"

// toolsCmd are tools that do not need a valid connection to a remote running Cells instance
var toolsCmd = &cobra.Command{
	Use:    "tools",
	Short:  "Additional tools",
	Hidden: true,
	Long: `
DESCRIPTION

  Various commands that do not require a running Cells instance.
`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	RootCmd.AddCommand(toolsCmd)
}
