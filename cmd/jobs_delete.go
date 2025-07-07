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

	eeclient "github.com/pydio/cells-enterprise-sdk-go/client"
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

  Delete some jobs from your Cells server scheduler.
  See the parent "jobs" subcommand to get more info about the jobs. 

  If you are a standard user, you can only delete jobs that you own. 
  You can optionally use a filter to delete multiple jobs at once.
  An administrator can delete jobs owned by other users.
  
  To delete system jobs (that are owned by the "pydio.system.user" user):
   - you must have administrative privileges 
   - You must confirm the deletion
   - you can only delete **1** job at a time, using the job-id param: 
     trying to delete system jobs returned by a filter will always fail.

   If you are unsure, use the --dry-run flag to only list the action that would be done.

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
       - owner: the owner of the job (string type). This filter can be only used by a user with admin privileges
       - numtasks: number of tasks (a.k.a instance) of the job (numeric type)
       - task_status: status of the last task of the job. Warning: it is case sensitive 
	     and the valid values are: Unknown | Idle | Running | Interrupted | Paused | Error | Queued | Finished 
    2. Known operators are:
	   - numeric values: eq | ne | gt | lt
	   - string values: eq | ne 	
    3. If you filter with more than one field, we apply the 'AND' operator between fields

EXAMPLES

  # Check all jobs owned by user alice that will get deleted (dry run):
  $` + os.Args[0] + ` jobs delete --filter "{\"owner\": {\"op\": \"eq\", \"value\":\"alice\"}}" --format table --dry-run

  # Really delete a job by id:
  $` + os.Args[0] + ` jobs delete --job-id 18ab830f-439a-4123-ad7a-1fdeb6f705a3

  # Delete all user jobs (a.k.a *not* system jobs) that are in error
  $` + os.Args[0] + ` jobs delete  --filter "{\"owner\": {\"op\":\"ne\", \"value\": \"pydio.system.user\"},\"task_status\": {\"op\":\"eq\", \"value\": \"Error\"}}"

  # Delete system job without confirmation (at you own risk)
  $` + os.Args[0] + ` jobs delete --job-id d29be854-e369-4f7d-86e5-2292f3fee49b --force

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
			filterMap := make(map[string]any)
			for _, j := range jobs {
				if _, ok := filters["owner"]; ok {
					filterMap["owner"] = j.Owner
				}
				if _, ok := filters["numtasks"]; ok {
					filterMap["numtasks"] = len(j.Tasks)
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
		delResultMsg := "done"

		hasDeleteConfirmation := false
		if deleteJobJobId != "" {
			var err error
			filteredJobs, err = getJobById(cmd.Context(), apiClient, deleteJobJobId)
			if err != nil {
				cmd.PrintErr(err)
				return
			}

			if len(filteredJobs) == 1 {
				// without --force parameter, ask user for confirmation
				if !deleteSystemJob {
					fmt.Printf("⚠️  Are you sure you want to delete [%s] job? Type 'yes' to confirm: ", filteredJobs[0].Label)
					_, err := fmt.Scanln(&deleteSystemJobConfirmation)
					if err != nil {
						log.Fatalf("unexpected error while getting user's confirmation: %s", err)
						return
					}

					if deleteSystemJobConfirmation != "yes" {
						fmt.Println("Aborting upon user's request.")
						return
					}
				}

			}
		}

		hasDeleteConfirmation = deleteSystemJob || (deleteSystemJobConfirmation == "yes")

		if len(filteredJobs) == 0 {
			cmd.Printf("No job found with provided filter, nothing to delete!\n")
			return
		}

		if deleteJobDryRun {
			cmd.Printf("Note: you are running in dry run mode. Your server's jobs won't be touched\n")
		}

		for _, j := range filteredJobs {
			// filter system jobs
			if j.Owner == PydioSystemUser && !hasDeleteConfirmation {
				delResultMsg = "skipped: you cannot delete system jobs using batches"
			} else {
				if deleteJobDryRun {
					delResultMsg = "done (dry-run)"
				} else {
					if err2 := deleteUserJobs(cmd.Context(), j.ID); err2 != nil {
						delResultMsg = "cannot delete: ," + err.Error()
					} else {
						delResultMsg = "done"
					}
				}
			}
			results = append(results, &Result{JobID: j.ID, JobLabel: j.Label, Result: delResultMsg})
		}
		renderResult(deleteJobOutputFormat, results)
	},
}

func init() {
	flags := jobsDelete.PersistentFlags()
	flags.StringVar(&deleteJobOutputFormat, "format", "table", "Output format table|json")
	flags.StringVar(&deleteJobFilter, "filter", "", "JSON encoded filter string")
	flags.StringVar(&deleteJobJobId, "job-id", "", "Job ID")
	flags.BoolVar(&deleteSystemJob, "force", false, "Deleting a system job requires confirmation: you might skip the validation witrh this flag. WARNING: this is dangerous and you might break your server, handle with care")
	flags.BoolVar(&deleteJobDryRun, "dry-run", false, "Only display the jobs to be deleted, without actually impacting anything on the server")
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
	entClient := eeclient.Default
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
