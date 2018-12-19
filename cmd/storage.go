package cmd

import (
	"github.com/spf13/cobra"
)

var StorageCmd = &cobra.Command{
	Use:  "idm",
	Long: `Users / Groups / Roles commands`,
	Run: func(cm *cobra.Command, args []string) {
		cm.Usage()
	},
}

func init() {
	RootCmd.AddCommand(StorageCmd)
}
