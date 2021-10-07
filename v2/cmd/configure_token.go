package cmd

import (
	"errors"
	"fmt"
	"log"
	"strings"

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
	Run: func(cmd *cobra.Command, args []string) {

		newConf := &rest.CecConfig{
			AuthType:    common.PatType,
			SkipKeyring: skipKeyring,
		}

		var err error
		if token != "" && serverURL != "" {
			// non interactive
			newConf.IdToken = token
			newConf.Url = serverURL

		} else {
			// interactive

			p := promptui.Prompt{Label: "Server URL", Validate: rest.ValidURL}
			newConf.Url, err = p.Run()
			if err != nil {
				if errors.Is(err, promptui.ErrInterrupt) {
					log.Fatalf("operation aborted by user")
				}
				log.Fatalf("%s URL is not valid %s", promptui.IconBad, err.Error())
			}
			newConf.Url, err = rest.CleanURL(newConf.Url)
			if err != nil {
				log.Fatalf("%s %s", promptui.IconBad, err.Error())
			}

			p = promptui.Prompt{Label: "Token", Validate: func(s string) error {
				s = strings.TrimSpace(s)
				if len(s) == 0 {
					return fmt.Errorf("field cannot be empty")
				}
				return nil
			}}
			newConf.IdToken, err = p.Run()
			if err != nil {
				if errors.Is(err, promptui.ErrInterrupt) {
					log.Fatalf("operation aborted by user")
				}
				log.Fatalf(err.Error())
			}
		}

		err = PersistConfig(newConf)
		if err != nil {
			log.Fatalf(err.Error())
		}
	},
}

func init() {
	configureCmd.AddCommand(configurePersonalAccessTokenCmd)
}
