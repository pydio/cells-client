package main

import (
	"github.com/pydio/cells-client/v2/cmd"
	"github.com/pydio/cells-client/v2/common"

	// Force import of sub-commands that are in children packages
	_ "github.com/pydio/cells-client/v2/cmd/bench"
)

func main() {
	common.PackageType = "CellsClient"
	common.PackageLabel = "Cells Client"
	cmd.RootCmd.Execute()
}
