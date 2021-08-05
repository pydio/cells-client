package bench

import (
	"github.com/pydio/cells-client/v2/cmd"
	"github.com/spf13/cobra"
)

var (
	benchPoolSize    int
	benchMaxRequests int
	benchSkipCreate  bool
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
	flags.IntVarP(&benchPoolSize, "pool", "p", 1, "Pool size (number of parallel requests)")
	flags.IntVarP(&benchMaxRequests, "max", "m", 100, "Total number of Stat requests sent")
	flags.BoolVarP(&benchSkipCreate, "no-create", "n", false, "Skip test resource creation (if it is already existing)")
}
