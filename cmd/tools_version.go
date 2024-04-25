package cmd

import (
	"fmt"
	"os"

	hashivers "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
)

var versionQuiet bool

// tvCmd are helpers to manipulate software versions.
var tvCmd = &cobra.Command{
	Use:   "version",
	Short: "Version helpers",
	Long: `
DESCRIPTION

  Various commands to manipulate, verify and compare software versions.
`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var ivCmd = &cobra.Command{
	Use:   "isvalid",
	Short: "Check if a given string represents a valid version",
	Long: `
DESCRIPTION 
  
  Check the passed string to validate that it respects semantic versioning rules.
  It returns an error if the string is not correctly formatted.

  Under the hood, we try to parse the version string using the hashicorp/go-version library.
   - If the passed version is *not* valid, the process exits with status 1.
   - When it is valid, the process simply exits with status 0.

   If the "quiet" mode is enabled, the command simply returns 1 (true) or 0 (false), depending
   on the passed argument, without writing to error out.

EXAMPLES

  A valid version:
   ` + os.Args[0] + ` tools version isvalid 2.0.6-dev.20191205

  A *non* valid version:
   ` + os.Args[0] + ` tools version isvalid 2.a
`,
	Run: func(cm *cobra.Command, args []string) {
		if len(args) != 1 {
			cm.Printf("Please provide a version to parse\n")
			os.Exit(1)
		}
		versionStr := args[0]
		_, err := hashivers.NewVersion(versionStr)
		if versionQuiet {
			if err == nil {
				cm.Println("1")
			} else {
				cm.Println("0")
			}
			os.Exit(0)
		} else {
			if err != nil {
				cm.Printf("[%s] is *not* a valid version\n", versionStr)
				os.Exit(1)
			}
		}
	},
}

var irCmd = &cobra.Command{
	Use:   "isrelease",
	Short: "Check if a given string represents a valid **RELEASE** version",
	Long: `
DESCRIPTION

  Check the passed string to validate that it respects semantic versioning rules
  *and* represents a valid release version.
  It returns an error if the string is not correctly formatted or represents a SNAPSHOT.
  
  Under the hood, we try to parse the version string using the hashicorp/go-version library.
  We then check that the passed string is not a pre-release, that is that is not suffixed 
  by "a hyphen and a series of dot separated identifiers immediately following the patch version", 
  see: https://semver.org
  
  In case the passed version is *not* a valid release version, the process prints an error 
  and exits with status 1. Otherwise it simply exits silently with status 0.

  If the "quiet" mode is enabled, the command simply returns 1 (true) or 0 (false), depending
  on the passed argument, without writing to error out.

EXAMPLES

  A valid release version:
   ` + os.Args[0] + ` tools version isrelease 2.0.6

  A *non* release version:
   ` + os.Args[0] + ` tools version isrelease 2.0.6-dev.20191205
`,
	Run: func(cm *cobra.Command, args []string) {
		if len(args) != 1 {
			cm.Printf("Please provide a single version to be parsed\n")
			os.Exit(1)
		}
		versionStr := args[0]

		resultOK := true
		errMessage := ""

		v, err := hashivers.NewVersion(versionStr)
		if err != nil {
			resultOK = false
			errMessage = fmt.Sprintf("[%s] is *not* a valid version", versionStr)
		} else if v.Prerelease() != "" {
			resultOK = false
			errMessage = fmt.Sprintf("[%s] is *not* a valid release version", versionStr)
		}

		if versionQuiet {
			if resultOK {
				cm.Println("1")
			} else {
				cm.Println("0")
			}
			os.Exit(0)
		} else {
			if !resultOK {
				cm.Println(errMessage)
				os.Exit(1)
			}
			// Valid release version and not in quiet mode, we simply do nothing.
		}
	},
}

var igtCmd = &cobra.Command{
	Use:   "isgreater",
	Short: "Compare the two versions, succeed when the first is greater than the second",
	Long: `
DESCRIPTION

  Check the passed strings to validate that they respects semantic versioning rules
  and then compare them.
  
  Under the hood, it tries to parse the version strings using the hashicorp/go-version library
  and then compare the 2 resulting version structs.
  
  The command prints an error and exits with status 1 if:
    - one of the 2 strings is not a valid semantic version,
    - the first version is not strictly greater than the second.

  Otherwise, the command simply exits with status 0.	

  If the "quiet" mode is enabled, the command simply returns 1 (true) or 0 (false), depending
  on the passed arguments, without writing to error out.

EXAMPLE

  This exits with status 1:
   ` + os.Args[0] + ` tools version isgreater 2.0.6-dev.20191205 2.0.6

  This returns 0 - false (and exits with status 0):
   ` + os.Args[0] + ` tools version isgreater --quiet 4.0.5-rc2 4.0.5

`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			cmd.Printf("Please provide two versions to be compared\n")
			os.Exit(1)
		}

		v1Str := args[0]
		v2Str := args[1]

		resultOK := true
		errMessage := ""

		v1, err := hashivers.NewVersion(v1Str)
		if err != nil {
			resultOK = false
			errMessage = fmt.Sprintf("Passed version [%s] is not a valid version\n", v1Str)
		}
		v2, err := hashivers.NewVersion(v2Str)
		if err != nil {
			resultOK = false
			errMessage = fmt.Sprintf("Passed version [%s] is not a valid version\n", v2Str)
		}
		if resultOK && !v1.GreaterThan(v2) {
			resultOK = false
			errMessage = fmt.Sprintf("Passed version [%s] is *not* greater than [%s]\n", v1Str, v2Str)
		}

		if versionQuiet {
			if resultOK {
				cmd.Println("1")
			} else {
				cmd.Println("0")
			}
			os.Exit(0)
		} else {
			if !resultOK {
				cmd.Println(errMessage)
				os.Exit(1)
			}
			// Valid and ordered release versions, nothing to do.
		}
	},
}

// hiddenIvCmd is a hidden shortcut to keep an alias to the pre v4 existing command.
var hiddenIvCmd = &cobra.Command{
	Use:    "isvalid",
	Hidden: true,
	Short:  "Check if a given string represents a valid version",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("[WARNING] this command is deprecated and will be removed in a future release.")
		cmd.Println("Rather use:", os.Args[0], " tools version isvalid 4.1.1-dev.20240425")
		ivCmd.Run(cmd, args)
	},
}

// hiddenIrCmd is a hidden shortcut to keep an alias to the pre v4 existing command.
var hiddenIrCmd = &cobra.Command{
	Use:    "isrelease",
	Hidden: true,
	Short:  "Check if a given string represents a valid **RELEASE** version",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("[WARNING] this command is deprecated and will be removed in a future release.")
		cmd.Println("Rather use:", os.Args[0], " tools version isrelease 4.1.1-dev.20240425")
		irCmd.Run(cmd, args)
	},
}

// hiddenIgtCmd is a hidden shortcut to keep an alias to the pre v4 existing command.
var hiddenIgtCmd = &cobra.Command{
	Use:    "isgreater",
	Hidden: true,
	Short:  "Compare the two versions, succeed when the first is greater than the second",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("[WARNING] this command is deprecated and will be removed in a future release.")
		cmd.Println("Rather use:", os.Args[0], " tools version isgreater 4.1.1 4.1.1-dev.20240425")
		igtCmd.Run(cmd, args)
	},
}

func init() {
	ivCmd.Flags().BoolVarP(&versionQuiet, "quiet", "q", false, "Simply returns 1 (true) or 0 (false) if the version is valid or not, without writing to standard error stream")
	irCmd.Flags().BoolVarP(&versionQuiet, "quiet", "q", false, "Simply returns 1 (true) or 0 (false) if the version represents valid release or not, without writing to standard error stream")
	igtCmd.Flags().BoolVarP(&versionQuiet, "quiet", "q", false, "Simply returns 1 (true) or 0 (false) if first passed version is greater than the second, without writing to standard error stream")
	tvCmd.AddCommand(ivCmd, irCmd, igtCmd)
	ToolsCmd.AddCommand(tvCmd)
}
