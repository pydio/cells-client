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
	"fmt"
	"time"

	hashivers "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
)

var (
	PackageName   = "Cells Client"
	BuildStamp    string
	BuildRevision string
	version       string
)

// versionCmd represents the versioning command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display current version of this software",
	Run: func(cmd *cobra.Command, args []string) {

		var t time.Time
		if BuildStamp != "" {
			t, _ = time.Parse("2006-01-02T15:04:05", BuildStamp)
		} else {
			t = time.Now()
		}

		sV := "N/A"
		if v, e := hashivers.NewVersion(version); e == nil {
			sV = v.String()
		}

		fmt.Println("")
		fmt.Println("    " + fmt.Sprintf("%s (%s)", PackageName, sV))
		fmt.Println("    " + fmt.Sprintf("Published on %s", t.Format(time.RFC822Z)))
		fmt.Println("    " + fmt.Sprintf("Revision number %s", BuildRevision))
		fmt.Println("")

	},
}

func init() {

	RootCmd.AddCommand(versionCmd)

}
