// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"

	"github.com/google/go-github/v45/github"
	"github.com/hashicorp/copywrite/dispatch"
	gh "github.com/hashicorp/copywrite/github"
	"github.com/hashicorp/copywrite/repodata"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/thanhpk/randstr"
)

var dispatchCmd = &cobra.Command{
	Use:   "dispatch",
	Short: "Dispatches audit jobs for a list of repos",
	Long:  `Dispatches audit jobs for all public and non-archived repos`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Map command flags to config keys
		mapping := map[string]string{
			`batch-id`:     `dispatch.batch_id`,
			`branch`:       `dispatch.branch`,
			`max-attempts`: `dispatch.max_attempts`,
			`sleep`:        `dispatch.sleep`,
			`workers`:      `dispatch.workers`,
			`workflow`:     `dispatch.workflow_file_name`,
			`github-org`:   `dispatch.github_org_to_audit`,
		}

		// update the running config with any command-line flags
		clobberWithDefaults := false
		err := conf.LoadCommandFlags(cmd.Flags(), mapping, clobberWithDefaults)
		if err != nil {
			cliLogger.Error("Error merging configuration", err)
		}
		cobra.CheckErr(err)

		// Dynamically generate a batchID if none is supplied
		if conf.Dispatch.BatchID == "" {
			conf.Dispatch.BatchID = randstr.Hex(8) // 8-digit random string
			cliLogger.Debug(fmt.Sprintf("Using auto-generated batchID: %s", conf.Dispatch.BatchID))
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		client := gh.NewGHClient().Raw()

		// Retrieve all public, non-archived GitHub repos for auditing
		allRepos, err := repodata.GetRepos(conf.Dispatch.GitHubOrgToAudit)
		cobra.CheckErr(err)

		targetRepos := repodata.FilterRepos(allRepos)

		if len(conf.Dispatch.IgnoredRepos) > 0 {
			gha.StartGroup("Exempting the following repos:")
			for _, v := range conf.Dispatch.IgnoredRepos {
				cliLogger.Info(text.FgCyan.Sprint(v))
			}
			gha.EndGroup()

			// Filter out any repos that are on the ignore list
			targetRepos = lo.Filter(targetRepos, func(r *github.Repository, i int) bool {
				fqn := fmt.Sprintf("%v/%v", conf.Dispatch.GitHubOrgToAudit, r.GetName())
				return !lo.Contains(conf.Dispatch.IgnoredRepos, fqn)
			})
		}

		cliLogger.Info(fmt.Sprintf("Repositories will be audited with the \"%v\" GitHub Actions workflow", conf.Dispatch.WorkflowFileName))
		cliLogger.Info(fmt.Sprintf("Set to process %v GitHub repositories with %v concurrent workers", len(targetRepos), conf.Dispatch.Workers))

		if plan {
			cliLogger.Info(text.Bold.Sprint("The following repos would be audited:"))
			for _, v := range targetRepos {
				cliLogger.Info(fmt.Sprintf("%v/%v", conf.Dispatch.GitHubOrgToAudit, *v.Name))
			}
			cliLogger.Info(text.FgYellow.Sprintf("Executing in dry-run mode. Rerun without the `--plan` flag to trigger audits on all %v repos.", len(targetRepos)))
			return
		}

		// The actual stuff

		repo, err := gh.DiscoverRepo()
		cobra.CheckErr(err)

		opts := dispatch.Options{
			SecondsBetweenPolls: conf.Dispatch.Sleep,
			MaxAttempts:         conf.Dispatch.MaxAttempts,
			Logger:              cliLogger.Named("dispatch"),
			BranchRef:           conf.Dispatch.Branch,
			BatchID:             conf.Dispatch.BatchID,
			WorkflowFileName:    conf.Dispatch.WorkflowFileName,
			GitHubOwner:         repo.Owner,
			GitHubRepo:          repo.Name,
		}

		numJobs := len(targetRepos)
		jobs := make(chan string, numJobs)
		results := make(chan dispatch.Result, numJobs)

		// Create a worker pool
		for w := 1; w <= conf.Dispatch.Workers; w++ {
			go dispatch.Worker(client, opts, w, jobs, results)
		}

		// Queue up all of the repos to be processed by the worker pool
		for _, v := range targetRepos {
			jobs <- *v.Name
		}

		// TODO: the 'jobs' channel will need to remain open if we decide to requeue
		// failed jobs in the future.
		close(jobs)

		// Let's print out any failure cases
		failures := []dispatch.Result{}
		for a := 1; a <= numJobs; a++ {
			result := <-results
			if !result.Success {
				failures = append(failures, result)
			}
		}

		if len(failures) > 0 {
			cliLogger.Error(fmt.Sprintf("Job failures occurred %d times:", len(failures)))
			for _, f := range failures {
				cliLogger.Error(fmt.Sprint(f))
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(dispatchCmd)

	// These flags are only locally relevant
	dispatchCmd.Flags().BoolVar(&plan, "plan", false, "Performs a dry-run, printing the names of all repos that would be audited")
	dispatchCmd.Flags().Int("max-attempts", 15, "Number of times a worker will check if a job has completed before timing out")
	dispatchCmd.Flags().IntP("sleep", "s", 10, "Seconds to sleep between polling opts")
	dispatchCmd.Flags().IntP("workers", "w", 2, "Concurrent jobs that can be ran")
	dispatchCmd.Flags().StringP("branch", "b", "main", "The GitHub Branch to base workflow runs off of")
	dispatchCmd.Flags().StringP("batch-id", "i", "", "A unique identifier for the current batch of workflow runs (defaults to an autogenerated ULID)")
	dispatchCmd.Flags().StringP("workflow", "n", "repair-repo-license.yml", "The workflow file name to be triggered")
	dispatchCmd.Flags().String("github-org", "hashicorp", "Sets the target GitHub org who's repos you wish to audit")
}
