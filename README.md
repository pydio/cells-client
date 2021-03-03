<img src="https://github.com/pydio/cells/wiki/images/PydioCellsColor.png" width="400" />

[Homepage](https://pydio.com/) | [GitHub-Repository](https://github.com/pydio/cells-client) |
[Issue-Tracker](https://github.com/pydio/cells-client/issues)

[![License Badge](https://img.shields.io/badge/License-Apache2-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/pydio/cells-client?rand=2)](https://goreportcard.com/report/github.com/pydio/cells-client)

This command line client allows interacting with a [Pydio Cells](https://github.com/pydio/cells) server via the command line. It uses the [Cells SDK for Go](https://github.com/pydio/cells-sdk-go) and the REST API under the hood.

We try our best to be backward compatible, yet you will have a better user experience if your server is up-to-date (typically in version 2.2+) and use the latest client (2.1+ at the time of writing).

Typically we introduced the Personal Access Token that is easier to use and more secure in version 2.2 of the Pydio Cells server.

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
- Put it in your path or add a symlink to the binary location, for instance:  
  `sudo ln -s /<path-to-bin>/cec /usr/local/bin/cec`

You can verify that `cec` is correctly installed and configured by launching any command, for instance:  
`cec version`

_NOTE: you **must** add `cec` to you local `PATH` if you want to configure the completion helper (see below). Otherwise, you can also call `./cec` directly (in such case, adapt the suggested commands to run the examples)._

### Build from source

If you rather want to directly compile the source code on your workstation, you require:

- Go language 1.13 or higher (tested with latest 1.13, 1.14 & 1.15), with a [correctly configured](https://golang.org/doc/install#testing) Go toolchain,
- The necessary build utils (typically `make`, `gcc`, ...)
- A git client

You can then retrieve the source code and use the `Makefile` to compile a binary for your OS:

```sh
git clone https://github.com/pydio/cells-client.git
cd ./cells-client/v2
make dev
```

#### Important Notes

Cells Client uses the Go Modules mechanism to manage dependencies, this has 2 consequences:

- as current active development cycle is 2.x, the latest code from master is **in the v2 subfolder**
- you can checkout the code anywhere in your local machine, it does not have to be within your `GOPATH`

## Connecting To Your Server

The Cells Client is just another client to talk to your Cells server instance to manage your files.  
Thus, it needs to establish a connection using a valid user with sufficient permission to achieve what you are trying to do, typically:

- you will not be able to download a file from a workspace where you do not have read access
- you need write access in the workspace where you want to upload

Once you have a valid user, you have 2 choices:

- Pass the necessary connection information at each call (Non Persistent Mode)
- Go through a configuration step and persists necessary information on the client machine (Persistent Mode)

### Non Persistent Mode

This is typically useful if you want to use the Cells Client in your CICD pipe or via cron jobs.  
In such case, we strongly advise that you create a Personal Access Token on the server and use this.

Let's say that you have created a user `robot` that has sufficient permissions for what you want to do.  
To create a token that is valid for 90 days, log via SSH into your server as `pydio` (a.k.a. as the user that **runs** the `cells` service) and execute:

```sh
$ cells admin user token -u robot -e 90d
✔ This token for robot will expire on Tuesday, 01-Jun-21 16:46:40 CEST.
✔ d-_-x3N8jg9VYegwf5KpKFTlYnQIzCrvbXHzS24uB7k.mibFBN2bGy3TUVzJvcrnUlI9UuM3-kzB1OekrPLLd4U
⚠ Make sure to secure it as it grants access to the user resources!
```

You can then use environment variables (or the corresponding flags) to configure the connection. Typically, in our case:

```sh
export CEC_URL=https://files.example.com
export CEC_TOKEN=d-_-x3N8jg9VYegwf5KpKFTlYnQIzCrvbXHzS24uB7k.mibFBN2bGy3TUVzJvcrnUlI9UuM3-kzB1OekrPLLd4U
```

You can then directly talk to your server, for instance:

```sh
cec ls common-files 
```

### Persistent Mode

In your local workstation, you can also interactively configure your connection once and store credential locally.

If you have a keyring that is correctly configured and running on your machine, we transparently use it to avoid storing sensitive information in clear text.  
You can simply test if the keyring is present and usable with:

```sh
cec configure check-keyring 
```

Calling the `cec configure` command let you then choose between the available authentication mechanism. For persistent mode, we advise to use the default OAuth _Authorization Code_ flow.

```sh
cec configure oauth
```

You are then guided through a few steps to configure and persist your connection. Mainly:

- Enter your server address: the full URL to access your Cells instance, e.g.: `https://files.example.com/`
- Choose OAuth2 process either by opening a browser or copy/pasting the URL in your browser to get a valid token
- Test and validate the connection.

The token is then automatically saved in your keychain and will be refreshed and stored again as necessary.

## Command Completion

Cells Client provides a handy feature that provides completion on commands and paths; both on local and remote machines.

To enable this feature, you must have `bash-completion` third party add-on installed on your workstation.

```sh
# on Debian / Ubuntu
sudo apt install bash-completion

# on RHEL / CentOS
sudo yum install bash-completion

# on MacOS (make sure to follow the instructions displayed by Homebrew)
brew install bash-completion
```

_MacOS users should update their bash version to v5, (by default it is using v3)_.

Then, to add the completion in a persistent manner:

```sh
# Linux users
cec completion bash | sudo tee /etc/bash_completion.d/cec
# MacOS users
cec completion bash | sudo tee /usr/local/etc/bash_completion.d/cec
```

You can also only _source_ the file in current session, the feature will be gone when you start a new shell.

```sh
source <(cec completion bash)
```

Note: if you want to use completion for remote paths while using `scp` sub command, you have to prefix the _remote_ path with `cells//` rather than `cells://`; that is to omit the column character before the double slash. Typically:

```sh
cec scp ./README.md cells//com <press the tab key>
# Completes the path to
cec scp ./README.md cells//common-files/
...
```

Note: when you update the Cells Client, you also have to update the completion file, typically on Linux machines:

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
$ cec scp cells://personal-files/IMG_9723.JPG ./
Copying cells://personal-files/IMG_9723.JPG to ./
Written 822601 bytes to file
```

## License

This project is licensed under the Apache V2 License - see the LICENSE file for details.
