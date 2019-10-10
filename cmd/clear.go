package cmd

import (
	"fmt"
	"os"

	"github.com/micro/go-log"
	"github.com/pydio/cells-client/rest"
	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:  "clear",
	Long: "Clear current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		filePath := rest.DefaultConfigFilePath()
		if err := os.Remove(filePath); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Successfully removed config file")
	},
}

func init() {
	RootCmd.AddCommand(clearCmd)
}
