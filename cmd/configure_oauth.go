package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gookit/color"
	"github.com/manifoldco/promptui"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	sdk_rest "github.com/pydio/cells-sdk-go/v4/transport/rest"

	"github.com/pydio/cells-client/v4/common"
	"github.com/pydio/cells-client/v4/rest"
)

var oauthIDToken string

// This cannot be changed on the client side: the callback URL, including this port, must be registered on the server side.
const callbackPort = 3000

type oAuthHandler struct {
	// Input
	done   chan bool
	closed bool
	state  string
	// Output
	code string
	err  error
}

var configureOAuthCmd = &cobra.Command{
	Use:   "oauth",
	Short: "Use OAuth2 credential flow to login to the server",
	Long: `
DESCRIPTION

  Configure your Cells Client to connect to your distant server using OAuth2 standard procedures.

  Please beware that the retrieved ID and refresh tokens will be stored in clear text if you do not have a **correctly configured and running** keyring on your client machine.

USAGE

  This command launches an interactive process that gather necessary information.
  If you are on a workstation with a browser, you are then redirected to your Cells' web UI to authenticate.
  Otherwise, we provide you with a link that will help you terminate the procedure with 2 copy/pastes.
  
  If you are quick enough, (or if the default JWT token duration is long enough), 
  you can also initialise this configuration by providing an ID token that you have retrieved using an alternative procedure,
  and go through the configuration process in a non-interactive manner by using the provided flags.
`,
	Run: func(cm *cobra.Command, args []string) {

		newConf := rest.DefaultCecConfig()
		newConf.AuthType = common.OAuthType
		newConf.SkipKeyring = skipKeyring

		var err error
		if serverURL != "" && oauthIDToken != "" {
			err = oAuthNonInteractive(newConf)
		} else {
			err = oAuthInteractive(newConf)
		}
		if err != nil {
			if err == promptui.ErrInterrupt {
				log.Fatal("operation aborted by user")
			}
			log.Fatal(err.Error())
		}
		err = persistConfig(newConf)
		if err != nil {
			log.Fatal(err.Error())
		}
	},
}

func (o *oAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if !o.closed {
			o.closed = true
			close(o.done)
		}
	}()
	values := r.URL.Query()
	if values.Get("state") != o.state {
		o.err = fmt.Errorf("wrong state received")
		return
	}
	if values.Get("code") == "" {
		o.err = fmt.Errorf("empty code received")
		return
	}
	o.code = values.Get("code")
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
		<p style="display: flex;height: 100%;width: 100%;align-items: center;justify-content: center;font-family: sans-serif;color: #607D8B;font-size: 20px;">
			You can now close this window and go back to your shell!
		</p>
		<script type="text/javascript">window.close();</script>
	`))
}

func oAuthInteractive(newConf *rest.CecConfig) error {
	var e error
	// PROMPT URL
	p := promptui.Prompt{
		Label:    "Server Address (provide a valid URL)",
		Validate: rest.ValidURL,
		Default:  "",
	}

	newConf.Url, e = p.Run()
	if e != nil {
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

	openBrowser := true
	p3 := promptui.Select{Label: "Can you open a browser on this computer? If not, you will make the authentication process by copy/pasting", Items: []string{"Yes", "No"}}
	if _, v, e := p3.Run(); e == nil && v == "No" {
		openBrowser = false
	}

	// Check default port availability: Note that we do not offer option to change the port,
	// because it would also require impacting the registered client in the pydio.json
	// of the server that is not an acceptable option.
	avail := isPortAvailable(callbackPort, 10)
	if !avail {
		fmt.Printf("Warning: default port %d is not available on this machine, "+
			"you thus won't be able to automatically complete the auth code flow with the implicit callback URL."+
			"Please free this port or choose the copy/paste solution.\n", callbackPort)
		openBrowser = false
	}

	// Starting authentication process
	var returnCode string
	state := rest.RandString(16)
	directUrl, callbackUrl, err := sdk_rest.OAuthPrepareUrl(common.AppName, newConf.Url, state, openBrowser)
	if err != nil {
		log.Fatal(err)
	}
	if openBrowser {
		go open.Run(directUrl)
		h := &oAuthHandler{
			done:  make(chan bool),
			state: state,
		}
		srv := &http.Server{Addr: fmt.Sprintf(":%d", callbackPort)}
		srv.Handler = h
		go func() {
			<-h.done
			srv.Shutdown(context.Background())
		}()
		srv.ListenAndServe()
		if h.err != nil {
			log.Fatal("Could not correctly connect", h.err)
		}
		returnCode = h.code
	} else {
		col := color.FgLightRed.Render
		fmt.Println("Please copy and paste this URL in a browser", col(directUrl))
		var err error
		pr := promptui.Prompt{
			Label:    "Please Paste the code returned to you in the browser",
			Validate: notEmpty,
		}
		returnCode, err = pr.Run()
		if err != nil {
			log.Fatal("Could not read code!")
		}
	}

	fmt.Println(promptui.IconGood + " Now exchanging the code for a valid IdToken")
	if err := sdk_rest.OAuthExchangeCode(common.AppName, newConf.SdkConfig, returnCode, callbackUrl); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s Successfully Received Token. It will be refreshed at %v\n", promptui.IconGood, time.Unix(int64(newConf.TokenExpiresAt), 0))

	return nil
}

func isPortAvailable(port int, timeout int) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func oAuthNonInteractive(conf *rest.CecConfig) error {

	conf.Url = serverURL
	conf.IdToken = oauthIDToken
	conf.SkipVerify = skipVerify

	// Insure values are legal
	err := rest.ValidURL(conf.Url)
	if err != nil {
		return fmt.Errorf("URL %s is not valid: %s", conf.Url, err.Error())
	}

	conf.Url, err = rest.CleanURL(conf.Url)
	if err != nil {
		return err
	}

	// Test a simple PING with this config before saving!
	rest.DefaultConfig = conf
	if _, _, e := rest.GetApiClient(); e != nil {
		return fmt.Errorf("test connection to newly configured server failed")
	}

	return nil
}

func init() {
	flags := configureOAuthCmd.PersistentFlags()
	flags.StringVar(&oauthIDToken, "id_token", "", "A currently valid OAuth2 ID token, retrived via the OIDC credential flow")
	configureCmd.AddCommand(configureOAuthCmd)
	configAddCmd.AddCommand(configureOAuthCmd)
}
