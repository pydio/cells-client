package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pydio/cells-sdk-go/v5/client/jobs_service"
	"github.com/pydio/cells-sdk-go/v5/models"
)

func MoveParams(source []string, targetFolder string) string {
	if !strings.HasSuffix(targetFolder, "/") {
		return BuildParams(source, targetFolder, false)
	}
	return BuildParams(source, targetFolder, true)
}

func RenameParams(source []string, targetFolder string) string {
	return BuildParams(source, targetFolder, false)
}

func CopyParams(source []string, targetFolder string) string {
	return BuildParams(source, targetFolder, true)
}

func BuildParams(source []string, targetFolder string, targetParent bool) string {
	type p struct {
		Target       string   `json:"target"`
		Nodes        []string `json:"nodes"`
		TargetParent bool     `json:"targetParent"`
	}
	i := &p{
		Nodes:        source,
		Target:       targetFolder,
		TargetParent: targetParent,
	}
	data, _ := json.Marshal(i)
	return string(data)
}

func CopyJob(ctx context.Context, jsonParams string) (string, error) {
	return RunJob(ctx, "copy", jsonParams)
}

func MoveJob(ctx context.Context, jsonParams string) (string, error) {
	return RunJob(ctx, "move", jsonParams)
}

// RunJob runs a job.
func RunJob(ctx context.Context, jobName string, jsonParams string) (string, error) {

	client, err := GetApiClient()
	if err != nil {
		return "", err
	}
	param := jobs_service.NewUserCreateJobParamsWithContext(ctx)
	param.Body = jobs_service.UserCreateJobBody{JSONParameters: jsonParams}
	param.JobName = jobName

	job, err := client.JobsService.UserCreateJob(param)
	if err != nil {
		return "", err
	}
	return job.Payload.JobUUID, nil
}

// GetTaskStatusForJob retrieves the task status, progress and message.
func GetTaskStatusForJob(ctx context.Context, jobID string) (status models.JobsTaskStatus, msg string, pg float32, e error) {
	client, err := GetApiClient()
	if err != nil {
		e = err
		return
	}
	body := &models.JobsListJobsRequest{
		JobIDs:    []string{jobID},
		LoadTasks: models.NewJobsTaskStatus(models.JobsTaskStatusAny),
	}
	params := jobs_service.NewUserListJobsParamsWithContext(ctx)
	params.Body = body
	jobs, err := client.JobsService.UserListJobs(params)
	if err != nil {
		e = err
		return
	}
	for _, job := range jobs.Payload.Jobs {
		if len(job.Tasks) == 0 {
			e = fmt.Errorf("no task found")
			return
		}
		for _, task := range job.Tasks {
			status = *task.Status
			msg = task.StatusMessage
			if task.HasProgress {
				pg = task.Progress
			}
		}
	}
	return
}

// MonitorJob monitors a job status every second.
func MonitorJob(ctx context.Context, JobID string) (err error) {
	for {
		status, _, _, e := GetTaskStatusForJob(ctx, JobID)
		if err != nil {
			err = e
			return
		}

		switch status {
		case models.JobsTaskStatusRunning, models.JobsTaskStatusPaused, models.JobsTaskStatusQueued:
			//fmt.Println("running, progress: ", pg)
			<-time.After(500 * time.Millisecond)

		case models.JobsTaskStatusError:
			err = fmt.Errorf("JobTask status error, %s", status)
			return
		case models.JobsTaskStatusInterrupted:
			err = fmt.Errorf("JobTask was interrupted by user")
			return
		case models.JobsTaskStatusUnknown:
			err = fmt.Errorf("JobTask unknown status, this is abnormal")
			return
		case models.JobsTaskStatusIdle:
			fmt.Println("IDLE")
			return
		case models.JobsTaskStatusFinished:
			// TODO remove this and add progress bar
			// fmt.Printf("Job : %s | Status : %s\n", JobID, status)
			return
		default:
			return
		}
	}
}
