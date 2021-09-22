package main

import (
	"github.com/pydio/cells-client/v2/cmd"
	"github.com/pydio/cells-client/v2/common"
)

func main() {
	common.AppName = "cells-client"
	common.PackageType = "CellsClient"
	common.PackageLabel = "Cells Client"
	cmd.RootCmd.Execute()
}
