package common

import (
	"time"

	cellsSdk "github.com/pydio/cells-sdk-go/v5"
)

var (
	// AppName stores the technical name of the Cells Client application.
	AppName = "cells-client"

	PackageType  string
	PackageLabel string
	Version      = "4.2.0-local"
)

const (
	// UpdateServerURL gives access to Pydio's update server.
	UpdateServerURL = "https://updatecells.pydio.com/"
	// UpdatePublicKey enables verification of downloaded binaries.
	UpdatePublicKey = "-----BEGIN PUBLIC KEY-----\nMIIBCgKCAQEAwh/ofjZTITlQc4h/qDZMR3RquBxlG7UTunDKLG85JQwRtU7EL90v\nlWxamkpSQsaPeqho5Q6OGkhJvZkbWsLBJv6LZg+SBhk6ZSPxihD+Kfx8AwCcWZ46\nDTpKpw+mYnkNH1YEAedaSfJM8d1fyU1YZ+WM3P/j1wTnUGRgebK9y70dqZEo2dOK\nn98v3kBP7uEN9eP/wig63RdmChjCpPb5gK1/WKnY4NFLQ60rPAOBsXurxikc9N/3\nEvbIB/1vQNqm7yEwXk8LlOC6Fp8W/6A0DIxr2BnZAJntMuH2ulUfhJgw0yJalMNF\nDR0QNzGVktdLOEeSe8BSrASe9uZY2SDbTwIDAQAB\n-----END PUBLIC KEY-----"

	// UpdateStableChannel is mainstream update channel.
	UpdateStableChannel = "stable"
	// UpdateDevChannel enable updating before the official release for testing purposes.
	UpdateDevChannel = "dev"

	DefaultConfigFileName = "config.json"

	// EnvPrefix insures we have a reserved namespace for Cells Client specific ENV vars.
	EnvPrefix = "CEC"
)

// Labels for well-known supported auth types
const (
	AuthTypePatLabel   = "PAT"
	AuthTypeBasicLabel = "Login/Password"
	AuthTypeOAuthLabel = "OAuth2"
)

func GetAuthTypeLabel(authType string) string {
	var label string
	switch authType {
	case cellsSdk.AuthTypeOAuth:
		label = AuthTypeOAuthLabel
	case cellsSdk.AuthTypePat:
		label = AuthTypePatLabel
	case cellsSdk.AuthTypeClientAuth:
		label = AuthTypeBasicLabel
	case LegacyCecConfigAuthTypePat,
		LegacyCecConfigAuthTypeBasic,
		LegacyCecConfigAuthTypeOAuth:
		// TODO this should never be used, remove once we are confidant the migration has been correctly implemented
		label = "Unmigrated - " + authType
	default:
		label = "Unknown"
	}
	return label
}

// Legacy values before we moved this in the Cells SDK (for v5+)
const (
	LegacyCecConfigAuthTypePat   = "Personal Access Token"
	LegacyCecConfigAuthTypeBasic = "Client Credentials"
	LegacyCecConfigAuthTypeOAuth = "OAuth2"
)

type LogLevel int

const (
	Trace LogLevel = iota
	Debug
	Info
	Warn
	Error
	Fatal

	TransferRetryMaxAttemptsDefault = 3
	TransferRetryMaxBackoffDefault  = time.Second * 3
)

var (
	UploadSwitchMultipart = int64(100)
	UploadDefaultPartSize = int64(50)
	UploadMaxPartsNumber  = int64(5000)

	TransferRetryMaxAttempts = TransferRetryMaxAttemptsDefault
	TransferRetryMaxBackoff  = TransferRetryMaxBackoffDefault

	UploadPartsSteps       = int64(10 * 1024 * 1024)
	UploadPartsConcurrency = 3
	UploadSkipMD5          = false
	S3RequestTimeout       = int64(-1)

	CurrentLogLevel = Info
)
