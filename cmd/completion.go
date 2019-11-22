package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var shType string

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Auto completion for Pydio Cells Client",
	Long: `Completion for Pydio Cells Client
	 
	 # Add to current session
	 source <(cec completion bash)
	 # Add to current zsh session
	 source <(cec completion zsh)
	 
	 # Add bashcompletion file (might require sudo)
	 cec completion bash > /etc/bash_completion.d/cec
	 # Add zshcompletion file
	 cec	completion zsh > ~/.zsh/completion/_cec
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
