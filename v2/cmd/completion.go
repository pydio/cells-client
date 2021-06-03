package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func bashCompletionExample(bin string) string {
	return `Using Bash

	# Add to current bash session:
		source <(` + bin + ` completion bash)

	# Debian/Ubuntu/CentOS
		` + bin + ` completion bash | sudo tee /etc/bash_completion.d/cec

	# macOS
		` + bin + ` completion bash | tee /usr/local/etc/bash_completion.d/cec

#### You must insure the 'bash-completion' library is installed:
	
	# Debian / Ubuntu
		sudo apt install bash-completion
	
	# RHEL / CentOS
		sudo yum install bash-completion
	
	# On MacOS (after the installation make sure to follow the instructions displayed by Homebrew)
		brew install bash-completion
`
}

func zshCompletionExample(bin string) string {
	return `Zsh

	# Add to current zsh session:
	source <(` + bin + ` completion zsh)

	# Debian/Ubuntu/CentOS:
	` + bin + ` completion zsh | sudo tee <path>/<to>/<your zsh completion folder>

	# macOS
	` + bin + ` completion zsh | tee ${fpath[1]}/_cec`
}

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Add auto-completion helper to Cells Client",
	Long: `
DESCRIPTION

  Install a completion helper to the Cells Client.

For the installation manuals, run the respective helpers:

	Bash
		` + os.Args[0] + " completion " + "bash --help" + `

	Zsh
		` + os.Args[0] + " completion " + "zsh --help" + `

  This command configures an additional plugin to provide suggestions when hitting the 'tab' key.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Help()
		if err != nil {
			cmd.PrintErr(err)
			os.Exit(1)
		}
	},
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"zsh", "bash"},
}

var bashCompletionCmd = &cobra.Command{
	Use:     "bash",
	Example: bashCompletionExample(os.Args[0]),
	Run: func(cmd *cobra.Command, args []string) {
		err := RootCmd.GenBashCompletion(os.Stdout)
		if err != nil {
			return
		}
	},
}

var zshCompletionCmd = &cobra.Command{
	Use:     "zsh",
	Example: zshCompletionExample(os.Args[0]),
	Run: func(cmd *cobra.Command, args []string) {
		err := RootCmd.GenZshCompletionNoDesc(os.Stdout)
		if err != nil {
			return
		}
	},
}

func init() {
	RootCmd.AddCommand(completionCmd)
	completionCmd.AddCommand(bashCompletionCmd)
	completionCmd.AddCommand(zshCompletionCmd)

}
