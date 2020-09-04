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

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

var (
	updateToVersion string
	updateDryRun    bool
	devChannel      bool
	unstableChannel bool
	defaultChannel  = common.UpdateStableChannel
)

var updateBinCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for available updates and apply them",
	Long: `Without argument, this command will list the available updates for this binary.
To apply the actual update, re-run the command with a --version parameter.
`,
	Run: func(cmd *cobra.Command, args []string) {

		// if set it will use the selected channel to list and perform the update
		if devChannel {
			defaultChannel = common.UpdateDevChannel
		}

		binaries, e := rest.LoadUpdates(context.Background(), defaultChannel)
		if e != nil {
			log.Fatal("Cannot retrieve available updates", e)
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

			// fmt.Println("Updating binary now")
			fmt.Printf("Starting upgrade\n\n")
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
						fmt.Printf("\nChecking downloaded file integrity\n")
						fmt.Println("Replacing binary now")
						fmt.Printf("\n\nCells Client has been upgraded to version %v\n", apply.Version)
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
	updateBinCmd.Flags().BoolVarP(&devChannel, "dev", "", false, "If set this flag will use the dev channel to load the updates")
	updateBinCmd.Flags().BoolVarP(&unstableChannel, "unstable", "", false, "If set this flag will use the unstable channel to load the updates")

}
