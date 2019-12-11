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
  This last step is **required** if you want to configure the completion helper (see below).  
  Otherwise, you can also do `./cec ls` directly (in such case, adapt the suggested commands to run the examples).

You can verify that `cec` is correctly installed and configured by launching any command, for instance:  
`cec version`

###  Installing from source 

If you want to install from source, you must have go version 1.12+ installed and configured on your machine and the necessary build utils (typically `make`, `gcc`, ...). You can then download the source code and use the Makefile to compile a binary for your OS:

```sh
git clone https://github.com/pydio/cells-client.git
cd ./cells-client
make dev
```

_Note: Cells Client uses the Go Modules mechanism to manage dependencies, so you do not have to be in your GOPATH._

## Configuration

This step is compulsory if you do not want to precise all configuration information each time you call the command, typically on your working station. It guides through a few steps to gather necessary information and store sensitive bits in your keyring _**if you have one configured and running on your machine**_.

Default authentication mechanism is a OAuth _Authorization Code_ flow. 

```sh
# simply call
cec configure
```

You are then prompted for the following information:

- Server Address: full URL to Cells, e.g.: `https://cells.yourdomain.com/`
- Client ID / Client Secret: this is used by the OpenIDConnect service for authentication.  
  Note that since the v2.0, a default **public** client is registered by default, using the suggested default values should work out of the box:
  - Client ID: `cells-client`
  - Client Secret: (leave empty)
- Then follow the OAuth2 process either by opening a browser or copy/pasting the URL in your browser to get a valid token.
- The token is automatically saved in your keychain. It will be refreshed as necessary.

**Example:**

Assuming that I have a Pydio Cells instance running under this URL `https://cells.my-files.com` and that I am running the command on the same **graphical environment**.

``` sh
$ cec configure
Server Address (provide a valid URL): https://cells.my-files.com
✔ No
OAuth APP ID (found in your server pydio.json): cells-client
OAuth APP Secret (leave empty for a public client):
✔ Yes
Opening URL https://cells.my-files.com/oidc/oauth2/auth?client_id=cells-client&redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fservers%2Fcallback&response_type=code&state=XVlBzgbaiCMRAjWw
Now exchanging the code for a valid IdToken
✔Successfully Received Token!
⚠ Testing this configuration before saving
✔ Successfully logged to server, token will be refreshed at 2019-12-09 12:42:58 +0100 CET
✔ Configuration saved, you can now use the client to interract with https://cells.my-files.com.
```

*If you have no tab opening in your browser you can manually copy the URL and put it in your browser*

## Command completion for BASH

Cells Client provides a handy feature that provides completion on both available commands and path, both on local and remote machines.

This feature requires that you have `bash-completion` third party add-on installed on your workstation.

```sh
## On Linux, you must insure the 'bash-completion' library is installed:
# on Debian / Ubuntu
sudo apt install bash-completion

# on RHEL / CentOS
sudo yum install bash-completion

# on MacOS (make sure to follow the instructions displayed by Homebrew)
brew install bash-completion
```

_MacOS users should update their bash version to v5, (by default it is using v3)_

Then to add the completion in a persistent manner:

- Linux users: `cec completion bash | sudo tee /etc/bash_completion.d/cec`
- MacOS users: `cec completion bash | sudo tee /usr/local/etc/bash_completion.d/cec`.

Otherwise you can source it to the current session with:
`source <(cec completion bash)`

Note: if you want to use completion for remote paths while using `scp` sub command, you have prefix the _remote_ path with `cells//` rather than `cells://` - that is omit column character before the double slash. Typically:

```sh
cec scp ./README.md cells//com <press the tab key>
# will complete the path to 
cec scp ./README.md cells//common-files/
...
```

Note: when you update the Cells Client, you also have to update the completion file, typically on linux machines:

```sh
cec completion bash | sudo tee /etc/bash_completion.d/cec
source /etc/bash_completion.d/cec
```

## Usage

Use the `cec --help` command to know about available commands. Below are a few interresting ones for manipulating files:

- `cec ls`: List files and folders on the server, when no path is provided, it lists the workspaces that the current user can access.
- `cec scp`: Upload/Download file to/from a remote server.
- `cec cp`, `cec cp` and `cec rm`: Copy, move, rename and delete files **within the server**.
- `cec mkdir`: Create a folder on the remote server
- `cec clear`: Clear authentication tokens stored in your keychain.

For your convenience, below are a few examples.

### 1/ Listing the content of the personal-files workspace

```sh
$ cec ls personal-files
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
$ cec ls personal-files/P5021040.jpg -d
Listing: 1 results for personal-files/P5021040.jpg
+------+--------------------------------------+-----------------------------+--------+------------+
| TYPE |                 UUID                 |            NAME             |  SIZE  |  MODIFIED  |
+------+--------------------------------------+-----------------------------+--------+------------+
| File | 98bbd86c-acb9-4b56-a6f3-837609155ba6 | personal-files/P5021040.jpg | 3.1 MB | 5 days ago |
+------+--------------------------------------+-----------------------------+--------+------------+
```

### 3/ Uploading a file to server

```sh
$ cec scp ./README.md cells://common-files/
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
