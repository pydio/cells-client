package cmd

import (
	"github.com/spf13/cobra"
)

var metaCmd = &cobra.Command{
	Use: "meta",
	Short: "Metadata commands",
	Long: `
DESCRIPTION

 Commands to manage node's metadatas
 See the help of respective sub-commands for further details.
	`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cm *cobra.Command, args []string){
		_ = cm.Usage()
	},
}

func init() {
	RootCmd.AddCommand(metaCmd)
}