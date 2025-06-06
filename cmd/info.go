package cmd

import (
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/common"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display the active connection's info: user, server and authentication type",
	Run: func(cmd *cobra.Command, args []string) {
		dc := sdkClient.GetConfig()
		t := tablewriter.NewTable(os.Stdout,
			tablewriter.WithConfig(tablewriter.Config{
				Row: tw.CellConfig{
					Formatting: tw.CellFormatting{AutoWrap: tw.WrapNone},
					Alignment:  tw.CellAlignment{Global: tw.AlignLeft},
				},
			}),
		)
		t.Header([]string{"Username", "Server URL", "Auth Type"})
		t.Append([]string{dc.User, dc.Url, common.GetAuthTypeLabel(dc.AuthType)})
		t.Render()
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
}
