package cmd

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/pydio/cells-client/rest"
)

var updateToVersion string
var updateDryRun bool

var updateBinCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for available updates and apply them",
	Long: `Without argument, this command will list the available updates for this binary.
To apply the actual update, re-run the command with a --version parameter.
`,
	Run: func(cmd *cobra.Command, args []string) {

		binaries, e := rest.LoadUpdates(context.Background())
		if e != nil {
			log.Fatal("Cannot retrieve available updates", zap.Error(e))
		}
		if len(binaries) == 0 {
			c := color.New(color.FgRed)
			c.Println("\nNo updates are available for this version")
			c.Println("")
			return
		}

		if updateToVersion == "" {
			// List versions
			c := color.New(color.FgGreen)
			c.Println("\nNew packages are available. Please run the following command to upgrade to a given version")
			c.Println("")
			c = color.New(color.FgBlack, color.Bold)
			c.Println(os.Args[0] + " update --version=x.y.z")
			c.Println("")

			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader([]string{"Version", "UpdatePackage Name", "Description"})

			for _, bin := range binaries {
				table.Append([]string{bin.Version, bin.Label, bin.Description})
			}

			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.Render()

		} else {

			var apply *rest.UpdatePackage
			for _, binary := range binaries {
				if binary.Version == updateToVersion {
					apply = binary
				}
			}
			if apply == nil {
				log.Fatal("Cannot find the requested version")
			}

			c := color.New(color.FgBlack)
			c.Println("Updating binary now")
			c.Println("")
			pgChan := make(chan float64)
			errorChan := make(chan error)
			doneChan := make(chan bool)
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case pg := <-pgChan:
						fmt.Printf("\rDownloading binary: %v%%", math.Floor(pg*100))
					case e := <-errorChan:
						fmt.Printf("\rError while updating binary: " + e.Error())
						return
					case <-doneChan:
						// TODO use another color or let the default color
						fmt.Printf("\n\nCells Client binary successfully upgraded\n")
						return
					}
				}
			}()
			rest.ApplyUpdate(context.Background(), apply, updateDryRun, pgChan, doneChan, errorChan)
			wg.Wait()
		}

	},
}

func init() {

	RootCmd.AddCommand(updateBinCmd)

	updateBinCmd.Flags().StringVarP(&updateToVersion, "version", "v", "", "Pass a version number to apply the upgrade")
	updateBinCmd.Flags().BoolVarP(&updateDryRun, "dry-run", "d", false, "If set, this flag will grab the package and save it to the tmp directory instead of replacing current binary")

}
