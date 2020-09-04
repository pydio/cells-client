package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Displays current config",
	Long: `
Displays the current active config, show the users and the cells instance
`,
	Run: func(cmd *cobra.Command, args []string) {

		t := tabwriter.NewWriter(os.Stdout, 5, 0, 3, ' ', tabwriter.TabIndent)
		fmt.Fprintln(t, "URL\tUser")
		fmt.Fprintf(t, "%v\t%v", rest.DefaultConfig.Url, rest.DefaultConfig.User)
		fmt.Fprintln(t)
		t.Flush()
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)
}
