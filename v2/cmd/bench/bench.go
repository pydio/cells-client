package bench

import (
	"github.com/pydio/cells-client/v2/cmd"
	"github.com/spf13/cobra"
)

var (
	benchTimeout     int
	benchPoolSize    int
	benchMaxRequests int
	benchSkipCreate  bool
	benchSkipClean   bool
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Set of commands to benchmark a running instance",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	cmd.RootCmd.AddCommand(benchCmd)
	flags := benchCmd.PersistentFlags()
	flags.IntVar(&benchTimeout, "timeout", 2, "Timeout for HTTP requests (in minutes)")
	flags.IntVarP(&benchPoolSize, "pool", "p", 1, "Pool size (number of parallel requests)")
	flags.IntVarP(&benchMaxRequests, "max", "m", 100, "Total number of Stat requests sent")
	flags.BoolVarP(&benchSkipCreate, "no_create", "n", false, "Skip test resource creation (if it is already existing)")
	flags.BoolVar(&benchSkipClean, "no_clean", false, "Skip cleaning the resources that have been created for this test")
}
