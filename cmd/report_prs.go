// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v45/github"
	gh "github.com/hashicorp/copywrite/github"
	"github.com/mergestat/timediff"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

var (
	author string
	status string
)

var reportPRsCmd = &cobra.Command{
	Use:   "prs",
	Short: "Lists all unmerged compliance pull requests",
	Long: `Lists all unmerged compliance pull requests

By default, PRs are found by searching by author. Any PRs created by the
copyright notice automation tooling will be authored by "hashicorp-copywrite"`,
	Run: func(cmd *cobra.Command, args []string) {
		// Disable color pretty-print if not intended for human eyes
		if csv {
			text.DisableColors()
		}

		client := gh.NewGHClient().Raw()

		opt := &github.SearchOptions{
			ListOptions: github.ListOptions{PerPage: 100}, // 100 is the max page size
			Sort:        "created",
			Order:       "asc",
		}

		query := fmt.Sprintf("is:pr author:%s", author)

		// validate status flag and append to query if needed
		switch status {
		case "open", "closed":
			query = query + fmt.Sprintf(" is:%s", status)
		case "all":
			// Do nothing, omitting a filter defaults to "all"
		default:
			err := fmt.Sprintf("Invalid argument \"%s\" for \"--status\" flag. Valid options are: open|closed|all", status)
			cliLogger.Error(err)
			cobra.CheckErr(err)

		}

		// pagination to retrieve all issues
		var prs []github.Issue
		for {
			page, current, err := client.Search.Issues(context.Background(), query, opt)

			// TODO: retry and gracefully degrade
			cobra.CheckErr(err)

			for _, issue := range page.Issues {
				prs = append(prs, *issue)
			}

			// check if no more pages before continuing pagination
			if current.NextPage == 0 {
				break
			}
			opt.Page = current.NextPage
		}

		// Let's turn this into some tabular data and render it out

		t := newTableWriter(cmd.OutOrStdout())
		t.AppendHeader(table.Row{"Pull Request", "Name", "Age", "Link"})

		for _, i := range prs {
			// The repo name is not a field on Issues, so we have to infer by
			// extracting from the RepositoryURL string
			s := strings.SplitAfter(*i.RepositoryURL, "https://api.github.com/repos/")
			repoName := s[len(s)-1]

			// let's format the pull request reference as "org/repo#number"
			prRef := text.FgCyan.Sprint(repoName + "#" + fmt.Sprint(*i.Number))

			// get a human-friendly age string (e.g., "1 month ago")
			age := timediff.TimeDiff(*i.CreatedAt)

			t.AppendRow(table.Row{prRef, *i.Title, age, *i.HTMLURL})
		}

		if csv {
			t.RenderCSV()
		} else {
			t.Render() // Pretty-print table
		}
	},
}

func init() {
	reportCmd.AddCommand(reportPRsCmd)

	reportPRsCmd.Flags().BoolVar(&csv, "csv", false, "Outputs data in CSV format")
	reportPRsCmd.Flags().StringVar(&author, "author", "app/hashicorp-copywrite", "Search for PRs created by a specific author")
	reportPRsCmd.Flags().StringVar(&status, "status", "open", "Filters on PR status, valid options are: open|closed|all")
}
