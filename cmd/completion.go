package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var shType string

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Add auto-completion helper to Cells Client",
	Long: `
Install completion helper to Pydio Cells Client.

This command installs an additional plugin to provide suggestions when working with the Cells Client and hitting the 'tab' key.

1) Using Bash

	On Linux, you must insure the 'bash-completion' library is installed:
		Debian / Ubuntu
			sudo apt install bash-completion
		
		RHEL / CentOS
			sudo yum install bash-completion
		
		On MacOS (be sure to follow the instructions displayed on Homebrew)
			brew install bash-completion

Then, to enable completion in your current session:
	
	source <(` + os.Args[0] + ` completion bash)

Or in a persistent manner:

	Debian/Ubuntu/CentOS
		` + os.Args[0] + ` completion bash | sudo tee /etc/bash_completion.d/cec

	MacOS
		` + os.Args[0] + ` completion bash | tee /usr/local/etc/bash_completion.d/cec

2) Using Zsh

	Add to current zsh session:
		source <(` + os.Args[0] + ` completion zsh)

	Add in a persistent manner:
		` + os.Args[0] + ` completion zsh | sudo tee <path>/<to>/<your zsh completion folder>
	
	On macOS
		` + os.Args[0] + ` completion zsh | tee /Users/<your current user>/.zsh/completion/_cec
	 `,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	ValidArgs: []string{"zsh", "bash"},
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

func init() {
	RootCmd.AddCommand(completionCmd)
	completionCmd.AddCommand(bashCompletionCmd)
	completionCmd.AddCommand(zshCompletionCmd)

}

// Reads the bash autocomplete file and prints it to stdout
func bashAutocomplete() {
	RootCmd.GenBashCompletion(os.Stdout)
}

func zshAutocomplete() {
	RootCmd.GenZshCompletion(os.Stdout)
}
