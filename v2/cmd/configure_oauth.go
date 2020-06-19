package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/manifoldco/promptui"
	"github.com/micro/go-log"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"

	cells_sdk "github.com/pydio/cells-sdk-go"

	"github.com/pydio/cells-client/v2/rest"
)

const authTypeOAuth = "oauth"

var (
	oAuthUrl        string
	oAuthIdToken    string
	oAuthSkipVerify bool

	callbackPort = 3000
)

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
	Use:   authTypeOAuth,
	Short: "User OAuth2 to login to server",
	Long:  `Configure Authentication using OAuth2`,
	Run: func(cm *cobra.Command, args []string) {

		var err error
		newConf := &cells_sdk.SdkConfig{}

		if oAuthUrl != "" && oAuthIdToken != "" {
			err = oAuthNonInteractive(newConf)
		} else {
			err = oAuthInteractive(newConf)
		}
		if err != nil {
			log.Fatal(err)
		}

		// Now save config!
		if !skipKeyring {
			if err := rest.ConfigToKeyring(newConf); err != nil {
				fmt.Println(promptui.IconWarn + " Cannot save token in keyring! " + err.Error())
			}
		}
		filePath := rest.DefaultConfigFilePath()
		data, _ := json.Marshal(newConf)
		err = ioutil.WriteFile(filePath, data, 0644)
		if err != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration file! " + err.Error())
		} else {
			fmt.Printf("%s Configuration saved, you can now use the client to interract with %s.\n", promptui.IconGood, newConf.Url)
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

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func oAuthInteractive(newConf *cells_sdk.SdkConfig) error {
	var e error
	// PROMPT URL
	p := promptui.Prompt{
		Label:    "Server Address (provide a valid URL)",
		Validate: validUrl,
		Default:  "",
	}

	if newConf.Url, e = p.Run(); e != nil {
		return e
	} else {
		newConf.Url = strings.Trim(newConf.Url, " ")
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

	newConf.ClientKey = "cells-client"
	pE := promptui.Select{Label: "Do you want to edit OAuth client data (defaults generally work)?", Items: []string{"Use defaults", "Edit OAuth client"}}
	if _, v, e := pE.Run(); e == nil && v != "Use defaults" {
		// PROMPT CLIENT ID
		p = promptui.Prompt{
			Label:     "OAuth APP ID (found in your server pydio.json)",
			Validate:  notEmpty,
			Default:   "cells-client",
			AllowEdit: true,
		}
		if newConf.ClientKey, e = p.Run(); e != nil {
			return e
		}
		p = promptui.Prompt{Label: "OAuth APP Secret (leave empty for a public client)", Default: "", Mask: '*'}
		newConf.ClientSecret, _ = p.Run()
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
	state := RandString(16)
	directUrl, callbackUrl, err := rest.OAuthPrepareUrl(newConf.Url, newConf.ClientKey, newConf.ClientSecret, state, openBrowser)
	if err != nil {
		log.Fatal(err)
	}
	if openBrowser {
		fmt.Println("Opening URL", directUrl)
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
	if err := rest.OAuthExchangeCode(newConf, returnCode, callbackUrl); err != nil {
		log.Fatal(err)
	}
	fmt.Println(promptui.IconGood + " Successfully Received Token!")

	// Test a simple PING with this config before saving!
	fmt.Println(promptui.IconWarn + " Testing this configuration before saving")
	rest.DefaultConfig = newConf
	if _, _, e := rest.GetApiClient(); e != nil {
		fmt.Println("\r" + promptui.IconBad + " Could not connect to server, please recheck your configuration")
		fmt.Printf("Id_token: [%s]\n", newConf.IdToken)

		fmt.Println("Cause: " + e.Error())
		return fmt.Errorf("test connection failed")
	}
	fmt.Println("\r" + promptui.IconGood + fmt.Sprintf(" Successfully logged to server, token will be refreshed at %v", time.Unix(int64(newConf.TokenExpiresAt), 0)))
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

func oAuthNonInteractive(conf *cells_sdk.SdkConfig) error {

	conf.Url = oAuthUrl
	conf.IdToken = oAuthIdToken
	conf.SkipVerify = configSkipVerify

	// Insure values are legal
	if err := validUrl(conf.Url); err != nil {
		return fmt.Errorf("URL %s is not valid: %s", conf.Url, err.Error())
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

	flags.StringVarP(&oAuthUrl, "url", "u", "", "HTTP URL to server")
	flags.StringVarP(&oAuthIdToken, "idToken", "t", "", "Valid IdToken")
	flags.BoolVar(&oAuthSkipVerify, "skipVerify", false, "Skip SSL certificate verification (not recommended)")

	configureCmd.AddCommand(configureOAuthCmd)
}
