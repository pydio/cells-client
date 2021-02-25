// Package cmd implements some basic use case to manage your files on your remote server
// via the  command line of your local workstation or any server you can access with SSH.
// It also demonstrates what can be achieved when combining the use of the Go SDK for Cells
// with the powerful Cobra framework to implement CLI client applications for Cells.
package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

const (
	// EnvPrefix represents the prefix used to insure we have a reserved namespacce for cec specific ENV vars.
	EnvPrefix = "CEC"
	// EnvPrefixOld represents the legacy prefix for environment variables, kept for backward compat.
	// EnvPrefixOld = "CELLS_CLIENT_TARGET"

	unconfiguredMsg = "unconfigured"
)

var (
	// These commands and respective children do not need an already configured environment.
	infoCommands = []string{"help", "configure", "version", "completion", "oauth", "clear", "doc", "update", "token", "--help"}

	configFilePath string

	serverURL string
	idToken   string
	authType  string
	login     string
	password  string

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
	Args:                   cobra.MinimumNArgs(1),
	Long: `
The Cells Client tool allows interacting with a Pydio Cells server instance directly via the command line. 
It uses the Cells SDK for Go and the REST API under the hood.

See the respective help pages of the various commands to get detailed explanation and some examples.

You should probably start with configuring your setup by running:
 ` + os.Args[0] + ` configure

This will guide you through a quick procedure to get you up and ready in no time.
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		needSetup := true

		for _, skip := range infoCommands { // info commands do not require a configured env.
			if os.Args[1] == skip {
				needSetup = false
				break
			}
		}

		// Manually bind to viper instead of flags.StringVar, flags.BoolVar, etc
		// => This is useful to ease implementation of retrocompatibility
		configFilePath = viper.GetString("config") + "/config.json"
		tmpURLStr := viper.GetString("url")
		// Clean URL string
		if tmpURLStr != "" {
			tmpURL, err := url.Parse(tmpURLStr)
			if err != nil {
				log.Fatalf("server URL %s seems to be unvalid, please double check and adapt. Cause: %s", tmpURLStr, err.Error())
			}
			serverURL = tmpURL.Scheme + "://" + tmpURL.Host
		}
		authType = viper.GetString("auth_type")
		idToken = viper.GetString("id_token")
		login = viper.GetString("login")
		password = viper.GetString("password")
		noCache = viper.GetBool("no_cache")
		skipKeyring = viper.GetBool("skip_keyring")
		skipVerify = viper.GetBool("skip_verify")

		//fmt.Println("[DEBUG] flags: ")
		//fmt.Printf("- configFilePath: %s\n", configFilePath)
		//fmt.Printf("- serverURL: %s\n", serverURL)
		//fmt.Printf("- authType: %s\n", authType)
		//fmt.Printf("- idToken: %s\n", idToken)
		//fmt.Printf("- login: %s\n", login)
		//fmt.Printf("- password: %s\n", password)
		//fmt.Printf("- noCache: %v\n", noCache)
		//fmt.Printf("- skipKeyring: %v\n", skipKeyring)
		//fmt.Printf("- skipVerify: %v\n", skipVerify)
		////log.Println("... iterating over ENV vars")
		//for _, pair := range os.Environ() {
		//	//log.Printf("- %s \n", pair)
		//	if strings.HasPrefix(pair, "CEC_") {
		//		parts := strings.Split(pair, "=")
		//		if len(parts) == 2 && parts[1] != "" {
		//			fmt.Printf("- %s : %s\n", parts[0], parts[1])
		//		}
		//	}
		//}

		if needSetup {
			e := setUpEnvironment()
			if e != nil {
				if e.Error() != unconfiguredMsg {
					log.Fatalf("unexpected error during initialisation phase: %s", e.Error())
				}
				// TODO Directly launch necessary configure command
				log.Fatalf("No configuration has been found, please make sure to run '%s configure' first.\n", os.Args[0])
			}
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	handleLegagyParams()
	viper.SetEnvPrefix(EnvPrefix)
	viper.AutomaticEnv()

	flags := RootCmd.PersistentFlags()

	dflt := rest.DefaultConfigDirPath()
	flags.String("config", dflt, fmt.Sprintf("Location of cells client config files (default %s)", dflt))

	flags.StringP("url", "u", "", "Full URL of the target server")
	flags.StringP("auth_type", "a", "", "Authorizaton mechanism used: Personnal Access Token (Default), OAuth2 flow or Client Credentials")
	flags.StringP("id_token", "t", "", "Valid IdToken")
	flags.StringP("login", "l", "", "User login")
	flags.StringP("password", "p", "", "User password")

	flags.Bool("skip_verify", false, "By default the Cells Client will verify the validity of TLS certificates for each communication. This option skips TLS certificate verification.")
	flags.Bool("skip_keyring", false, "Explicitly tell the tool to *NOT* try to use a keyring, even if present. Warning: sensitive information will be stored in clear text.")
	flags.Bool("no_cache", false, "Force token refresh at each call. This might slow down scripts with many calls.")

	bindViperFlags(flags, map[string]string{})
}

// SetUpEnvironment configures the current runtime by setting the SDK Config that is used by child commands.
// It first tries to retrieve parameters via flags or environment variables. If it is not enough to define a valid connection,
// we check for a locally defined configuration file (that might also relies on local keyring to store sensitive info).
func setUpEnvironment() error {

	if configFilePath != "" { // override default location for the configuration file
		rest.SetConfigFilePath(configFilePath)
	}

	// First Check if an environment is defined via the context (flags or ENV vars)
	c := getCecConfigFromEnv()

	if c.Url == "" {
		confPath := rest.GetConfigFilePath()
		if _, err := os.Stat(confPath); os.IsNotExist(err) {
			return fmt.Errorf(unconfiguredMsg)
		}

		s, err := ioutil.ReadFile(confPath)
		if err != nil {
			return err
		}
		err = json.Unmarshal(s, &c)
		if err != nil {
			return err
		}
		// Retrieves sensible info from the keyring if one is present
		rest.ConfigFromKeyring(&c)

		// Refresh token if required
		if refreshed, err := rest.RefreshIfRequired(&c); refreshed {
			if err != nil {
				log.Fatal("Could not refresh authentication token:", err)
			}
			// Copy config as IdToken will be cleared
			storeConfig := c
			if !c.SkipKeyring {
				rest.ConfigToKeyring(&storeConfig)
			}
			// Save config to renew TokenExpireAt
			confData, _ := json.MarshalIndent(&storeConfig, "", "\t")
			ioutil.WriteFile(confPath, confData, 0666)
		}
	}

	// Store current computed config in a public static singleton
	rest.DefaultConfig = &c

	return nil
}

// getCecConfigFromEnv first check if a valid connection has been configured with flags and/or ENV var
// **before** it even tries to retrieve info for the local file configuration.
func getCecConfigFromEnv() rest.CecConfig {

	// Flags and env variable have been managed by viper => we can rely on local variable
	c := new(rest.CecConfig)
	validConfViaContext := false

	if len(serverURL) > 0 {
		if len(login) > 0 && len(password) > 0 {
			authType = common.ClientAuthType
			c.Password = password
			c.User = login
			validConfViaContext = true

			// TODO do we want to enable OAuth from flags ?
			// } else if len(idToken) > 0 && len(refreshToken) {
			// 	authType = common.OAuthType
			// 	validConfViaContext = true

		} else if len(idToken) > 0 { // PAT auth
			authType = common.PatType
			c.IdToken = idToken
			validConfViaContext = true
		}
	}

	if !validConfViaContext {
		return *c
	}

	c.Url = serverURL
	c.AuthType = authType

	c.SkipVerify = skipVerify
	c.SkipKeyring = skipKeyring
	c.UseTokenCache = !noCache

	return *c
}

// handleLegagyParams manages backward compatibility for ENV variables and flags.
func handleLegagyParams() {

	prefOld := "CELLS_CLIENT_TARGET_"

	for _, pair := range os.Environ() {
		if strings.HasPrefix(pair, prefOld) {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 && parts[1] != "" {
				switch parts[0] {
				case "CELLS_CLIENT_TARGET_URL":
					os.Setenv("CEC_URL", parts[1])
				case "CELLS_CLIENT_TARGET_CLIENT_KEY", "CELLS_CLIENT_TARGET_CLIENT_SECRET":
					log.Printf("[WARNING] %s is not used anymore. Double check your configuration", parts[0])
				case "CELLS_CLIENT_TARGET_USER_LOGIN":
					os.Setenv("CEC_LOGIN", parts[1])
				case "CELLS_CLIENT_TARGET_USER_PWD":
					os.Setenv("CEC_PASSWORD", parts[1])
				case "CELLS_CLIENT_TARGET_SKIP_VERIFY":
					os.Setenv("CEC_SKIP_VERIFY", parts[1])
				}
			}
		}
	}
}

// bindViperFlags visits all flags in FlagSet and bind their key to the corresponding viper variable.
func bindViperFlags(flags *pflag.FlagSet, replaceKeys map[string]string) {
	flags.VisitAll(func(flag *pflag.Flag) {
		key := flag.Name
		if replace, ok := replaceKeys[flag.Name]; ok {
			key = replace
		}
		viper.BindPFlag(key, flag)
	})
}

var bashCompletionFunc = `__` + os.Args[0] + `_custom_func() {
  case ${last_command} in
  ` + os.Args[0] + `_mv | ` + os.Args[0] + `_cp | ` + os.Args[0] + `_rm | ` + os.Args[0] + `_ls)
    _path_completion
    return
    ;;
	` + os.Args[0] + `_storage_resync-ds)
    _datasources_completion
    return
    ;;
  ` + os.Args[0] + `_scp)
    _scp_path_completion
    return
    ;;
  *) ;;
  esac
}
_path_completion() {
  local lsopts cur dir
  cur="${COMP_WORDS[COMP_CWORD]}"
  dir="$(dirname "$cur" 2>/dev/null)"

  currentlength=${#cur}
  last_char=${cur:currentlength-1:1}

  if [[ $last_char == "/" ]] && [[ currentlength -gt 2 ]]; then
    dir=$cur
  elif [[ -z $dir ]]; then
    dir="/"
  elif [[ $dir == "." ]]; then
    dir="/"
  fi

  IFS=$'\n'
  lsopts="$(` + os.Args[0] + ` ls --raw $dir)"

  COMPREPLY=($(compgen -W "${lsopts[@]}" -- "$cur"))
  compopt -o nospace
  compopt -o filenames
}

_scp_path_completion() {
  local lsopts cur dir
  cur="${COMP_WORDS[COMP_CWORD]}"
	
  if [[ $cur != cells//* ]]; then
    return
  fi

  prefix="cells//"
  cur=${cur#$prefix}

  dir="$(dirname "$cur" 2>/dev/null)"
  currentlength=${#cur}
  last_char=${cur:currentlength-1:1}

  if [[ $last_char == "/" ]] && [[ currentlength -gt 2 ]]; then
      dir=$cur
  elif [[ -z $dir ]]; then
      dir="/"
  elif [[ $dir == "." ]]; then
      dir="/"
  fi

  IFS=$'\n'
  lsopts="$(` + os.Args[0] + ` ls --raw $dir)"

  COMPREPLY=($(compgen -P "$prefix" -W "${lsopts[@]}" -- "$cur"))
  #COMPREPLY=(${COMPREPLY[@]/#/"${prefix}"})
  compopt -o nospace
  compopt -o filenames
}

_datasources_completion() {
  local dsopts cur
  cur="${COMP_WORDS[COMP_CWORD]}"

  dsopts="$(` + os.Args[0] + ` storage list-datasources --raw)"
  COMPREPLY=($(compgen -W "${dsopts[@]}" -- "$cur"))
}
`
