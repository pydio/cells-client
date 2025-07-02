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
	deleteJobOutputFormat string
	deleteJobFilter       string
	deleteJobJobId        string
	deleteSystemJob       bool
	deleteJobDryRun       bool
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

  Delete jobs from Cells Flow (the embedded job scheduler).
  Note that deleting a "pydio.system.user" job requires administrative privileges and the user's confirmation.
  You can use the filter parameter to delete multiple jobs at once.

SYNTAX

  To delete more than one job at a time, you might pass a filter defined as a simple JSON encoded string. 
  The filter must be structured as follow (formatted as a reader friendly blob):

  {
	"field1": {
		"op": "",
		"value": "",
	}, 
	"field2": {
		"op": "",
		"value": "",
	},
	...
  }

  Where:
    1. Known fields are:
       - numtask: number of task of the job (numeric type)
       - owner: the owner of task (string type)
       - task_status: status of the last task of the job. Warning: it is case sensitive 
	     and the valid values are: Unknown | Idle | Running | Interrupted | Paused | Error | Queued | Finished 
    2. Known operators are: eq | ne | gt | lt
	3. If you filter with more than one field, we apply the 'AND' operator between fields

EXAMPLES

  # Delete jobs owned by 'admin' user:
  $` + os.Args[0] + ` jobs delete --filter "{\"owner\": {\"op\": \"eq\", \"value\":\"admin\"}}" --format table

  # Delete job by id:
  $` + os.Args[0] + ` jobs delete --job-id 18ab830f-439a-4123-ad7a-1fdeb6f705a3 --dry-run=false

  # Delete user jobs with Error status
  $` + os.Args[0] + ` jobs delete  --filter "{\"owner\": {\"op\":\"ne\", \"value\": \"pydio.system.user\"},\"task_status\": {\"op\":\"eq\", \"value\": \"Error\"}}" --format table

  # Delete system job
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
		var filteredJobs []*models.JobsJob

		if deleteJobFilter != "" {
			err := json.Unmarshal([]byte(deleteJobFilter), &filters)
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

		var results []*Result
		retMessage := "done"

		if deleteJobJobId != "" {
			var err error
			filteredJobs, err = getJobById(cmd.Context(), apiClient, deleteJobJobId)
			if err != nil {
				cmd.PrintErr(err)
				return
			}

			if deleteSystemJob && len(filteredJobs) > 0 {
				fmt.Printf("⚠️  Are you sure you want to delete [%s] job? Type 'yes' to confirm: ", filteredJobs[0].Label)
				_, err := fmt.Scanln(&deleteSystemJobConfirmation)
				if err != nil {
					log.Fatalf("unexpected error while getting user'S confirmation: %s", err)
					return
				}
				if deleteSystemJobConfirmation != "yes" {
					fmt.Println("Aborting upon user's request.")
					return
				}
			}
		}

		if len(filteredJobs) == 0 {
			cmd.Printf("No job found with provided filter, nothing to delete!\n")
			return
		}

		if deleteJobDryRun {
			cmd.Printf("Note: you are running in dry run mode. Your server's jobs won't be touched\n")
		}

		for _, j := range filteredJobs {
			var err error
			if deleteJobDryRun {
				retMessage = "(dry-run)" + retMessage
			} else {
				// filter system jobs
				if j.Owner == PydioSystemUser && deleteSystemJobConfirmation != "yes" {
					retMessage = fmt.Sprintf("ignored: you are not authorized to delete %s jobs", PydioSystemUser)
				} else {
					err = deleteUserJobs(cmd.Context(), j.ID)
				}
			}

			if err != nil {
				retMessage = err.Error()
			}
			results = append(results, &Result{JobID: j.ID, JobLabel: j.Label, Result: retMessage})
			retMessage = "done"
		}

		renderResult(deleteJobOutputFormat, results)
	},
}

func init() {
	flags := jobsDelete.PersistentFlags()
	flags.StringVar(&deleteJobOutputFormat, "format", "json", "Output format json|table")
	flags.StringVar(&deleteJobFilter, "filter", "", "JSON encoded filter string")
	flags.StringVar(&deleteJobJobId, "job-id", "", "Job ID")
	flags.BoolVar(&deleteSystemJob, "force", false, "Deleting system jobs requires administrator privileges. Only deleting by job-id is supported")
	flags.BoolVar(&deleteJobDryRun, "dry-run", true, "Only display the jobs to be deleted, without actually impacting anything on the server")
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

	return fmt.Errorf("%s", ret.Error())
}
