package cmd

import (
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Displays the active user, server and authentication type.",
	Run: func(cmd *cobra.Command, args []string) {

		dc := rest.DefaultConfig

		t := tablewriter.NewWriter(cmd.OutOrStdout())
		t.SetHeader([]string{"Username", "URL", "Type"})
		t.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		t.Append([]string{dc.User, dc.Url, dc.AuthType})
		t.Render()
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
}
