// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dispatch

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v45/github"
	"github.com/hashicorp/go-hclog"
)

// Result reports on the outcome of a given job, including if it was successful
// or not, and (if unsuccessful) details on any errors that ocurred
type Result struct {
	Name    string
	Success bool
	Error   error
}

// Options provides a way to define how frequently the GitHub APIs should be
// polled for results, as well as the maximum number of attempts before stopping
type Options struct {
	SecondsBetweenPolls int
	MaxAttempts         int
	Logger              hclog.Logger
	BranchRef           string
	BatchID             string
	WorkflowFileName    string
	GitHubOwner         string
	GitHubRepo          string
}

// WaitRunFinished watches a GitHub Actions Workflow Run and returns once the
// workflow has finished processing
func WaitRunFinished(client *github.Client, opts Options, run github.WorkflowRun) error {
	// Short circuit if stuff went really fast
	if *run.Status == "completed" {
		return nil
	}

	for i := 0; i < opts.MaxAttempts; i++ {
		opts.Logger.Debug(fmt.Sprintf("Waiting %d of 5 for run to finish: %s", i, *run.Name))
		time.Sleep(time.Duration(opts.SecondsBetweenPolls) * time.Second)

		this, _, err := client.Actions.GetWorkflowRunByID(context.Background(), opts.GitHubOwner, opts.GitHubRepo, *run.ID)
		if err != nil {
			return err
		}

		switch *this.Status {
		case "completed":
			return nil
		case "queued":
			// Do nothing, keep watching
		case "in_progress":
			// Do nothing, keep watching
		default:
			return fmt.Errorf("Workflow \"%s\" is in unrepairable state: %s", *run.Name, *this.Status)
		}
	}

	return fmt.Errorf("Timed out polling for workflow job")
}

// FindRun finds the most recent GitHub Actions run matching a given run name.
//
// FindRun requires that the `run-name:` tag in a workflow match the `runName`
// input. For more information about how `run-name:` works in GitHub Actions,
// refer to: https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#run-name
//
// Polling is defined by the `Options.SecondsBetweenPolls` parameter.
// If no run is returned after `Options.MaxAttempts` attempts, an error is returned
func FindRun(client *github.Client, opts Options, runName string) (github.WorkflowRun, error) {
	searchOpts := &github.ListWorkflowRunsOptions{
		Branch: opts.BranchRef,
		// Only search for workflow runs from today
		Created: time.Now().Format("2006-01-02"),
	}

	for i := 0; i < opts.MaxAttempts; i++ {
		opts.Logger.Debug(fmt.Sprintf("Attempt %d of %d to find run for %s", i, opts.MaxAttempts, runName))

		runs, _, err := client.Actions.ListWorkflowRunsByFileName(context.Background(), opts.GitHubOwner, opts.GitHubRepo, opts.WorkflowFileName, searchOpts)
		if err != nil {
			// TODO: handle rate limiting
			return github.WorkflowRun{}, fmt.Errorf("Error attempting to find the \"%s\" workflow run: %w", runName, err)
		}

		for _, v := range runs.WorkflowRuns {
			if *v.Name == runName {
				return *v, nil
			}
		}

		time.Sleep(time.Duration(opts.SecondsBetweenPolls) * time.Second)
	}
	return github.WorkflowRun{}, fmt.Errorf("Timed out polling for workflow job")
}

// Worker spawns an instance of a goroutine that listens for new job requests
// and then processes those requests until complete. Multiple workers can be
// instantiated to create a pool for concurrent processing.
//
// Workers create a GitHub Actions workflow run and follow the status of the job
// until it completes or errors out. The `results` channel is populated with
// the outcome of any jobs.
func Worker(client *github.Client, opts Options, id int, jobs <-chan string, results chan<- Result) {
	for repo := range jobs {
		opts.Logger.Info(fmt.Sprint("worker ", id, " started job ", repo))

		// The run name is in the form of `<batchID>: Audit <repoName>`, e.g.:
		// 01GFS35ZP6MQJHBF4QX1EFD6Y3: Audit go-hclog
		// TODO: This formatting is highly coupled to the `run-name:` tag in the
		// `repair-repo-license.yml` file. Perhaps explore other ways of declaring
		// this format only once instead of twice.
		runName := fmt.Sprintf("%s: Audit %s", opts.BatchID, repo)

		// Dispatch a Github Actions job to audit the given repo
		event := github.CreateWorkflowDispatchEventRequest{
			Ref: opts.BranchRef,
			Inputs: map[string]interface{}{
				"repo":      repo,
				"unique_id": opts.BatchID,
				"dry_run":   "false",
			},
		}

		opts.Logger.Debug(fmt.Sprintf("Starting workflow run: %s", runName))
		_, err := client.Actions.CreateWorkflowDispatchEventByFileName(context.Background(), opts.GitHubOwner, opts.GitHubRepo, opts.WorkflowFileName, event)
		if err != nil {
			results <- Result{
				Name:    repo,
				Success: false,
				Error:   err,
			}
			opts.Logger.Debug(fmt.Sprintf("Failed workflow run: %s", runName))
			continue
		}

		// GitHub Actions only returns a 200 OK when dispatching a job. It doesn't
		// return any Job ID or other identifying info, so we have to poll GitHub's
		// API to grab info about the actual run we spawned.
		run, err := FindRun(client, opts, runName)
		if err != nil {
			results <- Result{
				Name:    repo,
				Success: false,
				Error:   err,
			}
			opts.Logger.Debug(fmt.Sprintf("Failed workflow run: %s", runName))
			continue
		}

		// Now that we have identified a Job ID for the run we care about, let's
		// follow it until the run is done (successful, failed, or cancelled)
		err = WaitRunFinished(client, opts, run)
		if err != nil {
			results <- Result{
				Name:    repo,
				Success: false,
				Error:   err,
			}
			opts.Logger.Debug(fmt.Sprintf("Failed workflow run: %s", runName))
			continue
		}

		// All done here! No errors, so let's send a successful result back
		opts.Logger.Info(fmt.Sprint("worker ", id, " finished job ", repo))
		results <- Result{
			Name:    repo,
			Success: true,
			Error:   nil,
		}
	}
}
