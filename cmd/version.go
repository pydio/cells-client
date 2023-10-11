package cmd

import (
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"text/template"

	hashivers "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/common"
)

var cellsVersionTpl = `{{.PackageLabel}}
 Version: 	{{.Version}}
 Git commit: 	{{.GitCommit}}
 Timestamp: 	{{.GitTime}}
 OS/Arch: 	{{.OS}}/{{.Arch}}
 Go version: 	{{.GoVersion}}
`

type cecVersion struct {
	PackageLabel string
	Version      string
	GitCommit    string
	GitTime      string
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
   - GoVersion
   - GitCommit
   - GitTime
   - OS
   - Arch
`,
	Run: func(cm *cobra.Command, args []string) {

		sV := "N/A"
		if v, e := hashivers.NewVersion(common.Version); e == nil {
			sV = v.String()
		}

		rev := ""
		ts := ""

		if info, ok := debug.ReadBuildInfo(); ok {
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					rev = s.Value
				case "vcs.time":
					ts = s.Value
				default:
				}
			}
		}

		cv := &cecVersion{
			PackageLabel: common.PackageLabel,
			Version:      sV,
			GitCommit:    rev,
			GitTime:      ts,
			GoVersion:    runtime.Version(),
			OS:           runtime.GOOS,
			Arch:         runtime.GOARCH,
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

func init() {

	// Add hidden deprecated commands to keep retro-compatibility
	versionCmd.AddCommand(hiddenIvCmd)
	versionCmd.AddCommand(hiddenIrCmd)
	versionCmd.AddCommand(hiddenIgtCmd)

	versionCmd.Flags().StringVarP(&format, "format", "f", "", "Use go template to format version output")
	RootCmd.AddCommand(versionCmd)
}
