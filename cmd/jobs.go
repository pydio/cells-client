package cmd

import (
	"github.com/spf13/cobra"
)

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Manage jobs",
	Long: `
DESCRIPTION

  In Pydio Cells, a job represents the definition of a sequence of actions 
  (basic building blocks that can be chained together) to process data and 
  perform long running tasks in the background. 
  Typically, copying a folder with many files in it.

  There are 2 types of jobs:
  - System Jobs: they are owned by the "pydio.system.user" user and can be triggered 
    by events, the scheduler or manually launched. The server comes with a bunch of 
    jobs that are useful to maintain your system and e.g perform house keeping tasks.
    You can create new job templates using Cells Flow to fine-tune your business processes.
  - User Jobs: they start upon events received when a user performs certain action 
    (like extracting a thumbnail after successful upload of an image or deleting a large folder) 
    and are then owned by the user who triggered the action.

  When a job is launched, one (or more) instance is created and launched: we call them task. 

  Use the provided job sub-commands to search, list and delete existing jobs;
  see their respective help for further details.
	`,
	// Args: cobra.MinimumNArgs(1),
	RunE: func(cm *cobra.Command, args []string) error {
		return cm.Help()
	},
}

func init() {
	RootCmd.AddCommand(jobsCmd)
}
