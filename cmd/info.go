package cmd

import (
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/common"
	"github.com/pydio/cells-client/v4/rest"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display the active user, server and authentication type",
	Run: func(cmd *cobra.Command, args []string) {

		dc := rest.DefaultConfig

		t := tablewriter.NewWriter(cmd.OutOrStdout())
		t.SetHeader([]string{"Username", "Server URL", "Auth Type"})
		t.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		t.Append([]string{dc.User, dc.Url, common.GetAuthTypeLabel(dc.AuthType)})
		t.Render()
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
}
