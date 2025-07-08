package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/pydio/cells-sdk-go/v4/client"
	"github.com/pydio/cells-sdk-go/v4/client/jobs_service"
	"github.com/pydio/cells-sdk-go/v4/models"
)

var (
	jobsOutputFormat string
	filterRaw        string
)

type FilterCondition struct {
	Operator string `json:"op"`
	Value    any    `json:"value"`
}

type FilterMap map[string]FilterCondition

var jobsGet = &cobra.Command{
	Use:   "get",
	Short: "Query and list existing jobs",
	Long: `
DESCRIPTION	

  Launch a query to retrieve jobs from the server. 
  See the parent "jobs" subcommand to get more info about the jobs. 
  
  If you are connected with a standard user, you can only list the jobs that you own.
  When you are connected with admin privileges, you can list:
  	- the jobs that you own as a user (e.g.: a long running move that you have launched and that has not yet terminated)
	- the jobs that are owned by other users 
	- the system jobs that are triggered by the scheduler, some events or manually launched from the admin console.
	  These jobs have a "pydio.system.user" owner.
	  
SYNTAX
  
  To reduce the number of returned results, you might pass a filter defined as a simple JSON encoded string. 
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
       - numtasks: number of tasks of the job (numeric type)
       - task_status: status of the last task of the job. Warning: it is case sensitive 
	     and the valid values are: Unknown | Idle | Running | Interrupted | Paused | Error | Queued | Finished 
    2. Known operators are: 
	   - numeric values: eq | ne | gt | lt
	   - string values: eq | ne 
    3. If you filter with more than one field, we apply the 'AND' operator between fields

EXAMPLES

  # Get all jobs of current user:
  $` + os.Args[0] + ` jobs get

  # [admin only] List all jobs owned by user alice, formatted as a table:
  $` + os.Args[0] + ` jobs get --filter "{\"owner\": {\"op\": \"eq\", \"value\":\"alice\"}}" --format table

  # [admin only] List all jobs owned by user bob and that are in error in JSON:
  $` + os.Args[0] + ` jobs get --filter "{\"task_status\": {\"op\": \"eq\", \"value\":\"Error\"}, \"owner\": {\"op\":\"eq\", \"value\": \"bob\"}}" 

`,
	Run: func(cmd *cobra.Command, args []string) {

		// Connect to the Cells API
		ctx := cmd.Context()
		apiClient := sdkClient.GetApiClient()

		jobs, err := listUserJobs(ctx, apiClient)
		if err != nil {
			cmd.Printf(err.Error())
		}

		var filters FilterMap
		var filteredJobs []*models.JobsJob

		if filterRaw != "" {
			err := json.Unmarshal([]byte(filterRaw), &filters)
			if err != nil {
				log.Fatalf("invalid filter JSON: %v", err)
			}
			filterMap := make(map[string]interface{})
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
		} else {
			filteredJobs = jobs
		}

		switch jobsOutputFormat {
		case "json":
			data, _ := json.MarshalIndent(filteredJobs, "", "  ")
			fmt.Printf("%s\n", data)
			return
		case "table":
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Label", "Owner", "Num Tasks", "Last task status"})
			for _, job := range filteredJobs {
				taskStatus := ""
				if len(job.Tasks) > 0 {
					taskStatus = string(*job.Tasks[0].Status)
				}
				nbOfTasks := fmt.Sprintf("%d", len(job.Tasks))
				if nbOfTasks == "100" {
					nbOfTasks = "99+"
				}
				table.Append([]string{job.ID, job.Label, job.Owner, nbOfTasks, taskStatus})
			}
			table.Render()
			return
		default:
			cmd.Println("invalid output format, it must be either json or table")
			return
		}
	},
}

func init() {
	flags := jobsGet.PersistentFlags()
	flags.StringVar(&jobsOutputFormat, "format", "table", "Output format table|json")
	flags.StringVar(&filterRaw, "filter", "", "Filter in JSON encoded string")

	jobsCmd.AddCommand(jobsGet)
}

func listUserJobs(_ context.Context, api *client.PydioCellsRestAPI) ([]*models.JobsJob, error) {
	param := jobs_service.NewUserListJobsParams()
	param.Body = &models.JobsListJobsRequest{
		LoadTasks:  models.JobsTaskStatusAny.Pointer(),
		Owner:      "*",
		TasksLimit: 100,
	}
	// TODO Handle pagination until 1000
	jobs, err := api.JobsService.UserListJobs(param)
	if err != nil {
		return nil, err
	}
	return jobs.Payload.Jobs, nil
}

func getJobById(_ context.Context, api *client.PydioCellsRestAPI, jobId string) ([]*models.JobsJob, error) {
	param := jobs_service.NewUserListJobsParams()
	param.Body = &models.JobsListJobsRequest{
		JobIDs: []string{jobId},
		Owner:  "*",
	}

	jobs, err := api.JobsService.UserListJobs(param)
	if err != nil {
		return nil, err
	}
	return jobs.Payload.Jobs, nil
}

func compareNumbers(a, b interface{}, op string) bool {
	af, aok := toFloat64(a)
	bf, bok := toFloat64(b)
	if !aok || !bok {
		return false
	}

	switch op {
	case "gt", ">":
		return af > bf
	case "lt", "<":
		return af < bf
	case "eq", "==":
		return af == bf
	case "ne", "!=":
		return af != bf
	default:
		return false
	}
}

func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case int:
		return float64(v), true
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func matchesFilters(item map[string]any, filters FilterMap) bool {
	for key, cond := range filters {
		val, ok := item[key]
		if !ok {
			return false
		}

		// Try string comparison
		if vs, ok := normalizeToString(val); ok {
			if cs, ok := normalizeToString(cond.Value); ok {
				switch cond.Operator {
				case "eq", "==":
					if vs != cs {
						return false
					}
				case "ne", "!=":
					if vs == cs {
						return false
					}
				default:
					log.Printf("Unsupported operator %q for string values", cond.Operator)
					return false
				}
				continue
			}
		}

		// Try numeric comparison
		if compareNumbers(val, cond.Value, cond.Operator) {
			continue
		}

		// If neither string nor number comparison worked
		return false
	}
	return true
}

func normalizeToString(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case fmt.Stringer:
		return val.String(), true
	case *models.JobsTaskStatus:
		if val != nil {
			return string(*val), true
		}
	default:
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Ptr && !rv.IsNil() && rv.Elem().Kind() == reflect.String {
			return rv.Elem().String(), true
		}
	}
	return "", false
}
