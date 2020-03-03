package cmd

import (
	"fmt"
	"os"
	"runtime"
	"time"

	hashivers "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/common"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the current version of this application (and some utils)",
	Long: `
The version command simply shows the version that is currently running.

It also provides various utility sub commands than comes handy when manipulating software files. 
`,
	Run: func(cm *cobra.Command, args []string) {
		var t time.Time
		if common.BuildStamp != "" {
			t, _ = time.Parse("2006-01-02T15:04:05", common.BuildStamp)
		} else {
			t = time.Now()
		}

		sV := "N/A"
		if v, e := hashivers.NewVersion(common.Version); e == nil {
			sV = v.String()
		}

		fmt.Println("")
		fmt.Println("    " + fmt.Sprintf("%s (%s)", common.PackageName, sV))
		fmt.Println("    " + fmt.Sprintf("Published on %s", t.Format(time.RFC822Z)))
		fmt.Println("    " + fmt.Sprintf("Revision number %s", common.BuildRevision))
		fmt.Println("    " + fmt.Sprintf("OS %s ARCH %s", runtime.GOOS, runtime.GOARCH))
		fmt.Println("    " + fmt.Sprintf("GOVERSION %s", runtime.Version()))
	},
}

var ivCmd = &cobra.Command{
	Use:   "isvalid",
	Short: "Return an error if the passed version is not correctly formatted",
	Long: `Tries to parse the passed string version using the hashicorp/go-version library 
and hence validates that it respects semantic versionning rules.

In case the passed version is *not* valid, the process exits with status 1.`,
	Example: `
# A valid version
` + os.Args[0] + ` version isvalid 2.0.6-dev.20191205

# A *non* valid version
` + os.Args[0] + ` version isvalid 2.a
`,
	Run: func(cm *cobra.Command, args []string) {
		if len(args) != 1 {
			cm.Printf("Please provide a version to parse\n")
			os.Exit(1)
		}

		versionStr := args[0]

		_, err := hashivers.NewVersion(versionStr)
		if err != nil {
			cm.Printf("[%s] is *not* a valid version\n", versionStr)
			os.Exit(1)
			// do not output anything is case the version is correct.
			// } else {
			// 	cm.Printf("[%s] is a valid version\n", versionStr)
		}
	},
}

var irCmd = &cobra.Command{
	Use:   "isrelease",
	Short: "Return an error if the passed version is a snapshot",
	Long: `Tries to parse the passed string version using the hashicorp/go-version library 
and hence validates that it respects semantic versionning rules.

It then insures that the passed string is not a pre-release, 
that is that is not suffixed by "a hyphen and a series of dot separated identifiers 
immediately following the patch version", see: https://semver.org

In case the passed version is *not* a valid realease version, the process exits with status 1.`,
	Example: `
# A valid release version
` + os.Args[0] + ` version isvalid 2.0.6

# A *non* release version
` + os.Args[0] + ` version isvalid 2.0.6-dev.20191205
`,
	Run: func(cm *cobra.Command, args []string) {
		if len(args) != 1 {
			cm.Printf("Please provide a single version to be parsed\n")
			os.Exit(1)
		}

		versionStr := args[0]

		v, err := hashivers.NewVersion(versionStr)
		if err != nil {
			cm.Printf("[%s] is *not* a valid version\n", versionStr)
			os.Exit(1)
		}

		if v.Prerelease() != "" {
			// This is a pre-release, throw an error
			cm.Printf("[%s] is *not* a valid release version\n", versionStr)
			os.Exit(1)
		}
	},
}

var igtCmd = &cobra.Command{
	Use:   "isgreater",
	Short: "Compares the two passed versions and returns an error if the first is *not* strictly greater than the second",
	Long: `Tries to parse the passed string versions using the hashicorp/go-version library and returns an error if:
  - one of the 2 strings is not a valid semantic version,
  - the first version is not strictly greater than the second`,
	Example: `
# This exits with status 1.
` + os.Args[0] + ` version isgreater 2.0.6-dev.20191205 2.0.6
`,
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
	versionCmd.AddCommand(ivCmd)
	versionCmd.AddCommand(irCmd)
	versionCmd.AddCommand(igtCmd)
	RootCmd.AddCommand(versionCmd)
}
