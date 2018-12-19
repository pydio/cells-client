# cells-client

This command line client allows interacting with a [Pydio Cells](https://github.com/pydio/cells) server via the command line. It uses the [Cells-sdk-go](https://github.com/pydio/cells-sdk-go) and the REST API under the hood.

## Installation

Download source code and use the Makefile to compile binary on your os

```
$ go get -u github.com/pydio/cells-client
$ cd $GOPATH/github.com/pydio/cells-client
$ make dev
```

You should have a `cec` binary available

## Configuration

You must first configure the client to connect to the server. 

```
$ ./cec configure
```

You will be prompted with the following informations : 

 - Server Address : full URL to Cells, e.g. https://cells.yourdomain.com/
 - Client ID / Client Secret: this is used by the OpenIDConnect service for authentication. Look in your server `pydio.json` file for the following section (see below), **Id** is the Client ID and **Secret** is the client Secret.
```json
         "staticClients": [
           {
             "Id": "cells-front",
             "IdTokensExpiry": "10m",
             "Name": "cells-front",
             "OfflineSessionsSliding": true,
             "RedirectURIs": [
               "http://localhost:8080/auth/callback"
             ],
             "RefreshTokensExpiry": "30m",
             "Secret": "Nqjuhpzl839618VrbLrnEPyn"
           }
         ],

```
 - User Login and password
 
## Usage

Use the `cec --help` command to know about the available commands. There are currently two interesting commands for manipulating files : 

- `cec ls` : list files and folders on the server, when no path is passed, it lists the workspaces that use has access to.

- `cec cp` : Upload / Download file to/from a remote server.

Other commands are available for listing datasources, users, roles, etc... but it is still a WIP.

## Examples

**1/ Listing the content of the personal-files workspace**

```
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

**2/ Showing details about a file**

```
$ ./cec ls personal-files/P5021040.jpg -d
Listing: 1 results for personal-files/P5021040.jpg
+------+--------------------------------------+-----------------------------+--------+------------+
| TYPE |                 UUID                 |            NAME             |  SIZE  |  MODIFIED  |
+------+--------------------------------------+-----------------------------+--------+------------+
| File | 98bbd86c-acb9-4b56-a6f3-837609155ba6 | personal-files/P5021040.jpg | 3.1 MB | 5 days ago |
+------+--------------------------------------+-----------------------------+--------+------------+

```

**3/ Uploading a file to server**

```
$ ./cec cp ./README.md cells://common-files/
Copying ./README.md to cells://common-files/
 ## Waiting for file to be indexed...
 ## File correctly indexed
```
**4/ Download a file from server**

```
$ ./cec cp cells://personal-files/IMG_9723.JPG ./
Copying cells://personal-files/IMG_9723.JPG to ./
Written 822601 bytes to file
```


## License

This project is licensed under the Apache V2 License - see the LICENSE file for details