package cmd

import (
	"github.com/spf13/cobra"
)

var storageCmd = &cobra.Command{
	Use:  "storage",
	Long: `DataSources / Workspaces`,
	Run: func(cm *cobra.Command, args []string) {
		cm.Usage()
	},
}

func init() {
	RootCmd.AddCommand(storageCmd)
}
