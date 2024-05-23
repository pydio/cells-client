package cmd

import "github.com/spf13/cobra"

// ToolsCmd are tools that do not need a valid connection to a remote running Cells instance
var ToolsCmd = &cobra.Command{
	Use:    "tools",
	Short:  "Additional tools",
	Hidden: true,
	Long: `
DESCRIPTION

  Various additional useful helper commands.
`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Usage()
	},
}

func init() {
	RootCmd.AddCommand(ToolsCmd)
}
