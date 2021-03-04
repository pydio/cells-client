<img src="https://github.com/pydio/cells/wiki/images/PydioCellsColor.png" width="400" />

[Homepage](https://pydio.com/) | [GitHub-Repository](https://github.com/pydio/cells-client) |
[Issue-Tracker](https://github.com/pydio/cells-client/issues)

[![License Badge](https://img.shields.io/badge/License-Apache2-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/pydio/cells-client?rand=2)](https://goreportcard.com/report/github.com/pydio/cells-client)

Cells Client provides an easy way to communicate with a [Pydio Cells](https://github.com/pydio/cells) server instance from the command line (or from automation scripts). It uses the [Cells SDK for Go](https://github.com/pydio/cells-sdk-go) and the REST API under the hood.

Cells Client a.k.a `cec` works like standard command line tools like **ls**, **scp**, etc.  Using the `cec` command, you can list, download and upload directly to your remote Cells server.

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
 Version:       2.1.0-rc1
 Built:         03 Mar 21 16:26 +0000
 Git commit:    4d09aa8e33fc60e65625e9f8435fd90b99c1b801
 OS/Arch:       linux/amd64
 Go version:    go1.15.5
```

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

## Connecting To Cells

Cells Client requires an authenticated connection to the target Cells server. For a given user, read/write permissions are applied in the same way as what you see in the web interface.

Once a valid user is available, there are 2 options:

- Go through an interactive configuration and persist necessary information on the client machine (Persistent Mode)
- Pass the necessary connection information at each call (Non Persistent Mode)

### Persistent Mode

Connection can be configured and persisted locally on the client machine.

Calling the `cec configure` command offers various authentication mechanisms. For persistent mode, we advise to use the default OAuth _Authorization Code_ flow.

```sh
cec configure oauth
```

You will be guided through a few steps to configure and persist your connection:

- Enter your server address: the full URL to access your Cells instance, e.g.: `https://files.example.com/`
- Choose OAuth2 process either by opening a browser or copy/pasting the URL in your browser to get a valid token
- Test and validate the connection.

The token is saved locally and will be refreshed automatically as required. If a keyring mechanism is available on the machine, it used to store sensitive information. You can verify this with the following command:

```sh
cec configure check-keyring 
```

Supported keyrings are MacOSX Keychain, Linux DBUS and Windows Credential Manager API.

### Non Persistent Mode

This mode can be useful to use the Cells Client in a CI/CD pipe or via cron jobs. In such case, we strongly advise you to create a _Personal Access Token_ on the server and use this.

To create a token that is valid for user `robot` for 90 days, log via SSH into your server as `pydio` (a.k.a. as the user that **runs** the `cells` service) and execute:

```sh
$ cells admin user token -u robot -e 90d
✔ This token for robot will expire on Tuesday, 01-Jun-21 16:46:40 CEST.
✔ d-_-x3N8jg9VYegwf5KpKFTlYnQIzCrvbXHzS24uB7k.mibFBN2bGy3TUVzJvcrnUlI9UuM3-kzB1OekrPLLd4U
⚠ Make sure to secure it as it grants access to the user resources!
```

Then use environment variables (or the corresponding command flags) to pass connection information:

```sh
export CEC_URL=https://files.example.com
export CEC_TOKEN=d-_-x3N8jg9VYegwf5KpKFTlYnQIzCrvbXHzS24uB7k.mibFBN2bGy3TUVzJvcrnUlI9UuM3-kzB1OekrPLLd4U
```

Now you can directly talk to your server, for instance:

```sh
cec ls common-files 
```

## Usage

Use the `cec --help` command to know about available commands. Below are a few interesting ones for manipulating files:

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

## Appendix 1 - Authentication Modes Pros and Cons

There are 3 authentication methods to establish the connection:

### Personal Access Token

A token can be generated on the server for a given user. It can be limited in time or you might choose the auto-refresh mode.

Note about the auto-refresh option:

- Let's say you have given a validity of 10 days.
- If you connect to your server within 10 days, the token's validity is extended for 10 more days **on the server side**: on the client side the **token string remains unchanged**.  
- If you have a cron job that runs once a week during the night, typically to push some backups to your server, the token remains valid undefinitly.
- If your server is down and _misses_ a week, the upload will fail the week after, because 14 days have passed and the token expired.

**Pros**:

- Secure
- Non Interactive
- On client side, you only have to give the URL and the token to establish the connection
- This is the best solution if the client machine is a headless server that has no Keyring and must communicate with your server with daemon processes, typically `cron` jobs.

**Cons**:

- To create a token, you must either have access to the server as privileged user or ask your sysadmin.

### OAuth2 Credential Flows

This is the recommended strategy for persitent mode on your local workstation.  
Calling `cec configure oauth` will guide you through a quick process to securely generate an ID token and a refresh token.

Under the hood, `cec` will watch the validity of the token. When necessary, it will issue a refresh request and stores the updated tokens in your keychain without you even noticing it.  
For the record, in Cells 2.2, the default validity period of the refresh token is 60 days.

As tokens are represented as unique (random) complicated strings, this approach makes it difficult to steal your token by only looking at it, even if it ends up in clear text and is shown to third persons.

**Pros**:

- Secure
- Any user can use her own account to configure a connection, without asking the sysadmin.

**Cons**:

- You must go through an interactive process configure your connection.

### Client Credential Flows

This legacy method is not recommended and might disappear in a future version.

**Pros**:

- You only have to enter URL, login and password
- Can be used via configure process or directly using flags/ENV variable at each call
- Any user can use her own account to configure a connection, without asking the system administrator.

**Cons**:

- The user password ends up stored in clear text in case no keyring is present
- The process will fail if your server relies on external user repository to manage authentication (typically LDAP or SSO).

## Appendix 2 - Command Completion

Cells Client provides a handy feature that provides completion on commands and paths; both on local and remote machines.

_NOTE: you **must** add `cec` to you local `PATH` if you want to configure the completion helper (see above)._

### Bash completion

To enable this feature, you must have `bash-completion` third party add-on installed on your workstation.

```sh
# on Debian / Ubuntu
sudo apt install bash-completion

# on RHEL / CentOS
sudo yum install bash-completion

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
