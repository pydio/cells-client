package rest

// type tokenResponse struct {
// 	AccessToken  string `json:"access_token"`
// 	ExpiresIn    int    `json:"expires_in"`
// 	IdToken      string `json:"id_token"`
// 	RefreshToken string `json:"refresh_token"`
// 	// if the server returns an error, we will have this field so that we can check.
// 	StatusCode int `json:"status_code"`
// }

// // OAuthPrepareUrl makes a URL that can be opened in browser or copy/pasted by user
// func OAuthPrepareUrl(serverUrl, state string, browser bool) (redirectUrl string, callbackUrl string, e error) {

// 	authU, _ := url.Parse(serverUrl)
// 	authU.Path = "/oidc/oauth2/auth"
// 	values := url.Values{}
// 	values.Add("response_type", "code")
// 	values.Add("client_id", common.AppName)
// 	// if clientSecret != "" {
// 	// 	values.Add("client_secret", clientSecret)
// 	// }
// 	values.Add("scope", "openid email profile pydio offline")
// 	values.Add("state", state)
// 	if browser {
// 		callbackUrl = "http://localhost:3000/servers/callback"
// 	} else {
// 		callbackUrl = serverUrl + "/oauth2/oob"
// 	}
// 	values.Add("redirect_uri", callbackUrl)
// 	authU.RawQuery = values.Encode()

// 	redirectUrl = authU.String()

// 	return
// }

// // OAuthExchangeCode gets an OAuth code and retrieves an AccessToken/RefreshToken pair. It updates the passed Conf
// func OAuthExchangeCode(c *cells_sdk.SdkConfig, code, callbackUrl string) error {
// 	tokenU, _ := url.Parse(c.Url)
// 	tokenU.Path = "/oidc/oauth2/token"
// 	values := url.Values{}
// 	values.Add("grant_type", "authorization_code")
// 	values.Add("code", code)
// 	values.Add("redirect_uri", callbackUrl)
// 	values.Add("client_id", common.AppName)
// 	if c.SkipVerify {
// 		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
// 	}
// 	resp, err := http.Post(tokenU.String(), "application/x-www-form-urlencoded", strings.NewReader(values.Encode()))
// 	if err != nil {
// 		return err
// 	}
// 	b, _ := io.ReadAll(resp.Body)
// 	var r tokenResponse
// 	if err := json.Unmarshal(b, &r); err != nil {
// 		return err
// 	}

// 	if r.StatusCode > 299 {
// 		return fmt.Errorf("could not perfom authentication flow: response body %s", string(b))
// 	}

// 	c.IdToken = r.AccessToken
// 	c.RefreshToken = r.RefreshToken
// 	c.TokenExpiresAt = int(time.Now().Add(-5*time.Second).Unix()) + r.ExpiresIn

// 	return nil
// }

// // RefreshIfRequired refreshes the token inside the given conf if required.
// func RefreshIfRequired(cecConfig *CecConfig) (bool, error) {
// 	sdkConfig := cecConfig.SdkConfig

// 	// No token to refresh
// 	if sdkConfig.IdToken == "" || sdkConfig.RefreshToken == "" || sdkConfig.TokenExpiresAt == 0 {
// 		return false, nil
// 	}
// 	// Not yet expired, ignore
// 	if time.Unix(int64(sdkConfig.TokenExpiresAt), 0).After(time.Now().Add(60 * time.Second)) {
// 		return false, nil
// 	}
// 	data := url.Values{}
// 	data.Add("grant_type", "refresh_token")
// 	data.Add("client_id", common.AppName)
// 	data.Add("refresh_token", sdkConfig.RefreshToken)
// 	data.Add("scope", "openid email profile pydio offline")

// 	httpReq, err := http.NewRequest("POST", sdkConfig.Url+"/oidc/oauth2/token", strings.NewReader(data.Encode()))
// 	if err != nil {
// 		return true, err
// 	}
// 	httpReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
// 	httpReq.Header.Add("Cache-Control", "no-cache")

// 	client := http.DefaultClient
// 	if sdkConfig.SkipVerify {
// 		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
// 	}
// 	res, err := client.Do(httpReq)
// 	if err != nil {
// 		return true, err
// 	} else if res.StatusCode != 200 {
// 		bb, _ := io.ReadAll(res.Body)
// 		return true, fmt.Errorf("received status code %d - %s", res.StatusCode, string(bb))
// 	}
// 	defer res.Body.Close()
// 	var respMap tokenResponse
// 	err = json.NewDecoder(res.Body).Decode(&respMap)
// 	if err != nil {
// 		return true, fmt.Errorf("could not unmarshall response with status %d: %s\nerror cause: %s", res.StatusCode, res.Status, err.Error())
// 	}
// 	sdkConfig.IdToken = respMap.AccessToken
// 	sdkConfig.RefreshToken = respMap.RefreshToken
// 	sdkConfig.TokenExpiresAt = int(time.Now().Unix()) + respMap.ExpiresIn
// 	return true, nil
// }
