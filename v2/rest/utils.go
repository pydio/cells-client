package rest

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
)

// RetryCallback implements boiler plate code to easily call the same function until it suceeds
// or a time-out is reached.
func RetryCallback(callback func() error, number int, interval time.Duration) error {

	var e error
	for i := 0; i < number; i++ {
		if e = callback(); e == nil {
			break
		}
		if i < number-1 {
			<-time.After(interval)
		}
	}

	return e
}

// RetrieveCurrentSessionLogin requests the registry of the current configured server & login
// and parse the result to get current user login. Typically useful when using PAT auth.
func RetrieveCurrentSessionLogin() (string, error) {

	uri := "/a/frontend/state"
	resp, err := AuthenticatedGet(uri)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Simply check tags on the fly and stops when a <user id=""> tag has been found.
	decoder := xml.NewDecoder(resp.Body)
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "user" && se.Attr[0].Name.Local == "id" {
				return se.Attr[0].Value, nil
			}
		}
	}
	return "", fmt.Errorf("no <user> tag found in registry. Are you sure you are connected?")
}

func CleanURL(input string) (string, error) {
	input = strings.TrimSpace(input)
	tmpURL, err := url.Parse(input)
	if err != nil {
		return "", err
	}
	output := tmpURL.Scheme + "://" + tmpURL.Host
	return output, nil
}

func IsForbiddenError(err error) bool {
	switch e := err.(type) {
	case *runtime.APIError:
		return e.Code == 401
	}
	return false
}
