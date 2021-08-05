package bench

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/pydio/cells-client/v2/common"
	"github.com/pydio/cells-client/v2/rest"
	"github.com/pydio/cells-sdk-go/v2/client/tree_service"
	"github.com/pydio/cells-sdk-go/v2/models"
	"github.com/spf13/cobra"
)

var (
	shareResourcePath string
)

var shareAccessCmd = &cobra.Command{
	Use:   "share_access",
	Short: "Access a public link in concurrency",
	Long:  "This command creates a simple resource (a folder), shares it as a public link and then emulate access of many concurrent users in parallel",
	Run: func(cmd *cobra.Command, args []string) {

		if shareResourcePath == "" {
			shareResourcePath = "common-files/test-public-link-" + rest.Unique(4)
		}

		link, err := createLink(shareResourcePath)
		if err != nil {
			log.Fatal(err)
		}

		wg := &sync.WaitGroup{}
		wg.Add(benchMaxRequests)
		throttle := make(chan struct{}, benchPoolSize)
		for i := 0; i < benchMaxRequests; i++ {
			throttle <- struct{}{}
			go func(id int) {
				singleAccess(id, link)
				wg.Done()
				<-throttle
			}(i)
		}
		wg.Wait()

		if !benchSkipClean {
			rest.DeleteNode([]string{shareResourcePath})
		}
	},
}

func createLink(targetPath string) (*models.RestShareLink, error) {

	// Use current active connection
	ctx, apiClient, err := rest.GetApiClient()
	if err != nil {
		log.Fatal(err)
	}

	// Create a target resource and wait until available
	_, err = apiClient.TreeService.CreateNodes(&tree_service.CreateNodesParams{
		Body: &models.RestCreateNodesRequest{
			Nodes: []*models.TreeNode{{
				Path: targetPath,
			}},
		},
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}
	var node *models.TreeNode
	exists := false
	for {
		node, exists = rest.StatNode(targetPath)
		if exists {
			break
		}
		log.Printf("No node found at %s, wait for a second before retry\n", targetPath)
		time.Sleep(1 * time.Second)
	}

	// Put 2 files in the target folder
	_, err = rest.PutFile(targetPath+"/dummyFile.txt", strings.NewReader("Simple test for sharing"), true)
	if err != nil {
		return nil, err
	}
	_, err = rest.PutFile(targetPath+"/dummyFile2.txt", strings.NewReader("Simple test for sharing - second file"), true)
	if err != nil {
		return nil, err
	}

	// Create a public link
	createdLink, err := rest.CreateSimpleLink(node.UUID, path.Base(targetPath))
	if err != nil {
		return nil, err
	}

	log.Printf("Resource created, public link is at %s \n", rest.StandardizeLink(createdLink.LinkURL))
	return createdLink, nil

}

func singleAccess(i int, l *models.RestShareLink) error {
	s := time.Now()

	currURL := rest.StandardizeLink(l.LinkURL)

	resp, err := http.Get(currURL)
	if err != nil {
		return err
	} else if resp.StatusCode != 200 {
		return fmt.Errorf("unvalid response when getting public link at %s: %d (%s)", l.LinkURL, resp.StatusCode, resp.Status)
	}

	token, err := getPublicToken(l.UserLogin)
	if err != nil {
		return err
	}

	_, err = getState(token)
	if err != nil {
		return err
	}

	res := time.Since(s)
	if err != nil {
		log.Println(i, res, "error", err.Error())
	} else {
		//log.Println(i, res, "- OK, token used:", token)
		log.Println(i, res, "ok")
	}

	return err
}

func getPublicToken(login string) (token string, e error) {

	creds := map[string]string{
		"login":    login,
		"password": login + "#$!Az1",
		"type":     "credentials",
	}
	jsonData, err := json.Marshal(map[string]interface{}{"AuthInfo": creds})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(
		rest.StandardizeLink("/a/frontend/session"),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		log.Fatal(err)
	}

	var res map[string]interface{}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&res)

	jwt := res["JWT"]
	if jwt == nil {
		return "", fmt.Errorf("no token found for %s", login)
	}
	return jwt.(string), nil
}

func getState(token string) (activeRepoID string, e error) {

	req, err := http.NewRequest(
		"GET",
		rest.StandardizeLink("/a/frontend/state"),
		nil,
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	v := &common.Cpydio_registry{}
	xml.Unmarshal(body, v)
	if v.Cuser == nil || v.Cuser.Cactive_repo == nil {
		return "", fmt.Errorf("no active repo found")
	}

	return v.Cuser.Cactive_repo.Attrid, nil
}

func init() {
	benchCmd.AddCommand(shareAccessCmd)
	shareAccessCmd.Flags().StringVarP(&shareResourcePath, "resource", "r", "", "Folder created that will be shared as a public link")
}
