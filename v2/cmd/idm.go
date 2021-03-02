package cmd

import (
	"github.com/spf13/cobra"
)

var idmCmd = &cobra.Command{
	Use:   "idm",
	Short: "Identity Management commands",
	Long: `
DESCRIPTION

  Commands to manage users, groups and roles. 
  See the help of respective sub-commands for further details.
`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cm *cobra.Command, args []string) {
		cm.Usage()
	},
}

func init() {
	RootCmd.AddCommand(idmCmd)
}
