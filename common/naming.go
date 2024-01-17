package common

var (
	// AppName stores the technical name of the Cells Client application.
	AppName = "cells-client"

	PackageType  string
	PackageLabel string
	Version      string
)

const (
	// OAuthType uses OAuth2 credential retrieval flow.
	OAuthType = "OAuth2"
	// PatType relies on a Personal Access Token generated on the server for a given user.
	PatType = "Personal Access Token"
	// ClientAuthType is the legacy authentication method, based on user password.
	ClientAuthType = "Client Credentials"

	// UpdateServerURL gives access to Pydio's update server.
	UpdateServerURL = "https://updatecells.pydio.com/"
	// UpdatePublicKey enables verification of dowloaded binaries.
	UpdatePublicKey = "-----BEGIN PUBLIC KEY-----\nMIIBCgKCAQEAwh/ofjZTITlQc4h/qDZMR3RquBxlG7UTunDKLG85JQwRtU7EL90v\nlWxamkpSQsaPeqho5Q6OGkhJvZkbWsLBJv6LZg+SBhk6ZSPxihD+Kfx8AwCcWZ46\nDTpKpw+mYnkNH1YEAedaSfJM8d1fyU1YZ+WM3P/j1wTnUGRgebK9y70dqZEo2dOK\nn98v3kBP7uEN9eP/wig63RdmChjCpPb5gK1/WKnY4NFLQ60rPAOBsXurxikc9N/3\nEvbIB/1vQNqm7yEwXk8LlOC6Fp8W/6A0DIxr2BnZAJntMuH2ulUfhJgw0yJalMNF\nDR0QNzGVktdLOEeSe8BSrASe9uZY2SDbTwIDAQAB\n-----END PUBLIC KEY-----"

	// UpdateStableChannel is mainstream update channel.
	UpdateStableChannel = "stable"
	// UpdateDevChannel enable updating before the official release for testing purposes.
	UpdateDevChannel = "dev"

	DefaultConfigFileName = "config.json"

	// EnvPrefix insures we have a reserved namespace for Cells Client specific ENV vars.
	EnvPrefix = "CEC"
)

var (
	UploadSwitchMultipart  = int64(100)
	UploadDefaultPartSize  = int64(50)
	UploadMaxPartsNumber   = int64(5000)
	UploadPartsSteps       = int64(10 * 1024 * 1024)
	UploadPartsConcurrency = 3
	UploadSkipMD5          = false
	S3RequestTimeout       = int64(-1)
)
