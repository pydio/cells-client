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
	"os"
	"strconv"
	"strings"

	"github.com/ory/viper"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/pydio/cells-client/v2/rest"
)

const (
	// EnvPrefix represents the prefix used to insure we have a reserved namespacce for cec specific ENV vars.
	EnvPrefix = "CEC"
	// EnvPrefixOld represents the legacy prefix for environment variables, kept for backward compat.
	EnvPrefixOld = "CELLS_CLIENT"

	unconfiguredMsg = "unconfigured"
)

var (
	configFile string

	// These commands and respective children do not need an already configured environment.
	infoCommands = []string{"help", "configure", "version", "completion", "oauth", "clear", "doc", "update", "token", "--help"}
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

		needSetup := len(os.Args) > 1 // no args

		for _, skip := range infoCommands { // reserved info commands
			if os.Args[1] == skip {
				needSetup = false
				break
			}
		}

		if needSetup {
			e := setUpEnvironment(configFile)
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
	initEnvPrefixes()
	viper.SetEnvPrefix(EnvPrefix)
	viper.AutomaticEnv()

	flags := RootCmd.PersistentFlags()
	flags.StringVarP(&configFile, "config", "c", "", "Path to the configuration file")

	bindViperFlags(flags, map[string]string{})
}

func initEnvPrefixes() {
	prefOld := strings.ToUpper(EnvPrefixOld) + "_"
	prefNew := strings.ToUpper(EnvPrefix) + "_"
	//log.Println("... iterating over ENV vars")
	for _, pair := range os.Environ() {
		//log.Printf("- %s \n", pair)
		if strings.HasPrefix(pair, prefOld) {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 && parts[1] != "" {
				os.Setenv(prefNew+strings.TrimPrefix(parts[0], prefOld), parts[1])
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

// SetUpEnvironment retrieves parameters and stores them in the DefaultConfig of the SDK.
// It also puts the sensitive bits in the server's keyring if one is present.
// Note the precedence order (for each start of the app):
//  1) flags
// 	2) environment variables,
//  3) config files whose path is passed as argument of the start command
//  4) local config file (that are generated at first start with one of the 2 options above OR by calling the configure command.
func setUpEnvironment(confPath string) error {
	// Use a config file
	if confPath != "" {
		rest.SetConfigFilePath(confPath)
	}

	// Get config params from environment variables
	c, err := getSdkConfigFromEnv()
	if err != nil {
		return err
	}

	if c.Url == "" {

		confPath = rest.GetConfigFilePath()
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

func getSdkConfigFromEnv() (rest.CecConfig, error) {

	// var c CecConfig
	c := new(rest.CecConfig)

	// Check presence of environment variables
	url := os.Getenv(rest.KeyURL)
	clientKey := os.Getenv(rest.KeyClientKey)
	clientSecret := os.Getenv(rest.KeyClientSecret)
	user := os.Getenv(rest.KeyUser)
	password := os.Getenv(rest.KeyPassword)
	skipVerifyStr := os.Getenv(rest.KeySkipVerify)
	if skipVerifyStr == "" {
		skipVerifyStr = "false"
	}
	skipVerify, err := strconv.ParseBool(skipVerifyStr)
	if err != nil {
		return *c, err
	}

	// Client Key and Client Secret are not used anymore
	// if !(len(url) > 0 && len(clientKey) > 0 && len(clientSecret) > 0 && len(user) > 0 && len(password) > 0) {
	if !(len(url) > 0 && len(user) > 0 && len(password) > 0) {
		return *c, nil
	}

	c.Url = url
	c.ClientKey = clientKey
	c.ClientSecret = clientSecret
	c.User = user
	c.Password = password
	c.SkipVerify = skipVerify

	// Note: this cannot be set via env variable. Enhance?
	c.UseTokenCache = true

	return *c, nil
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
