package cmd

import (
	"fmt"
	"log"
	"net/url"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

var configureClientAuthCmd = &cobra.Command{
	Use:   "client-auth",
	Short: "Connect to the server using Client Credentials",
	Long: `
DESCRIPTION

  Configure your Cells Client to connect to your distant server using Client Credentials.
  Note that this procedure is less secure than the other ones (using OAuth2 or a Personal Access Token).

  Please also beware that the password will be stored in clear text if you do not have a **correctly configured and running** keyring on your client machine.

USAGE

  This command launches an interactive process that gather necessary information to configure your connection to a running Cells server.
  You must provide a valid login and password for a user with enough permissions to achieve what you want on the server.

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

		err = rest.SaveConfig(newConf)
		if err != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration, cause: " + err.Error())
		}
	},
}

func interactive(newConf *rest.CecConfig) error {

	var e error

	// PROMPT URL
	p := promptui.Prompt{Label: "Server Address (provide a valid URL)", Validate: rest.ValidURL}
	if newConf.Url, e = p.Run(); e != nil {
		return e
	}

	newConf.Url, e = rest.CleanURL(newConf.Url)
	if e != nil {
		return e
	}

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
	
	return nil
}

func nonInteractive(conf *rest.CecConfig) error {

	conf.Url = serverURL
	conf.User = login
	conf.Password = password
	conf.SkipVerify = skipVerify

	// Insure values are legit
	if err := rest.ValidURL(conf.Url); err != nil {
		return fmt.Errorf("URL %s is not valid: %s", conf.Url, err.Error())
	}

	// Test a simple ping with this config before saving
	rest.DefaultConfig = conf
	if _, _, e := rest.GetApiClient(); e != nil {
		return fmt.Errorf("Could not connect to newly configured server, cause: %s", e.Error())
	}

	return nil
}

func init() {
	configureCmd.AddCommand(configureClientAuthCmd)
}
