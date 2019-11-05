<img src="https://github.com/pydio/cells/wiki/images/PydioCellsColor.png" width="400" />

[Homepage](https://pydio.com/) | [GitHub-Repository](https://github.com/pydio/cells-client) |
[Issue-Tracker](https://github.com/pydio/cells-client/issues)

[![License Badge](https://img.shields.io/badge/License-Apache2-blue.svg)](LICENSE)
[![Build Status](https://travis-ci.org/pydio/cells-client.svg?branch=master)](https://travis-ci.org/pydio/cells-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/pydio/cells-client?rand=2)](https://goreportcard.com/report/github.com/pydio/cells-client)

This command line client allows interacting with a [Pydio Cells](https://github.com/pydio/cells) server via the command line. It uses the [Cells SDK for go](https://github.com/pydio/cells-sdk-go) and the REST API under the hood.

## Installation

Download source code and use the Makefile to compile binary on your os

```sh
go get -u github.com/pydio/cells-client
cd $GOPATH/github.com/pydio/cells-client
make dev
```

You should have a `cec` binary available

## Configuration

You must first configure the client to connect to the server.

```sh
./cec oauth
```

You will be prompted with the following informations:

- Server Address : full URL to Cells, e.g. `https://cells.yourdomain.com/`
- Client ID / Client Secret: this is used by the OpenIDConnect service for authentication: using Cells 2.0, a default public client `cells-client` is already created. 
- Then follow the OAuth2 process either by opening a browser or copy/pasting the URL in your browser to get a valid token.
- The token is automatically saved in your keychain, and will be refreshed as necessary.

## Usage

Use the `cec --help` command to know about the available commands. There are currently two interesting commands for manipulating files:

- `./cec ls` : list files and folders on the server, when no path is passed, it lists the workspaces that use has access to.
- `./cec cp` : Upload / Download file to/from a remote server (see below).
- `./cec mkdir` : Create a folder on remote server
- `./cec clear` : Clear authentication tokens stored in your keychain.

Other commands are available for listing datasources, users, roles, etc... but it is still a WIP.

## Examples

### 1/ Listing the content of the personal-files workspace

```sh
$ ./cec ls personal-files
+--------+--------------------------+
|  TYPE  |           NAME           |
+--------+--------------------------+
| Folder | personal-files           |
| File   | Huge Photo-1.jpg         |
| File   | Huge Photo.jpg           |
| File   | IMG_9723.JPG             |
| File   | P5021040.jpg             |
| Folder | UPLOAD                   |
| File   | anothercopy              |
| File   | cec22                    |
| Folder | recycle_bin              |
| File   | test_crud-1545206681.txt |
| File   | test_crud-1545206846.txt |
| File   | test_file2.txt           |
+--------+--------------------------+
```

### 2/ Showing details about a file

```sh
$ ./cec ls personal-files/P5021040.jpg -d
Listing: 1 results for personal-files/P5021040.jpg
+------+--------------------------------------+-----------------------------+--------+------------+
| TYPE |                 UUID                 |            NAME             |  SIZE  |  MODIFIED  |
+------+--------------------------------------+-----------------------------+--------+------------+
| File | 98bbd86c-acb9-4b56-a6f3-837609155ba6 | personal-files/P5021040.jpg | 3.1 MB | 5 days ago |
+------+--------------------------------------+-----------------------------+--------+------------+

```

### 3/ Uploading a file to server

```sh
$ ./cec cp ./README.md cells://common-files/
Copying ./README.md to cells://common-files/
 ## Waiting for file to be indexed...
 ## File correctly indexed
```

### 4/ Download a file from server

```sh
$ ./cec cp cells://personal-files/IMG_9723.JPG ./
Copying cells://personal-files/IMG_9723.JPG to ./
Written 822601 bytes to file
```

## License

This project is licensed under the Apache V2 License - see the LICENSE file for details.
