package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/micro/go-log"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

const authTypeClientAuth = "client-auth"

// var (
// 	configHost       string
// 	configUser       string
// 	configPwd        string
// 	configSkipVerify bool
// )

var configureClientAuthCmd = &cobra.Command{
	Use:   authTypeClientAuth,
	Short: "Connect to the server directly using the Client Credentials",
	Long: `
Launch an interactive process to gather necessary client information to configure a connection to a running Pydio Cells server instance.

You must use a valid userName/password with enough permissions to achieve what you want on the server.

Please beware that this sensitive information will be stored in clear text if you do not have a **correctly configured and running** keyring on your client machine.

You can also go through the whole process in a non-interactive manner by using the provided flags.
`,
	Run: func(cm *cobra.Command, args []string) {

		var err error
		newConf := &rest.CecConfig{
			SkipKeyring: skipKeyring,
			AuthType:    common.ClientAuthType,
		}

		if notEmpty(serverURL) == nil && notEmpty(login) == nil && notEmpty(password) == nil {
			err = nonInteractive(newConf)
		} else {
			err = interactive(newConf)
		}
		if err != nil {
			if err == promptui.ErrInterrupt {
				fmt.Println("Operation aborted by User")
				return
			}
			log.Fatal(err)
		}

		err = saveConfig(newConf)
		if err != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration file! " + err.Error())
		} else {
			fmt.Printf("%s Configuration saved, you can now use the client to interract with %s.\n", promptui.IconGood, newConf.Url)
		}
	},
}

func interactive(newConf *rest.CecConfig) error {

	var e error

	// PROMPT URL
	p := promptui.Prompt{Label: "Server Address (provide a valid URL)", Validate: validUrl}
	if newConf.Url, e = p.Run(); e != nil {
		return e
	}
	newConf.Url = strings.Trim(newConf.Url, " ")

	u, e := url.Parse(newConf.Url)
	if e != nil {
		return e
	}

	if u.Scheme == "https" {
		// PROMPT SKIP VERIFY
		p2 := promptui.Select{Label: "Skip SSL Verification? (not recommended)", Items: []string{"No", "Yes"}}
		if _, y, e := p2.Run(); y == "Yes" && e == nil {
			newConf.SkipVerify = true
		}
	}

	// PROMPT LOGIN
	p = promptui.Prompt{
		Label:    "User Login",
		Validate: notEmpty,
	}
	if newConf.User, e = p.Run(); e != nil {
		return e
	}

	// PROMPT PASSWORD
	p = promptui.Prompt{Label: "User Password", Mask: '*', Validate: notEmpty}
	if newConf.Password, e = p.Run(); e != nil {
		return e
	}

	// Test a simple PING with this config before saving
	fmt.Println(promptui.IconWarn + " Testing this configuration before saving")
	rest.DefaultConfig = newConf
	if _, _, e := rest.GetApiClient(); e != nil {
		fmt.Println("\r" + promptui.IconBad + " Could not connect to server, please recheck your configuration")
		fmt.Println("Cause: " + e.Error())
		return fmt.Errorf("test connection failed")
	}
	fmt.Println("\r" + promptui.IconGood + " Successfully logged to server")
	return nil
}

func nonInteractive(conf *rest.CecConfig) error {

	conf.Url = serverURL
	conf.User = login
	conf.Password = password
	conf.SkipVerify = skipVerify

	// Insure values are legal
	if err := validUrl(conf.Url); err != nil {
		return fmt.Errorf("URL %s is not valid: %s", conf.Url, err.Error())
	}

	// Test a simple ping with this config before saving
	rest.DefaultConfig = conf
	if _, _, e := rest.GetApiClient(); e != nil {
		return fmt.Errorf("Could not connect to newly configured server failed, cause: %s", e.Error())
	}

	return nil
}

func validUrl(input string) error {
	// Warning: trim must also be performed when retrieving the final value.
	// Here we only validate that the trimmed input is valid, but do not modify it.
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return fmt.Errorf("Field cannot be empty")
	}
	u, e := url.Parse(input)
	if e != nil || u == nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("Please, provide a valid URL")
	}
	return nil
}

func notEmpty(input string) error {
	if len(input) == 0 {
		return fmt.Errorf("Field cannot be empty")
	}
	return nil
}

func init() {

	// flags := configureClientAuthCmd.PersistentFlags()

	// flags.StringVarP(&configHost, "url", "u", "", "HTTP URL to server")
	// flags.StringVarP(&configUser, "login", "l", "", "User login")
	// flags.StringVarP(&configPwd, "password", "p", "", "User password")
	// flags.BoolVar(&configSkipVerify, "skipVerify", false, "Skip SSL certificate verification (not recommended)")

	configureCmd.AddCommand(configureClientAuthCmd)
}
