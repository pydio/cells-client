package cmd

import (
	"log"

	"github.com/pydio/go/docs"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v4/common"
)

var docPath string

var docCmd = &cobra.Command{
	Use:   "doc",
	Short: "Generate documentation of the Cells Client",
	Long: `
DESCRIPTION
  This command automatically generates the documentation of the Cells Client.
  It produces nice Markdown files based on the various comments that are in the code itself.

  Please, provide the '-p' flag with a path to define where to put the generated files.

  Also note that this command also generates the yaml files that we use for pydio.com documentation format.
`,
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {

		if docPath == "" {
			log.Fatal("Please provide a path to store output files")
		} else {

			docs.PydioDocsGeneratedBy = common.PackageLabel + " v" + common.Version
			err := docs.GenMarkdownTree(RootCmd, docPath)
			if err != nil {
				log.Fatal(err)
			}
		}

	},
}

func init() {
	docCmd.Flags().StringVarP(&docPath, "path", "p", "", "Target folder where to put the files")
	docCmd.Flags().StringVarP(&docs.PydioDocsMenuName, "menu", "m", "menu-admin-guide-v7", "Pydio Docs menu name")
	RootCmd.AddCommand(docCmd)
}
