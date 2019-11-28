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
func TargetLocation(target, source, nodeSource string) string {

	source = strings.Trim(source, "/")
	nodeSource = strings.Trim(nodeSource, "/")
	serverBase := path.Base(source)
	relativePath := strings.TrimPrefix(nodeSource, source)

	return filepath.Join(target, serverBase, relativePath)
}
