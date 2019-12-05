/*
 * Copyright (c) 2018. Abstrium SAS <team (at) pydio.com>
 * This file is part of Pydio Cells.
 *
 * Pydio Cells is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Pydio Cells is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Pydio Cells.  If not, see <http://www.gnu.org/licenses/>.
 *
 * The latest code can be found at <https://pydio.com>.
 */

package cmd

import (
	"log"

	"github.com/pydio/go/docs"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/common"
)

var docPath string

var docCmd = &cobra.Command{
	Use:   "doc",
	Short: "Generate documentation of the Cells Client",
	Long: `
This command automatically generates the documentation of the Cells Client.
It produces nice Markdown files based on the various comments that are in the code itself.

Please, provide the '-p' flag with a path to define where to put the generated files.

Also note that this command also generates the yaml files that we use for pydio.com documentation format.
`,
	Run: func(cmd *cobra.Command, args []string) {

		if docPath == "" {
			log.Fatal("Please provide a path to store output files")
		} else {

			docs.PydioDocsGeneratedBy = common.PackageName + " v" + common.Version
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
