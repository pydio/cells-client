// Package cmd implements basic use cases to manage your files on your remote server
// via the command line of your local workstation or any server you can access with SSH.
// It also demonstrates what can be achieved when combining the use of the Go SDK for Cells
// with the powerful Cobra framework to implement CLI client applications for Cells.
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	cellsSdk "github.com/pydio/cells-sdk-go/v4"

	"github.com/pydio/cells-client/v4/common"
	"github.com/pydio/cells-client/v4/rest"
)

const (
	EnvDisplayHiddenFlags = "CEC_DISPLAY_HIDDEN_FLAGS"
)

var (
	// These commands and respective children do not need an already configured environment.
	infoCommands = []string{
		"help", "config", "version", "completion", "oauth", "clear", "doc", "update", "token", "tools",
		// legacy
		"configure",
	}

	sdkClient *rest.SdkClient

	configFilePath string

	serverURL string
	token     string
	authType  string
	login     string
	password  string

	//proxyURL    string
	skipKeyring bool
	skipVerify  bool
	noCache     bool
)

// RootCmd is the parent of all commands defined in this package.
// It takes care of the pre-configuration of the default connection to the SDK in its PersistentPreRun phase.
var RootCmd = &cobra.Command{
	Use:                    os.Args[0],
	Short:                  "Connect to a Pydio Cells server using the command line",
	BashCompletionFunction: bashCompletionFunc,
	//Args:                   cobra.MinimumNArgs(1),
	Long: `
DESCRIPTION

  This command line client allows interacting with a Pydio Cells server via the command line. 
  It uses the Cells SDK for Go and the REST API under the hood.

  See the respective help pages of the various commands to get detailed explanation and some examples.

  *WARNING*: cec v4 only supports remote servers that are in v4 or newer. 

CONFIGURE

  For the very first run, use '` + os.Args[0] + ` config add' to begin the command-line based configuration wizard. 
  This will guide you through a quick procedure to get you up and ready in no time.

  Non-sensitive information are stored by default in a ` + common.DefaultConfigFileName + ` file under ` + rest.DefaultConfigDirPath() + `
  You can change this location by using the --config flag.
  Entered (or retrieved, in the case of OAuth2 procedure) credentials will be stored in your keyring.

  [Note]: if no keyring is found, all information are stored in clear text in the ` + common.DefaultConfigFileName + ` file, including sensitive bits.

ENVIRONMENT

  All the command flags documented below are mapped to their associated ENV var, using upper case and CEC_ prefix.

  For example:
    $ ` + os.Args[0] + ` ls --no-cache
  is equivalent to: 
    $ export CEC_NO_CACHE=true; ` + os.Args[0] + ` ls
   
  This is typically useful when using the Cells Client non-interactively on a server:
    $ export CEC_URL=https://files.example.com; export CEC_TOKEN=<Your Personal Access Token>; 
    $ ` + os.Args[0] + ` ls

`, PersistentPreRun: func(cmd *cobra.Command, args []string) {

		if len(os.Args) == 1 {
			return
		}

		logger, err := configureLogger(viper.GetString("log"))
		if err != nil {
			log.Fatalf("could not initialize logger: %v, aborting.", err)
		}
		defer func(logger *zap.Logger) {
			_ = logger.Sync()
		}(logger)

		needSetup := true
		for _, skip := range infoCommands { // info commands do not require a configured env.
			// We only check this at the 2 first command "levels" for the time being
			if os.Args[1] == skip || (len(os.Args) > 2 && os.Args[2] == skip) {
				needSetup = false
				break
			}
		}

		// We cannot initialise config path before:
		// default value is built upon the AppName that can be overwritten by an extending app
		parPath := viper.GetString("config")
		if parPath == "" {
			parPath = rest.DefaultConfigDirPath()
		}
		configFilePath = filepath.Join(parPath, common.DefaultConfigFileName)

		tmpURLStr := viper.GetString("url")
		if tmpURLStr != "" {
			// Also sanitize the passed URL
			serverURL, err = rest.CleanURL(tmpURLStr)
			if err != nil {
				rest.Log.Fatalf("server URL %s seems to be unvalid, please double check and adapt. Cause: %s", tmpURLStr, err.Error())
			}
		}

		token = viper.GetString("token")
		login = viper.GetString("login")
		password = viper.GetString("password")
		//proxyURL = viper.GetString("proxy-url")
		noCache = viper.GetBool("no-cache")
		skipKeyring = viper.GetBool("skip-keyring")
		skipVerify = viper.GetBool("skip-verify")

		// Tweak to support old flags
		if viper.GetBool("no_cache") {
			noCache = true
		}
		if viper.GetBool("skip_keyring") {
			skipKeyring = true
		}
		if viper.GetBool("skip_verify") {
			skipVerify = true
		}

		if needSetup {
			e := setUpEnvironment(cmd.Context())
			if e != nil {
				if !os.IsNotExist(e) {
					rest.Log.Fatalf("unexpected error during initialisation phase: %s", e.Error())
				}
				rest.Log.Fatalf("no configuration has been found, please make sure to run '%s config add' first", os.Args[0])
			}
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Usage()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if sdkClient != nil {
			sdkClient.Teardown()
		}
	},
}

func CurrentSdkClient() *rest.SdkClient {
	return sdkClient
}

// RegisterExtraOfflineCommand adds the passed commands to the list of commands that skip the verifications
// that ensure that we have a valid connection to a distant Cells server defined.
func RegisterExtraOfflineCommand(commands ...string) {
	infoCommands = append(infoCommands, commands...)
}

func init() {
	handleLegacyParams()
	viper.SetEnvPrefix(common.EnvPrefix)
	viper.AutomaticEnv()

	flags := RootCmd.PersistentFlags()

	flags.String("log", "info", "change log level (default: info)")

	flags.String("config", "", fmt.Sprintf("Location of Cells Client's config files (default: %s)", rest.DefaultConfigFilePath()))
	flags.StringP("url", "u", "", "The full URL of the target server")
	flags.StringP("token", "t", "", "A valid Personal Access Token (PAT)")
	flags.String("login", "", "The user login, for Client auth only")
	flags.String("password", "", "The user password, for Client auth only")

	//flags.String("proxy-url", "", "Define a custom proxy for outbound requests")
	flags.Bool("skip-verify", false, "By default the Cells Client verifies the validity of TLS certificates for each communication. This option skips TLS certificate verification")
	flags.Bool("skip-keyring", false, "Explicitly tell the tool to *NOT* try to use a keyring, even if present. Warning: sensitive information will be stored in clear text")
	flags.Bool("no-cache", false, "Force token refresh at each call. This might slow down scripts with many calls")

	// Keep backward compatibility until v5 for old flag names
	replaceMap := map[string]string{}
	// Does not work as expected -> skipped
	//replaceMap := map[string]string{
	//	"skip_verify":  "skip-verify",
	//	"skip_keyring": "skip-keyring",
	//	"no_cache":     "no-cache",
	//}

	flags.Bool("skip_verify", false, "Deprecated, rather use skip-verify flag")
	flags.Bool("skip_keyring", false, "Deprecated, rather use skip-keyring flag")
	flags.Bool("no_cache", false, "Deprecated, rather use no-cache flag")

	if os.Getenv(EnvDisplayHiddenFlags) == "" {
		_ = flags.MarkHidden("skip_verify")
		_ = flags.MarkHidden("skip_keyring")
		_ = flags.MarkHidden("no_cache")
	}

	bindViperFlags(flags, replaceMap)
}

// setUpEnvironment configures the current runtime by setting the SDK Config that is used by child commands.
// It first tries to retrieve parameters via flags or environment variables. If it is not enough to define a valid connection,
// we check for a locally defined configuration file (that might also rely on local keyring to store sensitive info).
func setUpEnvironment(ctx context.Context) error {

	if configFilePath != "" {
		// override default location for the configuration file
		rest.SetConfigFilePath(configFilePath)
	}

	// Try first to establish context using flag or ENV vars
	c := getCecConfigFromEnv()

	var err error
	// Fallback to latest active registered account
	if c.SdkConfig == nil {
		_, err = os.ReadFile(configFilePath)
		if err != nil {
			return err
		}

		cl, err := rest.GetConfigList()
		if err != nil {
			return err
		}

		activeConfig, err := cl.GetActiveConfig(ctx)
		if err != nil {
			return err
		}
		c = activeConfig
	}

	// Set the user agent
	if c.CustomHeaders == nil {
		c.CustomHeaders = map[string]string{cellsSdk.UserAgentKey: rest.UserAgent()}
	} else {
		c.CustomHeaders[cellsSdk.UserAgentKey] = rest.UserAgent()
	}

	// Initialize an SDK Client
	sdkClient, err = rest.NewSdkClient(ctx, c)
	if err != nil {
		log.Fatal(err)
	}
	sdkClient.Setup(ctx)

	return nil
}

func configureLogger(logLevel string) (*zap.Logger, error) {
	level := zapcore.InfoLevel
	switch logLevel {
	case "debug", "DEBUG":
		level = zapcore.DebugLevel
	case "warn", "WARN":
		level = zapcore.WarnLevel
	case "error", "ERROR":
		level = zapcore.ErrorLevel
	}
	return rest.SetLogger(level), nil
}

// bindViperFlags visits all flags in FlagSet and bind their key to the corresponding viper variable.
func bindViperFlags(flags *pflag.FlagSet, replaceKeys map[string]string) {
	flags.VisitAll(func(flag *pflag.Flag) {
		key := flag.Name
		if replace, ok := replaceKeys[flag.Name]; ok {
			key = replace
		}
		if err := viper.BindPFlag(key, flag); err != nil {
			fmt.Printf("=== WARN: could not bind flag with key %s, cause:  %s ", key, err.Error())
		}
		if err := viper.BindEnv(key, getEnvVarName(key)); err != nil {
			fmt.Printf("=== WARN: could not bind flag with env var name: %s, cause:  %s\n", key, err.Error())
		}
	})
}

func getEnvVarName(flagName string) string {
	return fmt.Sprintf("%s_%s", common.EnvPrefix, strings.ToUpper(strings.ReplaceAll(flagName, "-", "_")))
}

// getCecConfigFromEnv first check if a valid connection has been configured with flags and/or ENV var
// **before** it even tries to retrieve info for the local file configuration.
// Also note that if both Token and User/Password are defined, we rather use the token for authentication.
func getCecConfigFromEnv() *rest.CecConfig {

	// Flags and env variable have been managed by viper => we can rely on local variable
	cecConfig := new(rest.CecConfig)
	sdkConfig := new(cellsSdk.SdkConfig)
	validConfViaContext := false

	if len(serverURL) > 0 {
		if len(token) > 0 { // PAT auth
			authType = cellsSdk.AuthTypePat
			sdkConfig.IdToken = token
			validConfViaContext = true
		} else if len(login) > 0 && len(password) > 0 { // client auth
			authType = cellsSdk.AuthTypeClientAuth
			sdkConfig.Password = password
			sdkConfig.User = login
			validConfViaContext = true
		}
	}

	if !validConfViaContext {
		return cecConfig
	}

	sdkConfig.Url = serverURL
	sdkConfig.SkipVerify = skipVerify
	sdkConfig.UseTokenCache = !noCache
	sdkConfig.AuthType = authType

	cecConfig.SdkConfig = sdkConfig
	cecConfig.SkipKeyring = skipKeyring

	return cecConfig
}

// handleLegacyParams manages backward compatibility for ENV variables and flags.
func handleLegacyParams() {

	prefOld := "CELLS_CLIENT_TARGET_"

	for _, pair := range os.Environ() {
		if strings.HasPrefix(pair, prefOld) {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 && parts[1] != "" {
				switch parts[0] {
				case "CELLS_CLIENT_TARGET_URL":
					_ = os.Setenv("CEC_URL", parts[1])
				case "CELLS_CLIENT_TARGET_CLIENT_KEY", "CELLS_CLIENT_TARGET_CLIENT_SECRET":
					log.Printf("[WARNING] %s is not used anymore. Double check your configuration", parts[0])
				case "CELLS_CLIENT_TARGET_USER_LOGIN":
					_ = os.Setenv("CEC_LOGIN", parts[1])
				case "CELLS_CLIENT_TARGET_USER_PWD":
					_ = os.Setenv("CEC_PASSWORD", parts[1])
				case "CELLS_CLIENT_TARGET_SKIP_VERIFY":
					_ = os.Setenv("CEC_SKIP_VERIFY", parts[1])
				}
			}
		}
	}
}
