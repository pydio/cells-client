// Package cmd implements some basic examples of what can be achieved when combining
// the use of the Go SDK for Cells with the powerful Cobra framework to implement CLI
// client applications for Cells.
package cmd

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
)

var configFile string

// RootCmd is the parent of all commands defined in this package.
// It takes care of the pre-configuration of the defaut connection to the SDK in its PersistentPreRun phase.
var RootCmd = &cobra.Command{
	Use:                    os.Args[0],
	Short:                  "Connect to a Pydio Cells server using the command line",
	BashCompletionFunction: bash_completion_func,
	Long: `
# Pydio Cells Client

This command line client allows interacting with a Pydio Cells server instance directly via the command line. 
It uses the Cells SDK for Go and the REST API under the hood.

See the respective help pages of the various commands to get detailed explanation and some examples.

You should probably start with configuring your setup by running:
 ` + os.Args[0] + ` configure

This will guide you through a quick procedure to get you up and ready in no time.
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		parts := strings.Split(cmd.CommandPath(), " ")
		rc := ""
		if len(parts) > 1 {
			rc = parts[1]
		}
		switch rc {
		// These command and respective children do not need an already configured environment
		case "", "configure", "version", "completion", "oauth", "clear", "doc":
			break
		default:
			e := rest.SetUpEnvironment(configFile)
			if e != nil {
				log.Fatalf("cannot read config file, please make sure to run '%s configure' first. (Error: %s)\n", os.Args[0], e.Error())
			}
		}

	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var bash_completion_func = `__` + os.Args[0] + `_custom_func() {
  case ${last_command} in
  ` + os.Args[0] + `_mv | ` + os.Args[0] + `_cp | ` + os.Args[0] + `_rm | ` + os.Args[0] + `_ls)
    _path_completion
    return
    ;;
	` + os.Args[0] + `_storage_resync-ds)
    _datasources_completion
    return
    ;;
  ` + os.Args[0] + `_scp)
    _scp_path_completion
    return
    ;;
  *) ;;
  esac
}
_path_completion() {
  local lsopts cur dir
  cur="${COMP_WORDS[COMP_CWORD]}"
  dir="$(dirname "$cur" 2>/dev/null)"

  currentlength=${#cur}
  last_char=${cur:currentlength-1:1}

  if [[ $last_char == "/" ]] && [[ currentlength -gt 2 ]]; then
    dir=$cur
  elif [[ -z $dir ]]; then
    dir="/"
  elif [[ $dir == "." ]]; then
    dir="/"
  fi

  IFS=$'\n'
  lsopts="$(` + os.Args[0] + ` ls --raw $dir)"

  COMPREPLY=($(compgen -W "${lsopts[@]}" -- "$cur"))
  compopt -o nospace
  compopt -o filenames
}

_scp_path_completion() {
  local lsopts cur dir
  cur="${COMP_WORDS[COMP_CWORD]}"
	
  if [[ $cur != cells//* ]]; then
    return
  fi

  prefix="cells//"
  cur=${cur#$prefix}

  dir="$(dirname "$cur" 2>/dev/null)"
  currentlength=${#cur}
  last_char=${cur:currentlength-1:1}

  if [[ $last_char == "/" ]] && [[ currentlength -gt 2 ]]; then
      dir=$cur
  elif [[ -z $dir ]]; then
      dir="/"
  elif [[ $dir == "." ]]; then
      dir="/"
  fi

  IFS=$'\n'
  lsopts="$(` + os.Args[0] + ` ls --raw $dir)"

  COMPREPLY=($(compgen -P "$prefix" -W "${lsopts[@]}" -- "$cur"))
  #COMPREPLY=(${COMPREPLY[@]/#/"${prefix}"})
  compopt -o nospace
  compopt -o filenames
}

_datasources_completion() {
  local dsopts cur
  cur="${COMP_WORDS[COMP_CWORD]}"

  dsopts="$(` + os.Args[0] + ` storage list-datasources --raw)"
  COMPREPLY=($(compgen -W "${dsopts[@]}" -- "$cur"))
}
`

func init() {
	flags := RootCmd.PersistentFlags()
	flags.StringVarP(&configFile, "config", "c", "", "Path to the configuration file")
}
