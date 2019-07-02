package cmd

import (
	"fmt"
	"time"

	hashivers "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
)

var (
	PackageName   = "Cells Client"
	BuildStamp    string
	BuildRevision string
	version       string
)

// versionCmd represents the versioning command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display current version of this software",
	Run: func(cmd *cobra.Command, args []string) {

		var t time.Time
		if BuildStamp != "" {
			t, _ = time.Parse("2006-01-02T15:04:05", BuildStamp)
		} else {
			t = time.Now()
		}

		sV := "N/A"
		if v, e := hashivers.NewVersion(version); e == nil {
			sV = v.String()
		}

		fmt.Println("")
		fmt.Println("    " + fmt.Sprintf("%s (%s)", PackageName, sV))
		fmt.Println("    " + fmt.Sprintf("Published on %s", t.Format(time.RFC822Z)))
		fmt.Println("    " + fmt.Sprintf("Revision number %s", BuildRevision))
		fmt.Println("")

	},
}

func init() {

	RootCmd.AddCommand(versionCmd)

}
