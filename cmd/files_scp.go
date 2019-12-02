package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	. "github.com/pydio/cells-client/rest"
)

var scpFileExample = `
Prefix remote paths with cells:// to differentiate local from remote. Currently, copy can only be performed with both different ends.
For example:

1/ Uploading a file to server

$ ` + os.Args[0] + ` scp ./README.md cells://common-files/
Copying ./README.md to cells://common-files/
 ## Waiting for file to be indexed...
 ## File correctly indexed

2/ Download a file from server

$ ` + os.Args[0] + ` scp cells://personal-files/IMG_9723.JPG ./
Copying cells://personal-files/IMG_9723.JPG to /home/pydio/downloads/
`

const (
	prefixA = "cells://"
	prefixB = "cells//"
)

var (
	currentPrefix string
)

var scpFiles = &cobra.Command{
	Use:     "scp",
	Short:   `Copy files from/to Cells`,
	Long:    `Copy files between local server and remote Cells server`,
	Example: scpFileExample,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 2 {
			cmd.Help()
			log.Fatal(fmt.Errorf("please provide at least a source and a destination target"))
		}

		from := args[0]
		to := args[1]

		if strings.HasPrefix(from, prefixA) || strings.HasPrefix(to, prefixA) {
			currentPrefix = prefixA
		} else if strings.HasPrefix(from, prefixB) || strings.HasPrefix(to, prefixB) {
			currentPrefix = prefixB
		}

		DryRun = false // Debug option
		var crawlerPath, targetPath string
		var isLocal bool

		if strings.HasPrefix(from, currentPrefix) {
			// Download
			fromPath := strings.TrimPrefix(from, currentPrefix)
			localTarget, isRemote, e := targetToFullPath(to, from)
			if e != nil {
				log.Fatal(e)
			}
			if isRemote {
				log.Fatal(fmt.Errorf("source and target are both remote, copy remote to local or the opposite"))
			}
			crawlerPath = fromPath
			isLocal = false
			targetPath = localTarget
			fmt.Printf("Downloading %s to %s\n", from, to)

		} else {
			// Upload
			_, remote, e := targetToFullPath(to, from)
			if e != nil {
				log.Fatal(e)
			}
			if !remote {
				log.Fatal(fmt.Errorf("source and target are both local, copy remote to local or the opposite"))
			}
			crawlerPath = from
			isLocal = true
			targetPath = strings.TrimPrefix(to, currentPrefix)
			fmt.Printf("Uploading %s to %s\n", from, to)
		}

		crawler, e := NewCrawler(crawlerPath, isLocal)
		if e != nil {
			log.Fatal(e)
		}
		nn, e := crawler.Walk()
		if e != nil {
			log.Fatal(e)
		}
		targetNode := NewTarget(targetPath, crawler)

		pool := NewBarsPool(len(nn) > 1, len(nn))
		pool.Start()

		// CREATE FOLDERS
		e = targetNode.MkdirAll(nn, pool)
		if e != nil {
			log.Fatal(e)
		}

		// UPLOAD / DOWNLOAD FILES
		errs := targetNode.CopyAll(nn, pool)
		//pool.Stop()
		if len(errs) > 0 {
			log.Fatal(errs)
		}
	},
}

func init() {
	RootCmd.AddCommand(scpFiles)
}

func targetToFullPath(to, from string) (string, bool, error) {
	var toPath string
	//var isDir bool
	var isRemote bool
	var e error
	if strings.HasPrefix(to, currentPrefix) {
		// This is remote
		isRemote = true
		toPath = strings.TrimPrefix(to, currentPrefix)
		_, ok := StatNode(toPath)
		if !ok {
			// Does not exists => will be created
			return toPath, isRemote, nil
		}
	} else {
		// This is local
		toPath, e = filepath.Abs(to)
		if e != nil {
			return "", false, e
		}
		_, e := os.Stat(toPath)
		if e != nil {
			return "", false, e
		}
	}

	return toPath, isRemote, nil
}
