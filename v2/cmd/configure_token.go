package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
)

var configurePersonalAccessTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Configure Authentication using a Personal Access Token",
	Long: `
DESCRIPTION

  Configure your Cells Client to connect to your distant server using a Personal Acces Token.
  A token can be generated with the 'token' command for a given user on the server side (not in this client),
  see 'cells admin user token --help' for further details.

  Please beware that the Personal Access Token will be stored in clear text if you do not have a **correctly configured and running** keyring on your client machine.

  This is the preferred way to handle authentication between Cells and your client.
`,
	RunE: func(cmd *cobra.Command, args []string) error {

		//var err error
		var p promptui.Prompt
		newConf := &rest.CecConfig{
			AuthType:    common.PatType,
			SkipKeyring: true,
		}

		cl, err := rest.GetConfigList()
		if errors.Is(err, os.ErrNotExist) {
			cl = &rest.ConfigList{Configs: map[string]*rest.CecConfig{}}
		} else {
			if err != nil {
				return err
			}
		}

		// non interactive
		if token != "" && serverURL != "" {
			newConf.IdToken = token
			newConf.Url = serverURL
		} else {

			// interactive
			p = promptui.Prompt{Label: "Server URL", Validate: rest.ValidURL}
			newConf.Url, err = p.Run()
			if err != nil {
				if errors.Is(err, promptui.ErrInterrupt) {
					return fmt.Errorf("operation aborted by user")
				}
				return fmt.Errorf("%s URL is not valid %s", promptui.IconBad, err.Error())
			}
			newConf.Url, err = rest.CleanURL(newConf.Url)
			if err != nil {
				return fmt.Errorf("%s %s", promptui.IconBad, err.Error())
			}

			p = promptui.Prompt{Label: "Token"}
			newConf.IdToken, err = p.Run()
			if err != nil {
				if errors.Is(err, promptui.ErrInterrupt) {
					return fmt.Errorf("operation aborted by user")
				}
				return err
			}
		}

		label := createLabel(newConf)
		if err := cl.Add(label, newConf); err != nil {
			return err
		}

		if err := cl.SaveConfigFile(); err != nil {
			return err
		}
		cmd.Println("config saved under label ", label)
		return nil
	},
}

func createLabel(c *rest.CecConfig) string {
	rest.DefaultConfig = c
	uname, e := rest.RetrieveCurrentSessionLogin()
	if e != nil {
		uname = "username_not_found"
	}

	var port string
	u, _ := url.Parse(c.Url)
	port = u.Port()
	if port == "" {
		switch u.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}

	return fmt.Sprintf("%s-%s:%s", uname, u.Host, port)
}

func init() {
	configureCmd.AddCommand(configurePersonalAccessTokenCmd)
}
