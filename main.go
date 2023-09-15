package main

import (
	"github.com/pydio/cells-client/v4/cmd"
	"github.com/pydio/cells-client/v4/common"
)

func main() {
	common.PackageType = "CellsClient"
	common.PackageLabel = "Cells Client"
	cmd.RootCmd.Execute()
}
