<img src="https://github.com/pydio/cells/wiki/images/PydioCellsColor.png" width="400" />

[Homepage](https://pydio.com/) | [GitHub-Repository](https://github.com/pydio/cells-client) |
[Issue-Tracker](https://github.com/pydio/cells-client/issues)

[![License Badge](https://img.shields.io/badge/License-Apache2-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/pydio/cells-client?rand=2)](https://goreportcard.com/report/github.com/pydio/cells-client)

Cells Client provides an easy way to communicate with a [Pydio Cells](https://github.com/pydio/cells) server instance from the command line (or from automation scripts). It uses the [Cells SDK for Go](https://github.com/pydio/cells-sdk-go) and the REST API under the hood.

Cells Client a.k.a `cec` works like standard command line tools like **ls**, **scp**, etc.  Using the `cec` command, you can list, download and upload directly to your remote Cells server.

The v4 version of this tools is meant to be used with a remote Cells server that is also at version v4+, please rather use cells-client v2 if your server is older (but you really should update your Cells server!).

## Usage

Use the `cec --help` command to know about available commands. Below are a few interesting ones for manipulating files:

- `cec ls`: List files and folders on the server, when no path is provided, it lists the workspaces that the current user can access.
- `cec scp`: Upload/Download file to/from a remote server.
- `cec cp`, `cec mv` and `cec rm`: Copy, move, rename and delete files **within the server**.
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
....
| Folder | recycle_bin              |
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

## Installation

Cells Client is a single self-contained binary file and is easy to install.

### 1 - Download cec

Grab the built version for your corresponding amd64 architectures:

- [Linux](https://download.pydio.com/latest/cells-client/release/{latest}/linux-amd64/cec)
- [MacOS](https://download.pydio.com/latest/cells-client/release/{latest}/darwin-amd64/cec)
- [Windows](https://download.pydio.com/latest/cells-client/release/{latest}/windows-amd64/cec.exe)

### 2 - Make it executable

Give execution permissions to the binary file, typically on Linux: `chmod u+x cec`.

### 3 - Add it to the PATH (optional)

Add the command to your `PATH` environment variable, to makes it easy to call the command from anywhere in the system. On Linux, you can for instance add a symlink to the binary location (replace below with correct path):

```sh
sudo ln -s /path/to/your/binary/cec /usr/local/bin/cec
```

### 4 - Check for correct installation

To verify that `cec` is correctly installed, simply run for instance:

```sh
$ cec version
# Should output something like below
Cells Client
 Version:       4.0.0-dev
 Git commit:    ebf227c3519e16a91159c0e9a6304d3778827694
 Timestamp:     2023-10-11T10:08:30Z
 OS/Arch:       linux/amd64
 Go version:    go1.21.3
```

## Connecting To Cells

Cells Client requires an authenticated connection to the target Cells server. For a given user, read/write permissions are applied in the same way as what you see in the web interface.

Once a valid user is available, there are 2 options:

- Go through an interactive configuration and persist necessary information on the client machine (Persistent Mode)
- Pass the necessary connection information at each call (Non Persistent Mode)

### Persistent Mode

Connections can be configured and persisted locally on the client machine. As from version **v2.2.0**, you can configure multiple _accounts_ on the client side and switch between them as necessary.
Last used connection is persisted locally in the main configuration file and will be re-used the next time you call the `cec` command.

To switch between accounts, simply call `cec config use`. 

Calling the `cec config add` command offers various authentication mechanisms. For persistent mode, we advise to use the default OAuth _Authorization Code_ flow.

```sh
cec config add oauth
```

You will be guided through a few steps to configure and persist your connection:

- Enter your server address: the full URL to access your Cells instance, e.g.: `https://files.example.com/`
- Choose OAuth2 process either by opening a browser or copy/pasting the URL in your browser to get a valid token
- Test and validate the connection.

The token is saved locally and will be refreshed automatically as required. If a keyring mechanism is available on the machine, it is used to store sensitive information. You can verify this with the following command:

```sh
cec config check-keyring 
```

Supported keyrings are MacOSX Keychain, Linux DBUS and Windows Credential Manager API.

#### Troubleshooting

When launching cells-client, following error is thrown:

```sh
2022/10/14 17:33:07 unexpected error during initialisation phase: exec: "dbus-launch": executable file not found in $PATH
```

This mostly happens when Cells Client cannot find a keyring that it expects to store credentials. Under Debian/Ubuntu Linux, you can install necessary software with (even if you do not use GNOME):

```sh 
sudo apt install dbus-launch libpam-gnome-keyring gnome-keyring-pkcs11
```

### Non Persistent Mode

This mode can be useful to use the Cells Client in a CI/CD pipe or via cron jobs. In such case, we strongly advise you to create a _Personal Access Token_ on the server and use this.

To create a token that is valid for user `robot` for 90 days, log via SSH into your server as `pydio` (a.k.a. as the user that **runs** the `cells` service) and execute:

```sh
$ cells admin user token -u robot -e 90d
✔ This token for robot will expire on Tuesday, 01-Jun-21 16:46:40 CEST.
✔ d-_-x3N8jg9VYegwf5KpKFTlYnQIzCrvbXHzS24uB7k.mibFBN2bGy3TUVzJvcrnUlI9UuM3-kzB1OekrPLLd4U
⚠ Make sure to secure it as it grants access to the user resources!
```

Note: if you have the Enterprise Edition, you can also manage Personal Access Token via the web GUI:

- Go to `Cells Admin Console >> Identity Management >> People`
- Edit the user you want to create a token for
- Go to the `Personal Access Token` page.

Then use environment variables (or the corresponding command flags) to pass connection information:

```sh
export CEC_URL=https://files.example.com
export CEC_TOKEN=d-_-x3N8jg9VYegwf5KpKFTlYnQIzCrvbXHzS24uB7k.mibFBN2bGy3TUVzJvcrnUlI9UuM3-kzB1OekrPLLd4U
```

You can now directly talk to your server, for instance:

```sh
cec ls common-files 
```

> Note that environment variables take the precedence as soon as you have the `CEC_URL` variable defined. In such case please insure you have the necessary variables defined depending on your target authentication mode. 

## Command Completion

Cells Client provides a handy feature that provides completion on commands and paths; both on local and remote machines.

_NOTE: you **must** add `cec` to you local `PATH` if you want to configure the completion helper (see above)._

### Bash completion

To enable this feature, you must have `bash-completion` third party add-on installed on your workstation.

```sh
# on Debian / Ubuntu
sudo apt install bash-completion

# on RHEL / Rocky Linux
sudo dnf install bash-completion

# on MacOS (make sure to follow the instructions displayed by Homebrew)
brew install bash-completion
```

_MacOS latest release changed the default shell to ZSH_.

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

## Build from source

If you rather want to directly compile the source code on your workstation, you require:

- Go language 1.20 or higher (tested with latest 1.20 & 1.21), with a [correctly configured](https://golang.org/doc/install#testing) Go toolchain,
- The necessary build utils (typically `make`, `gcc`, ...)
- A git client

You can then retrieve the source code and use the `Makefile` to compile a binary for your OS:

```sh
git clone https://github.com/pydio/cells-client.git
cd ./cells-client
make dev
```

Cells Client uses the Go Modules mechanism to manage dependencies: you can checkout the code anywhere in your local machine, it does not have to be within your `GOPATH`.

## License

This project is licensed under the Apache V2 License - see the LICENSE file for details.
