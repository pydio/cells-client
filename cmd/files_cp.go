package cmd

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/micro/go-log"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
	"github.com/pydio/cells-sdk-go/models"
)

var cpFiles = &cobra.Command{
	Use:   "cp",
	Short: `Copy files from/to Cells`,
	Long: `Copy files between local server and remote Cells server

Prefix remote paths with cells:// to differentiate local from remote. Currently, copy can only be performed with both different ends.
For example:

1/ Uploading a file to server

$ ./cec cp ./README.md cells://common-files/
Copying ./README.md to cells://common-files/
 ## Waiting for file to be indexed...
 ## File correctly indexed

2/ Download a file from server

$ ./cec cp cells://personal-files/IMG_9723.JPG ./
Copying cells://personal-files/IMG_9723.JPG to ./
Written 822601 bytes to file


`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) < 2 {
			log.Fatal(fmt.Errorf("please provide at least a source and a destination target"))
		}

		from := args[0]
		to := args[1]
		fmt.Printf("Copying %s to %s\n", from, to)

		if strings.HasPrefix(from, "cells://") {
			// Download
			fromPath := strings.TrimPrefix(from, "cells://")
			toPath, remote, e := targetToFullPath(to, from)
			if e != nil {
				log.Fatal(e)
			}
			if remote {
				log.Fatal(fmt.Errorf("source and target are both remote, copy remote to local or the opposite"))
			}
			reader, e := rest.GetFile(fromPath)
			if e != nil {
				log.Fatal(e)
			}
			writer, e := os.OpenFile(toPath, os.O_CREATE|os.O_WRONLY, 0755)
			if e != nil {
				log.Fatal(e)
			}
			defer writer.Close()
			written, e := io.Copy(writer, reader)
			if e != nil {
				log.Fatal(e)
			}
			fmt.Printf("Written %d bytes to file\n", written)
		} else {
			// Upload
			toPath, remote, e := targetToFullPath(to, from)
			if e != nil {
				log.Fatal(e)
			}
			if !remote {
				log.Fatal(fmt.Errorf("source and target are both local, copy remote to local or the opposite"))
			}
			if s, e := os.Stat(from); e != nil || s.IsDir() {
				log.Fatal(fmt.Errorf("local source is not a valid file"))
			}
			reader, e := os.Open(from)
			if e != nil {
				log.Fatal(e)
			}
			_, e = rest.PutFile(toPath, reader, true)
			if e != nil {
				log.Fatal(e)
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(cpFiles)
}

func targetToFullPath(to string, from string) (string, bool, error) {
	var toPath string
	var isDir bool
	var isRemote bool
	var e error
	if strings.HasPrefix(to, "cells://") {
		// This is remote
		isRemote = true
		toPath = strings.TrimPrefix(to, "cells://")
		target, ok := rest.StatNode(toPath)
		if !ok {
			// Does not exists => will be created
			return toPath, isRemote, nil
		}
		isDir = target.Type == models.TreeNodeTypeCOLLECTION
	} else {
		// This is local
		toPath, e = filepath.Abs(to)
		if e != nil {
			return "", false, e
		}
		s, e := os.Stat(toPath)
		if e != nil {
			return "", false, e
		}
		isDir = s.IsDir()
	}

	if isDir {
		toPath = path.Join(toPath, path.Base(from))
	}
	return toPath, isRemote, nil
}
