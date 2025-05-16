package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmdExample = `1) Using Bash

	# Add to current bash session:
		source <(` + os.Args[0] + ` completion bash)

	# Debian/Ubuntu/CentOS
		` + os.Args[0] + ` completion bash | sudo tee /etc/bash_completion.d/cec

	# macOS
		` + os.Args[0] + ` completion bash | tee /usr/local/etc/bash_completion.d/cec

2) Zsh

	# Add to current zsh session:
		source <(` + os.Args[0] + ` completion zsh)

	# Debian/Ubuntu/CentOS:
		` + os.Args[0] + ` completion zsh | sudo tee <path>/<to>/<your zsh completion folder>

	# macOS
		` + os.Args[0] + ` completion zsh | tee /Users/<your current user>/.zsh/completion/_cec


#### You must insure the 'bash-completion' library is installed:
	
	# Debian / Ubuntu
		sudo apt install bash-completion
	
	# RHEL / CentOS
		sudo yum install bash-completion
	
	# On MacOS (after the installation make sure to follow the instructions displayed by Homebrew)
		brew install bash-completion
`

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Add auto-completion helper to Cells Client",
	Long: `
DESCRIPTION

  Install a completion helper to the Cells Client.

  This command configures an additional plugin to provide suggestions when hitting the 'tab' key.`,
	Example: completionCmdExample,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	ValidArgs: []string{"zsh", "bash"},
}

func init() {
	RootCmd.AddCommand(completionCmd)
	completionCmd.AddCommand(bashCompletionCmd)
	completionCmd.AddCommand(zshCompletionCmd)
}

var bashCompletionCmd = &cobra.Command{
	Use: "bash",
	Run: func(cmd *cobra.Command, args []string) {
		bashAutocomplete()
	},
}

var zshCompletionCmd = &cobra.Command{
	Use: "zsh",
	Run: func(cmd *cobra.Command, args []string) {
		zshAutocomplete()
	},
}

// Reads the bash autocomplete file and prints it to stdout
func bashAutocomplete() {
	RootCmd.GenBashCompletion(os.Stdout)
}

func zshAutocomplete() {
	RootCmd.GenZshCompletion(os.Stdout)
}

var bashCompletionFunc = `__` + os.Args[0] + `_custom_func() {
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
