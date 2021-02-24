// package common centralizes constants for the cells-client application.
package common

var (
	PackageType   string
	PackageLabel  string
	BuildStamp    string
	BuildRevision string
	Version       string
)

const (
	// OAuthType uses OAuth2 credentil retrival flow.
	OAuthType = "oauth"
	// PatType relies on a Personal Access Token generated on the server for a given user.
	PatType = "personal-access-token"
	// ClientAuthType is the legacy authenticaion method, based on user password.
	ClientAuthType = "client-credentials"

	// UpdateServerURL gives access to Pydio's update server.
	UpdateServerURL = "https://updatecells.pydio.com/"
	// UpdatePublicKey enables verification of dowloaded binaries.
	UpdatePublicKey = "-----BEGIN PUBLIC KEY-----\nMIIBCgKCAQEAwh/ofjZTITlQc4h/qDZMR3RquBxlG7UTunDKLG85JQwRtU7EL90v\nlWxamkpSQsaPeqho5Q6OGkhJvZkbWsLBJv6LZg+SBhk6ZSPxihD+Kfx8AwCcWZ46\nDTpKpw+mYnkNH1YEAedaSfJM8d1fyU1YZ+WM3P/j1wTnUGRgebK9y70dqZEo2dOK\nn98v3kBP7uEN9eP/wig63RdmChjCpPb5gK1/WKnY4NFLQ60rPAOBsXurxikc9N/3\nEvbIB/1vQNqm7yEwXk8LlOC6Fp8W/6A0DIxr2BnZAJntMuH2ulUfhJgw0yJalMNF\nDR0QNzGVktdLOEeSe8BSrASe9uZY2SDbTwIDAQAB\n-----END PUBLIC KEY-----"

	// UpdateStableChannel is mainstream update channel.
	UpdateStableChannel = "stable"
	// UpdateDevChannel enable updating before the official release for testing purposes.
	UpdateDevChannel = "dev"
)

