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

const (
	noTaskFoundMsg = "no task found"
)

func (client *SdkClient) CopyJob(ctx context.Context, jsonParams string) (string, error) {
	return client.RunJob(ctx, "copy", jsonParams)
}

func (client *SdkClient) MoveJob(ctx context.Context, jsonParams string) (string, error) {
	return client.RunJob(ctx, "move", jsonParams)
}

// RunJob runs a job.
func (client *SdkClient) RunJob(ctx context.Context, jobName string, jsonParams string) (string, error) {
	param := jobs_service.NewUserCreateJobParamsWithContext(ctx)
	param.Body = jobs_service.UserCreateJobBody{JSONParameters: jsonParams}
	param.JobName = jobName

	job, err := client.GetApiClient().JobsService.UserCreateJob(param)
	if err != nil {
		return "", err
	}
	return job.Payload.JobUUID, nil
}

// GetTaskStatusForJob retrieves the task status, progress and message.
func (client *SdkClient) GetTaskStatusForJob(ctx context.Context, jobID string) (status models.JobsTaskStatus, msg string, pg float32, e error) {
	// jobID = "copy-move-944bc5a5-d05e-4e4e-84e7-ba4f7989acd4"
	body := &models.JobsListJobsRequest{
		JobIDs:    []string{jobID},
		LoadTasks: models.NewJobsTaskStatus(models.JobsTaskStatusAny),
	}
	params := jobs_service.NewUserListJobsParamsWithContext(ctx)
	params.Body = body
	jobs, err := client.GetApiClient().JobsService.UserListJobs(params)
	if err != nil {
		e = err
		return
	}
	for _, job := range jobs.Payload.Jobs {
		if len(job.Tasks) == 0 {
			e = fmt.Errorf(noTaskFoundMsg)
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

// GetCurrentJobStatus retrieves the current task status, progress and message.
func (client *SdkClient) GetCurrentJobStatus(ctx context.Context, jobID string) (status models.JobsTaskStatus, msg string, pg float32, e error) {

	err := RetryCallback(func() error {
		var tmpErr error
		status, msg, pg, tmpErr = client.GetTaskStatusForJob(ctx, jobID)
		if tmpErr != nil {
			// We only retry if no task for the job has already been found
			if tmpErr.Error() == noTaskFoundMsg {
				return tmpErr
			}
			// Otherwise, we skip retrial
			e = tmpErr
			return nil
		}
		return nil
	}, 5, 200*time.Millisecond)

	if err != nil {
		e = err
	}
	return
}

// MonitorJob monitors a job status every second.
func (client *SdkClient) MonitorJob(ctx context.Context, jobID string) (err error) {
	i := 0
	for {
		status, _, _, e := client.GetCurrentJobStatus(ctx, jobID)
		if e != nil {
			err = fmt.Errorf("could not get task status for job ID %s: %s", jobID, e)
			return
		}

		Log.Debugf(" #%d - %s ", i, status)
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
			// Rather use a progress bar?
			Log.Debugf(" Job with id %s has finished", jobID)
			return
		default:
			return
		}
	}
}

// Various Helpers

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
