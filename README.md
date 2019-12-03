<img src="https://github.com/pydio/cells/wiki/images/PydioCellsColor.png" width="400" />

[Homepage](https://pydio.com/) | [GitHub-Repository](https://github.com/pydio/cells-client) |
[Issue-Tracker](https://github.com/pydio/cells-client/issues)

[![License Badge](https://img.shields.io/badge/License-Apache2-blue.svg)](LICENSE)
[![Build Status](https://travis-ci.org/pydio/cells-client.svg?branch=master)](https://travis-ci.org/pydio/cells-client)
[![Go Report Card](https://goreportcard.com/badge/github.com/pydio/cells-client?rand=2)](https://goreportcard.com/report/github.com/pydio/cells-client)

This command line client allows interacting with a [Pydio Cells](https://github.com/pydio/cells) server via the command line. It uses the [Cells SDK for Go](https://github.com/pydio/cells-sdk-go) and the REST API under the hood.

## Download

We provide binaries for the following amd64 architectures:

- [MacOS](https://download.pydio.com/latest/cells-client/release/{latest}/darwin-amd64/cec)
- [Windows](https://download.pydio.com/latest/cells-client/release/{latest}/windows-amd64/cec.exe)
- [Linux](https://download.pydio.com/latest/cells-client/release/{latest}/linux-amd64/cec)

## Installation

We do not provide a packaged installer for the various OSs.  
Yet, Cells Client is a single self-contained binary file and is easy to install. 

Typically on Linux, you have to:

- Download the [latest binary file](https://download.pydio.com/latest/cells-client/release/{latest}/linux-amd64/cec) from Pydio website,
- Make it executable: `chmod u+x cec`,
- Put it in your path or add a symlink to the binary location, typically:  
  `sudo ln -s /<path-to-bin>/cec /usr/local/bin/cec`  
  This last step is only required if you want to configure the completion helper (see below).  
  Otherwise, you can also do `./cec ls` directly.

You can verify that `cec` is correctly installed and configured by launching any command, for instance:  
`cec version show`

###  Installing from source 

If you want to install from source, you must have go version 1.12+ installed and configured on your machine and the necessary build utils (typically `make`, `gcc`, ...). You can then download the source code and use the Makefile to compile a binary for your OS:

```sh
git clone https://github.com/pydio/cells-client.git
cd ./cells-client
make dev
```

_Note: Cells Client uses the Go Modules mechanism to manage dependencies, so you do not have to be in your go path._

## Configuration

You must first configure the client to connect to the server.

```sh
./cec oauth
```

You are prompted for following informations:

- Server Address : full URL to Cells, e.g.: `https://cells.yourdomain.com/`
- Client ID / Client Secret: this is used by the OpenIDConnect service for authentication.  
  Note that since the v2.0, a default **public** client is registered by default, using the suggested default values should work out of the box:
  - Client ID: `cells-client`
  - Client Secret: (leave empty)
- Then follow the OAuth2 process either by opening a browser or copy/pasting the URL in your browser to get a valid token.
- The token is automatically saved in your keychain. It will be refreshed as necessary.

## Usage

Use the `cec --help` command to know about available commands. Below are a few interresting ones for manipulating files:

- `cec ls`: List files and folders on the server, when no path is provided, it lists the workspaces that current user can access.
- `cec scp`: Upload / Download file to/from a remote server (see below).
- `cec cp`, `cec cp` and `cec rm`: Copy, move, rename and delete files **within the server**.
- `cec mkdir`: Create a folder on the remote server
- `cec clear`: Clear authentication tokens stored in your keychain.

## Command completion for BASH

Make sure that you have bash-completion installed

`apt-get install bash-completion` `brew install bash-completion`

MacOS users need to add `cec completion bash > /usr/local/etc/bash_completion.d/cec`.

Linux users with `cec completion bash > /etc/bash_completion.d/cec`

Otherwise you can source it to the current session with:
`source <(cec completion bash)`

You should have a `cec` binary available


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
$ ./cec scp ./README.md cells://common-files/
Copying ./README.md to cells://common-files/
 ## Waiting for file to be indexed...
 ## File correctly indexed
```

### 4/ Download a file from server

```sh
$ ./cec scp cells://personal-files/IMG_9723.JPG ./
Copying cells://personal-files/IMG_9723.JPG to ./
Written 822601 bytes to file
```

## License

This project is licensed under the Apache V2 License - see the LICENSE file for details.
