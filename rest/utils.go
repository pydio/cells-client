package rest

import (
	"path"
	"path/filepath"
	"strings"
	"time"
)

// RetryCallback implements boiler plate code to easily call the same function until it suceeds
// or a time-out is reached.
func RetryCallback(callback func() error, number int, interval time.Duration) error {

	var e error
	for i := 0; i < number; i++ {
		if e = callback(); e == nil {
			break
		}
		if i < number-1 {
			<-time.After(interval)
		}
	}

	return e
}

// tbd: download -> /Users/j/downloads/ + /personal-files/folder + /personal-files/folder/meteo.jpg = /Users/j/downloads/folder/meteo.jpg
func TargetLocation(targetPath, sourcePath, nodeSourcePath string) string {

	sourcePath = strings.Trim(sourcePath, "/")
	nodeSourcePath = strings.Trim(nodeSourcePath, "/")
	serverBase := path.Base(sourcePath)
	relativePath := strings.TrimPrefix(nodeSourcePath, sourcePath)

	return filepath.Join(targetPath, serverBase, relativePath)
}
