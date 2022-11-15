package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"text/template"
	"time"

	hashivers "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/common"
)

var cellsVersionTpl = `{{.PackageLabel}}
 Version: 	{{.Version}}
 Built: 	{{.BuildTime}}
 Git commit: 	{{.GitCommit}}
 OS/Arch: 	{{.OS}}/{{.Arch}}
 Go version: 	{{.GoVersion}}
`

var versionQuiet bool

type cecVersion struct {
	PackageLabel string
	Version      string
	BuildTime    string
	GitCommit    string
	OS           string
	Arch         string
	GoVersion    string
}

var (
	format string
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show Cells Client version information",
	Long: `
DESCRIPTION

  Print version information.

  You can format the output with a go template using the --format flag.
  Typically, to only output a parsable version, call:

    $ ` + os.Args[0] + ` version -f '{{.Version}}'
 
  As reference, known attributes are:
   - PackageLabel
   - Version
   - BuildTime
   - GitCommit
   - OS
   - Arch
   - GoVersion

  This also provides various utility sub-commands that come handy when manipulating software files. 
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

		cv := &cecVersion{
			PackageLabel: common.PackageLabel,
			Version:      sV,
			BuildTime:    t.Format(time.RFC822Z),
			GitCommit:    common.BuildRevision,
			OS:           runtime.GOOS,
			Arch:         runtime.GOARCH,
			GoVersion:    runtime.Version(),
		}

		var runningTmpl string

		if format != "" {
			runningTmpl = format
		} else {
			// Default version template
			runningTmpl = cellsVersionTpl
		}

		tmpl, err := template.New("cells").Parse(runningTmpl)
		if err != nil {
			log.Fatalln("failed to parse template", err)
		}

		if err = tmpl.Execute(os.Stdout, cv); err != nil {
			log.Fatalln("could not execute template", err)
		}
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
   ` + os.Args[0] + ` version isvalid 2.0.6-dev.20191205

  A *non* valid version:
   ` + os.Args[0] + ` version isvalid 2.a
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
				cm.Printf("1")
			} else {
				cm.Printf("0")
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
   ` + os.Args[0] + ` version isrelease 2.0.6

  A *non* release version:
   ` + os.Args[0] + ` version isrelease 2.0.6-dev.20191205
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
			errMessage = fmt.Sprintf("[%s] is *not* a valid version\n", versionStr)
		}

		if v.Prerelease() != "" {
			resultOK = false
			errMessage = fmt.Sprintf("[%s] is *not* a valid release version\n", versionStr)
		}

		if versionQuiet {
			if resultOK {
				cm.Printf("1")
			} else {
				cm.Printf("0")
			}
			os.Exit(0)
		} else {
			if !resultOK {
				cm.Println(errMessage)
				os.Exit(1)
			}
			// Valid release version and not quiet mode, nothing to do.
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
   ` + os.Args[0] + ` version isgreater 2.0.6-dev.20191205 2.0.6

  This returns 0 - false (and exits with status 0):
   ` + os.Args[0] + ` version isgreater --quiet 4.0.5-rc2 4.0.5

`,
	Run: func(cm *cobra.Command, args []string) {
		if len(args) != 2 {
			cm.Printf("Please provide two versions to be compared\n")
			os.Exit(1)
		}

		v1Str := args[0]
		v2Str := args[1]

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

	ivCmd.Flags().BoolVarP(&versionQuiet, "quiet", "q", false, "Simply returns 1 (true) or 0 (false) if the version is valid or not, without writing to standard error stream")
	irCmd.Flags().BoolVarP(&versionQuiet, "quiet", "q", false, "Simply returns 1 (true) or 0 (false) if the version represents valid release or not, without writing to standard error stream")
	igtCmd.Flags().BoolVarP(&versionQuiet, "quiet", "q", false, "Simply returns 1 (true) or 0 (false) if first passed version is greater than the second, without writing to standard error stream")
	versionCmd.Flags().StringVarP(&format, "format", "f", "", "Use go template to format version output")

	versionCmd.AddCommand(ivCmd)
	versionCmd.AddCommand(irCmd)
	versionCmd.AddCommand(igtCmd)

	RootCmd.AddCommand(versionCmd)

}
