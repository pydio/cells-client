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
	Operator string      `json:"op"`
	Value    interface{} `json:"value"`
}

type FilterMap map[string]FilterCondition

var jobsGet = &cobra.Command{
	Use:   "get",
	Short: "Get jobs associated with current user",
	Long: `
DESCRIPTION	

	Get jobs of current user.

EXAMPLE

# Get all jobs of current user:

$` + os.Args[0] + ` jobs get

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
 - 'AND' operator applies multiple fields.
ops: 'eq' 'ne' 'gt' 'lt'

Example:
# List all jobs owned by 'admin' user:
$` + os.Args[0] + ` jobs get --filter "{\"owner\": {\"op\": \"eq\", \"value\":\"admin\"}}" --format table

$` + os.Args[0] + ` jobs get --filter "{\"task_status\": {\"op\": \"eq\", \"value\":\"Error\"}, \"owner\": {\"op\":\"eq\", \"value\": \"admin\"}}" 
--format table
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

				if _, ok := filters["numtask"]; ok {
					filterMap["numtask"] = len(j.Tasks)
				}
				if _, ok := filters["task_status"]; len(j.Tasks) > 0 && ok {
					filterMap["task_status"] = j.Tasks[0].Status
				}else{
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
				table.Append([]string{job.ID, job.Label, job.Owner, fmt.Sprintf("%d", len(job.Tasks)), taskStatus})
			}
			table.Render()
			return
		default:
			cmd.Printf("format must be either json or table\n")
			return
		}
	},
}

func init() {
	flags := jobsGet.PersistentFlags()
	flags.StringVarP(&jobsOutputFormat, "format", "f", "json", "Output format json|table")
	flags.StringVar(&filterRaw, "filter", "", "Filter in JSON encoded string")

	jobsCmd.AddCommand(jobsGet)
}

func listUserJobs(_ context.Context, api *client.PydioCellsRestAPI) ([]*models.JobsJob, error) {
	param := jobs_service.NewUserListJobsParams()
	param.Body = &models.JobsListJobsRequest{
		LoadTasks:  models.JobsTaskStatusAny.Pointer(),
		Owner:      "*",
		TasksLimit: 10,
	}

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
		Owner:      "*",
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
	case ">":
		return af > bf
	case "<":
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

func matchesFilters(item map[string]interface{}, filters FilterMap) bool {
	for key, cond := range filters {
		val, ok := item[key]
		if !ok {
			return false
		}

		// Try string comparison
		if vs, ok := normalizeToString(val); ok {
			if cs, ok := normalizeToString(cond.Value); ok {
				switch cond.Operator {
				case "eq":
					if vs != cs {
						return false
					}
				case "ne":
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

func normalizeToString(v interface{}) (string, bool) {
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
