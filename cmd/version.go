package cmd

import (
	"fmt"
	"os"
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

var verCmd = &cobra.Command{
	Use:   "version",
	Short: "Version related commands",
	Run: func(cm *cobra.Command, args []string) {
		cm.Usage()
	},
}

// showCmd displays information about the current version of this software.
var showCmd = &cobra.Command{
	Use:   "show",
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

var ivCmd = &cobra.Command{
	Use:   "isvalid",
	Short: "Return an error if the passed version is not correctly formatted",
	Long:  `Tries to parse passed version as string using the hashicorp/go-version library`,
	Run: func(cm *cobra.Command, args []string) {
		if len(args) != 1 {
			cm.Printf("Please provide a version to parse\n")
			os.Exit(1)
		}

		versionStr := args[0]
		fmt.Printf("Checking version %s \n", versionStr)

		_, err := hashivers.NewVersion(versionStr)
		if err != nil {
			cm.Printf("Passed version [%s] is not a valid version\n", versionStr)
			os.Exit(1)
		}
	},
}

var igtCmd = &cobra.Command{
	Use:   "isgreater",
	Short: "Compares the two passed versions and returns true if the first is strictly greater than the second",
	Long: `Tries to parse passed versions as string using the hashicorp/go-version library and returns an error if:
	- one of the 2 strings cannot be parsed
	- the first version is not strictly greater than the second`,
	Run: func(cm *cobra.Command, args []string) {
		if len(args) != 2 {
			cm.Printf("Please provide two versions to be compared\n")
			os.Exit(1)
		}

		v1Str := args[0]
		v2Str := args[1]
		fmt.Printf("Comparing versions %s & %s \n", v1Str, v2Str)

		v1, err := hashivers.NewVersion(v1Str)
		if err != nil {
			cm.Printf("Passed version [%s] is not a valid version\n", v1Str)
			os.Exit(1)
		}
		v2, err := hashivers.NewVersion(v2Str)
		if err != nil {
			cm.Printf("Passed version [%s] is not a valid version\n", v2Str)
			os.Exit(1)
		}
		if !v1.GreaterThan(v2) {
			cm.Printf("Passed version [%s] is *not* greater than [%s]\n", v1Str, v2Str)
			os.Exit(1)
		}
	},
}

func init() {
	verCmd.AddCommand(showCmd)
	verCmd.AddCommand(ivCmd)
	verCmd.AddCommand(igtCmd)
	RootCmd.AddCommand(verCmd)
}
