package rest

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/go-openapi/runtime"

	"github.com/pydio/cells-client/v4/common"
)

// RetryCallback implements boilerplate code to easily call the same function until it succeeds
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

// RetrieveSessionLogin tries to get the registry of the server defined by the passed configuration
// and parse the result to get current user login. Typically useful when using PAT auth.
func RetrieveSessionLogin(newConf *CecConfig) (string, error) {
	uri := "/a/frontend/state"
	resp, err := getFrom(newConf, uri)
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

// RetrieveCurrentSessionLogin requests the registry of the current configured server & login
// and parse the result to get current user login. Typically useful when using PAT auth.
func RetrieveCurrentSessionLogin() (string, error) {

	uri := "/a/frontend/state"
	resp, err := authenticatedGet(uri)
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

// RetrieveRemoteServerVersion gets the version info from the distant server.
// User must be authenticated (and admin ?).
func RetrieveRemoteServerVersion() (*common.ServerVersion, error) {

	uri := "/a/frontend/bootconf"
	resp, err := authenticatedGet(uri)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve bootconf: %s", err.Error())
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var result map[string]interface{}
	err = decoder.Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("could unmarshall bootconf in map[string]interface{}: %s", err.Error())
	}

	tmp, ok := result["backend"]
	if !ok {
		return nil, fmt.Errorf("no 'backend' key found in the boot conf, could not get remote server version")
	}
	backend := tmp.(map[string]interface{})
	return safelyDecode(backend), nil
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
	var e *runtime.APIError
	switch {
	case errors.As(err, &e):
		return e.Code == 401
	}
	return false
}

func StandardizeLink(old string) string {
	if strings.HasPrefix(old, "/") && !strings.HasPrefix(old, "http") {
		return DefaultConfig.Url + old
	}
	return old
}

func Unique(length int) string {
	rand := fmt.Sprintf("%d", time.Now().Nanosecond())
	hash := md5.New()
	hash.Write([]byte(rand))
	return hex.EncodeToString(hash.Sum(nil))[0:length]
}

func ValidURL(input string) error {
	// Warning: trim must also be performed when retrieving the final value.
	// Here we only validate that the trimmed input is valid, but do not modify it.
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return fmt.Errorf("field cannot be empty")
	}
	u, e := url.Parse(input)
	if e != nil || u == nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("please, provide a valid URL")
	}
	return nil
}

const LetterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandString(n int) string {
	b := make([]byte, n)
	// rand.Seed(time.Now().Unix())
	for i := range b {
		b[i] = LetterBytes[rand.Intn(len(LetterBytes))]
	}
	return string(b)
}

func safelyDecode(src map[string]interface{}) *common.ServerVersion {

	version := &common.ServerVersion{}
	version.PackageType = sanitize(src, "PackageType")
	version.PackageLabel = sanitize(src, "PackageLabel")
	version.Version = sanitize(src, "Version")
	version.License = sanitize(src, "License")
	version.BuildRevision = sanitize(src, "BuildRevision")
	if tmp, ok := src["ServerOffset"]; ok {
		version.ServerOffset = int64(math.Round(tmp.(float64)))
	}
	if tmp, ok := src["BuildStamp"]; ok {
		if v, err := time.Parse("2006-01-02T15:04:05", tmp.(string)); err == nil {
			version.BuildStamp = v
		}
	}
	version.PackagingInfo = sanitizeLines(src, "PackagingInfo")
	return version
}

func sanitize(dic map[string]interface{}, key string) string {
	if tmp, ok := dic[key]; ok && tmp != nil {
		return tmp.(string)
	}
	return ""
}

func sanitizeLines(dic map[string]interface{}, key string) []string {
	if tmp, ok := dic[key]; ok && tmp != nil {
		res := make([]string, 0)
		tmpArr := tmp.([]interface{})
		for _, item := range tmpArr {
			res = append(res, item.(string))
		}
		return res
	}
	return nil
}
