// Package cmd implements some basic examples of what can be achieved when combining
// the use of the Go SDK for Cells with the powerful Cobra framework to implement CLI
// client applications for Cells.
package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
)

var (
	configFile string
)

const (
	bash_completion_func = `__./cec_custom_func() {
		case ${last_command} in
		./cec_mv | ./cec_cp | ./cec_rm)
			_totoctl
			return
			;;
		*) ;;
	
		esac
	}
	_totoctl() {
		local lsopts cur dir
		cur="${COMP_WORDS[COMP_CWORD]}"
	
		dir=$(dirname "$cur" 2>/dev/null)
	
		curlen=${#cur}
		last_char=${cur:curlen-1:1}
	
		if [[ $last_char == "/" ]] && [[ curlen -gt 2 ]]; then
			dir=$cur
		elif [[ -z $dir ]]; then
			dir="/"
		elif [[ $dir == "." ]]; then
			dir="/"
		fi
	
		# set +x
	
		lsopts=$(./cec ls --raw $dir)
		COMPREPLY=($(compgen -W "${lsopts[*]}" -- "$cur"))
		compopt -o nospace
	}
	`
)

// RootCmd is the parent of all example commands defined in this package.
// It takes care of the pre-configuration of the defaut connection to the SDK
// in its PersistentPreRun phase.
var RootCmd = &cobra.Command{
	Use:                    os.Args[0],
	Short:                  "Connect to a Cells Server using the command line",
	BashCompletionFunction: bash_completion_func,
	Long: `
# This tool uses the REST API to connect a Cells Server.

Pydio Cells comes with a powerful REST API that exposes various endpoints and enable management of a running Cells instance.
As a convenience, the Pydio team also provide a ready to use SDK for the Go language that encapsulates the boiling code to wire things 
and provides a few chosen utilitary methods to ease implemantation when using the SDK in various Go programs.

The children commands defined here show some basic examples of what can be achieved when combining the use of this SDK with 
the powerful Cobra framework to easily implement small CLI client applications.
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		if cmd.Use != "configure" && cmd.Use != "oauth" && cmd.Use != "clear" && cmd.Use != "doc" {
			e := rest.SetUpEnvironment(configFile)
			if e != nil {
				log.Fatalf("cannot read config file, please make sure to run %s configure first (error %s)", os.Args[0], e)
			}
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	flags := RootCmd.PersistentFlags()
	flags.StringVarP(&configFile, "config", "c", "", "Path to the configuration file")

}
