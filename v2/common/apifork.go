// Package common centralize a "hard copy" of some of the objects that are defined in Cells
// to avoid getting all Cells dependency for just a few types. This will be refactored
// once we have refactored and modulified the Cells main code.
package common

import "time"

type ServerVersion struct {
	PackageType   string
	PackageLabel  string
	Version       string
	License       string
	BuildRevision string
	BuildStamp    time.Time
	PackagingInfo string
	ServerOffset  int64
}
