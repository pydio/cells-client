package cmd

import (
	"github.com/spf13/cobra"
)

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Manage existing Cells Flow Jobs",
	Long: `
DESCRIPTION

  Commands to manage jobs, mainly search, list and delete.
  See the help of respective sub-commands for further details.
	`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cm *cobra.Command, args []string) {
		_ = cm.Usage()
	},
}

func init() {
	RootCmd.AddCommand(jobsCmd)
}
