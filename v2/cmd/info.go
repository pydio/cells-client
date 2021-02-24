package cmd

import (
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Displays the active user, server and authentication type.",
	Run: func(cmd *cobra.Command, args []string) {

		dc := rest.DefaultConfig

		login, _ := rest.RetrieveCurrentSessionLogin()
		// if err != nil {
		// 	log.Fatal(err)
		// }
		t := tablewriter.NewWriter(cmd.OutOrStdout())
		t.SetHeader([]string{"username", "url", "type"})
		t.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		t.Append([]string{login, dc.Url, dc.AuthType})
		t.Render()
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
}

func loginFromConfig(cfg *rest.CecConfig) (login string) {
	switch cfg.AuthType {
	case common.PersonalTokenType, common.OAuthType:
		login, _ = rest.RetrieveCurrentSessionLogin()
	case common.ClientAuthType:
		return cfg.User
	}
	return ""
}
