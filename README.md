# cells-client

This command line client allows interacting with an Pydio Cells server via the command line. It uses the REST API under the hood.

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


## License

This project is licensed under the Apache V2 License - see the LICENSE file for details