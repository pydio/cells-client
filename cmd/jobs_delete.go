package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v4/models"

	ent_client "github.com/pydio/cells-enterprise-sdk-go/client"
	"github.com/pydio/cells-enterprise-sdk-go/client/scheduler_service"
)

var (
	jobsDeleteOutputFormat string
	filterDelRaw           string
	jobsDeleteJobId        string
	forceDeleteSystemJob   bool
	dryRun                 bool
)

type Result struct {
	JobID    string
	JobLabel string
	Result   string
}

const (
	PydioSystemUser string = "pydio.system.user"
)

var jobsDelete = &cobra.Command{
	Use:   "delete",
	Short: "Delete jobs",
	Long: `
DESCRIPTION	

	Delete jobs from Cells Scheduler/CellsFlows.
	Deleting "pydio.system.user" job requires administrator privileges and user's confirmation.
	Use 'filter' parameter to delete multiple jobs.


FILTER format
# Get jobs with filter:

Filter struct:
{
	"field": {
		"op": "",
		"value": "",
	}
}

field: 
 - numtask: number of task of the job (numeric type)
 - owner: the owner of task (string type)
 - task_status: casesensitive, last task's status (valid values: Unknown|Idle|Running|Interrupted|Paused|Error|Queued|Finished)
 - multiple fields will applied using AND operator.
ops: 'eq' 'ne' 'gt' 'lt'

EXAMPLE:
# Delete jobs owned by 'admin' user:
$` + os.Args[0] + ` jobs delete --filter "{\"owner\": {\"op\": \"eq\", \"value\":\"admin\"}}" --format table

# Delete job by id:
$` + os.Args[0] + ` jobs delete --job-id 18ab830f-439a-4123-ad7a-1fdeb6f705a3 --dry-run=false

# Delete users' jobs with Error status
$` + os.Args[0] + ` jobs delete  --filter "{\"owner\": {\"op\":\"ne\", \"value\": \"pydio.system.user\"},\"task_status\": {\"op\":\"eq\", \"value\": \"Er
ror\"}}" --format table

# Delete systemd job
$` + os.Args[0] + ` jobs delete --job-id d29be854-e369-4f7d-86e5-2292f3fee49b --dry-run=false --force=true
`,
	Run: func(cmd *cobra.Command, args []string) {
		deleteSystemJobConfirmation := "no"
		// Connect to the Cells API
		ctx := cmd.Context()
		apiClient := sdkClient.GetApiClient()

		jobs, err := listUserJobs(ctx, apiClient)
		if err != nil {
			cmd.Printf(err.Error())
		}

		var filters FilterMap
		filteredJobs := []*models.JobsJob{}

		if filterDelRaw != "" {
			err := json.Unmarshal([]byte(filterDelRaw), &filters)
			if err != nil {
				log.Fatalf("invalid filter JSON: %v", err)
			}
			filterMap := make(map[string]interface{})
			for _, j := range jobs {

				if _, ok := filters["owner"]; ok {
					filterMap["owner"] = j.Owner
				}

				if _, ok := filters["numtask"]; ok {
					filterMap["numtask"] = len(j.Tasks)
				}
				if _, ok := filters["task_status"]; len(j.Tasks) > 0 && ok {
					filterMap["task_status"] = j.Tasks[0].Status
				} else {
					filterMap["task_status"] = "undefined-value"
				}

				// Apply filter
				if matchesFilters(filterMap, filters) {
					filteredJobs = append(filteredJobs, j)
				}
			}
		}

		rets := []*Result{}
		retMessage := "done"

		if jobsDeleteJobId != "" {
			var err error
			filteredJobs, err = getJobById(cmd.Context(), apiClient, jobsDeleteJobId)
			if err != nil {
				cmd.PrintErr(err)
				return
			}

			if forceDeleteSystemJob && len(filteredJobs) > 0 {
				fmt.Printf("⚠️  Are you sure you want to delete [%s] job? Type 'yes' to confirm: ", filteredJobs[0].Label)
				fmt.Scanln(&deleteSystemJobConfirmation)
				if deleteSystemJobConfirmation != "yes" {
					fmt.Println("Aborted.")
					return
				}
			}
		}

		if len(filteredJobs) == 0 {
			cmd.Printf("No job found!\n")
			return
		}

		for _, j := range filteredJobs {
			var err error
			if dryRun {
				cmd.Printf("You are running in dryRun mode\n")
				retMessage = "(dry-run)" + retMessage
			} else {
				// filter system jobs
				if j.Owner == PydioSystemUser && deleteSystemJobConfirmation != "yes" {
					retMessage = fmt.Sprintf("ignored: you are not authorized to delete %s jobs", PydioSystemUser)
				} else {
					err = deleteUserJobs(context.Background(), j.ID)
				}
			}

			if err != nil {
				retMessage = err.Error()
			}
			rets = append(rets, &Result{JobID: j.ID, JobLabel: j.Label, Result: retMessage})
			retMessage = "done"
		}

		renderResult(jobsDeleteOutputFormat, rets)
	},
}

func init() {
	flags := jobsDelete.PersistentFlags()
	flags.StringVarP(&jobsDeleteOutputFormat, "format", "f", "json", "Output format json|table")
	flags.StringVarP(&filterDelRaw, "filter", "", "", "JSON encoded filter string")
	flags.StringVarP(&jobsDeleteJobId, "job-id", "", "", "Job ID")
	flags.BoolVarP(&forceDeleteSystemJob, "force", "", false, "Deleteing system jobs requires administrator privileges. Only deleting by id --job-id is supported")
	flags.BoolVarP(&dryRun, "dry-run", "", true, "dry-run default: true")

	jobsCmd.AddCommand(jobsDelete)
}

func renderResult(outputFormat string, rets []*Result) {
	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(rets, "", "  ")
		fmt.Printf("%s\n", data)
		return
	case "table":
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Label", "Result"})
		for _, job := range rets {
			table.Append([]string{job.JobID, job.JobLabel, job.Result})
		}
		table.Render()
		return
	default:
		fmt.Printf("format must be either json or table\n")
		return
	}
}

func deleteUserJobs(_ context.Context, jobID string) error {
	param := scheduler_service.NewDeleteJobParams()
	param.JobID = jobID

	client := sdkClient.GetApiClient()
	entClient := ent_client.Default
	entClient.SetTransport(client.Transport)

	ret, err := entClient.SchedulerService.DeleteJob(param)
	if err != nil {
		return err
	}

	if ret.IsSuccess() {
		return nil
	}

	return fmt.Errorf("%s",ret.Error())
}
