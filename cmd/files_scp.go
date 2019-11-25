package cmd

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gosuri/uiprogress"

	"github.com/micro/go-log"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
	"github.com/pydio/cells-sdk-go/models"
)

const scpFileExample = `
Prefix remote paths with cells:// to differentiate local from remote. Currently, copy can only be performed with both different ends.
For example:

1/ Uploading a file to server

$ ./cec scp ./README.md cells://common-files/
Copying ./README.md to cells://common-files/
 ## Waiting for file to be indexed...
 ## File correctly indexed

2/ Download a file from server

$ ./cec scp cells://personal-files/IMG_9723.JPG ./
Copying cells://personal-files/IMG_9723.JPG to ./
Written 822601 bytes to file`

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
			reader, length, e := rest.GetFile(fromPath)
			if e != nil {
				log.Fatal(e)
			}
			bar := uiprogress.AddBar(length).PrependCompleted()
			wrapper := &PgReader{
				Reader: reader,
				bar:    bar,
				total:  length,
			}
			writer, e := os.OpenFile(toPath, os.O_CREATE|os.O_WRONLY, 0755)
			if e != nil {
				log.Fatal(e)
			}
			defer writer.Close()
			uiprogress.Start()
			_, e = io.Copy(writer, wrapper)
			if e != nil {
				log.Fatal(e)
			}
			// Wait that progress bar finish rendering
			<-time.After(100 * time.Millisecond)
			uiprogress.Stop()
		} else {
			// Upload
			toPath, remote, e := targetToFullPath(to, from)
			if e != nil {
				log.Fatal(e)
			}
			if !remote {
				log.Fatal(fmt.Errorf("source and target are both local, copy remote to local or the opposite"))
			}
			var length int64
			if s, e := os.Stat(from); e != nil || s.IsDir() {
				log.Fatal(fmt.Errorf("local source is not a valid file"))
			} else {
				length = s.Size()
			}
			reader, e := os.Open(from)
			bar := uiprogress.AddBar(int(length)).PrependCompleted()
			wrapper := &PgReader{
				Reader: reader,
				Seeker: reader,
				bar:    bar,
				total:  int(length),
				double: true,
			}
			if e != nil {
				log.Fatal(e)
			}
			uiprogress.Start()
			_, e = rest.PutFile(toPath, wrapper, false)
			if e != nil {
				log.Fatal(e)
			}
			<-time.After(500 * time.Millisecond)
			uiprogress.Stop()
			// Now stat Node to make sure it is indexed
			e = rest.RetryCallback(func() error {
				fmt.Println(" ## Waiting for file to be indexed...")
				_, ok := rest.StatNode(toPath)
				if !ok {
					return fmt.Errorf("cannot stat node just after PutFile operation")
				}
				return nil

			}, 3, 3*time.Second)
			if e != nil {
				log.Fatal("File does not seem to be indexed!")
			}
			fmt.Println(" ## File correctly indexed")

		}
	},
}

func init() {
	RootCmd.AddCommand(scpFiles)
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

type PgReader struct {
	io.Reader
	io.Seeker
	bar   *uiprogress.Bar
	total int
	read  int

	double bool
	first  bool
}

func (r *PgReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if err == nil {
		if r.double {
			r.read += n / 2
		} else {
			r.read += n
		}
		r.bar.Set(r.read)
	} else if err == io.EOF {
		if r.double && !r.first {
			r.first = true
			r.bar.Set(r.total / 2)
		} else {
			r.bar.Set(r.total)
		}
	}
	return
}

func (r *PgReader) Seek(offset int64, whence int) (int64, error) {
	if r.double && r.first {
		r.read = r.total/2 + int(offset)/2
	} else {
		r.read = int(offset)
	}
	r.bar.Set(r.read)
	return r.Seeker.Seek(offset, whence)
}
