package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v4/client/config_service"
	"github.com/pydio/cells-sdk-go/v4/client/jobs_service"
	"github.com/pydio/cells-sdk-go/v4/models"

	"github.com/pydio/cells-client/v4/rest"
)

var (
	ldRaw bool
)

var listDatasources = &cobra.Command{
	Use:   "list-datasources",
	Short: "List configured datasources",
	Long: `
DESCRIPTION 

  List all the datasources that are defined on the server side.
  Note that the currently used user account must have be given the necessary Admin permissions.
`,
	Run: func(cmd *cobra.Command, args []string) {

		ctx := cmd.Context()
		apiClient := sdkClient.GetApiClient()

		/*ListDataSourcesParams contains all the parameters to send to the API endpoint
		for the list data sources operation typically these are written to a http.Request */
		params := &config_service.ListDataSourcesParams{Context: ctx}

		//assigns the datasources data retrieved above in the results variable
		result, err := apiClient.ConfigService.ListDataSources(params)
		if err != nil {
			if rest.IsForbiddenError(err) {
				log.Fatalf("[Forbidden access] You do not have necessary permission to list the datasources at %s", sdkClient.GetConfig().Url)
			}
			log.Fatalf("Could not list data sources of %s, cause: %s", sdkClient.GetConfig().Url, err.Error())
		}

		//prints the name of the datasources retrieved previously
		if len(result.Payload.DataSources) > 0 {
			if ldRaw {
				for _, ds := range result.Payload.DataSources {
					if ds.Name == "" {
						continue
					}
					_, _ = fmt.Fprintln(os.Stdout, ds.Name)
				}
				return
			}
			fmt.Printf("* %d datasources	\n", len(result.Payload.DataSources))
			for _, u := range result.Payload.DataSources {
				fmt.Println("  - " + u.Name)
			}
		}

	},
}

var resyncDs = &cobra.Command{
	Use:   "resync-ds",
	Short: "Launch a resync",
	Long:  `Launch a resync job on the specified datasource`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) != 1 {
			log.Fatal(fmt.Errorf("please provide the name of the datasource to resync"))
		}
		dsName := args[0]

		jsonParams := fmt.Sprintf("{\"dsName\":\"%s\"}", dsName)
		body := &models.RestUserJobRequest{JSONParameters: jsonParams}
		params := &jobs_service.UserCreateJobParams{JobName: "datasource-resync", Body: body, Context: cmd.Context()}

		_, err := sdkClient.GetApiClient().JobsService.UserCreateJob(params)
		if err != nil {
			log.Fatalf(fmt.Sprintf("could not start the sync job for ds %s, cause: %s", dsName, err.Error()))
		}
		fmt.Printf("Starting resync on %s \n", dsName)
	},
}

func init() {
	storageCmd.AddCommand(listDatasources)
	storageCmd.AddCommand(resyncDs)

	ldFlags := listDatasources.PersistentFlags()
	ldFlags.BoolVarP(&ldRaw, "raw", "r", false, "List datasources name in raw format")
}
