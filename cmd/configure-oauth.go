package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/syncthing/syncthing/lib/rand"

	"github.com/skratchdot/open-golang/open"

	"github.com/manifoldco/promptui"
	"github.com/micro/go-log"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/rest"
	cells_sdk "github.com/pydio/cells-sdk-go"
)

var (
	oAuthUrl        string
	oAuthIdToken    string
	oAuthSkipVerify bool
)

type oAuthHandler struct {
	// Input
	done  chan bool
	state string
	// Output
	code string
	err  error
}

func (o *oAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer close(o.done)
	values := r.URL.Query()
	//fmt.Println("Received values", values)
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
	w.Write([]byte(`<p>You can now close this window and go back to your shell!</p><script type="text/javascript">window.close();</script>`))
}

func oAuthInteractive(newConf *cells_sdk.SdkConfig) error {
	var e error
	// PROMPT URL
	p := promptui.Prompt{Label: "Server Address (provide a valid URL)", Validate: validUrl, Default: "https://local.pydio:8080"}
	if newConf.Url, e = p.Run(); e != nil {
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

	// PROMPT CLIENT ID
	p = promptui.Prompt{Label: "OAuth Static client ID (found in your server pydio.json)", Validate: notEmpty, Default: "cells-sync"}
	if newConf.ClientKey, e = p.Run(); e != nil {
		return e
	}

	openBrowser := true
	p3 := promptui.Select{Label: "Can you open a browser on this computer? If not, you will make the authentication process by copy/pasting", Items: []string{"Yes", "No"}}
	if _, v, e := p3.Run(); e == nil && v == "No" {
		openBrowser = false
	}

	var returnCode string
	var redirUri string
	authU := *u
	state := rand.String(16)
	authU.Path = "/oidc/oauth2/auth"
	values := url.Values{}
	values.Add("response_type", "code")
	values.Add("client_id", newConf.ClientKey)
	values.Add("state", state)
	if openBrowser {
		redirUri = "http://localhost:3000/servers/callback"
		values.Add("redirect_uri", redirUri)
		authU.RawQuery = values.Encode()
		fmt.Println("Opening URL", authU.String())
		go open.Run(authU.String())

		h := &oAuthHandler{
			done:  make(chan bool),
			state: state,
		}
		srv := &http.Server{Addr: ":3000"}
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
		redirUri = newConf.Url + "/oauth2/oob"
		values.Add("redirect_uri", redirUri)
		authU.RawQuery = values.Encode()
		fmt.Println("Please copy and paste this URL in a browser", authU.String())
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

	fmt.Println("Now exchanging the code for a valid IdToken")
	tokenU := *u
	tokenU.Path = "/oidc/oauth2/token"
	values = url.Values{}
	values.Add("grant_type", "authorization_code")
	values.Add("code", returnCode)
	values.Add("redirect_uri", redirUri)
	values.Add("client_id", newConf.ClientKey)
	resp, err := http.Post(tokenU.String(), "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	b, _ := ioutil.ReadAll(resp.Body)
	type respData struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		IdToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
	}
	var r respData
	if err := json.Unmarshal(b, &r); err != nil {
		log.Fatal("Cannot unmarshall token response")
	}
	fmt.Println("Successfully Received Token!")
	newConf.IdToken = r.AccessToken
	newConf.RefreshToken = r.RefreshToken
	newConf.TokenExpiresAt = int(time.Now().Unix()) + r.ExpiresIn

	// Test a simple PING with this config before saving!
	fmt.Println(promptui.IconWarn + " Testing this configuration before saving")
	rest.DefaultConfig = newConf
	if _, _, e := rest.GetApiClient(); e != nil {
		fmt.Println("\r" + promptui.IconBad + " Could not connect to server, please recheck your configuration")
		fmt.Println("   Error was " + e.Error())
		return fmt.Errorf("test connection failed")
	}
	fmt.Println("\r" + promptui.IconGood + " Successfully logged to server")
	return nil
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

var configureOAuthCmd = &cobra.Command{
	Use:  "oauth",
	Long: `Configure Authentication using OAuth`,
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
		filePath := rest.DefaultConfigFilePath()
		data, _ := json.Marshal(newConf)
		err = ioutil.WriteFile(filePath, data, 0755)
		if err != nil {
			fmt.Println(promptui.IconBad + " Cannot save configuration file! " + err.Error())
		} else {
			fmt.Printf("%s Configuration saved, you can now use the client to interract with %s.\n", promptui.IconGood, newConf.Url)
		}
	},
}

func init() {

	flags := configureOAuthCmd.PersistentFlags()

	flags.StringVarP(&oAuthUrl, "url", "u", "", "HTTP URL to server")
	flags.StringVarP(&oAuthIdToken, "idToken", "t", "", "Valid IdToken")
	flags.BoolVar(&oAuthSkipVerify, "skipVerify", false, "Skip SSL certificate verification (not recommended)")

	RootCmd.AddCommand(configureOAuthCmd)
}
