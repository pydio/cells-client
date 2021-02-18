package cmd

import (
	"encoding/xml"
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-client/v2/rest"
)

var currentSession = &cobra.Command{
	Use:   "get-state",
	Short: "Get State",
	Long:  `Tmp command to retrieve the registry for your current connection to your Pydio Cells instance.`,

	Run: func(cm *cobra.Command, args []string) {
		uri := "/a/frontend/state"
		resp, err := rest.AuthenticatedGet(uri)
		if err != nil {
			fmt.Printf("could retrieve state: %s\n", err.Error())
			log.Fatal(err)
		}
		defer resp.Body.Close()
		decoder := xml.NewDecoder(resp.Body)
		login := ""

	loop:
		for {
			t, _ := decoder.Token()
			if t == nil {
				break
			}
			switch se := t.(type) {
			case xml.StartElement:
				if se.Name.Local == "user" && se.Attr[0].Name.Local == "id" {
					login = se.Attr[0].Value
					break loop
				}
			}
		}
		
		fmt.Printf("Found login: [%s]\n", login)
	},
}

func init() {
	idmCmd.AddCommand(currentSession)
}
