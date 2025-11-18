// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/hashicorp/copywrite/repodata"
	"github.com/spf13/cobra"
)

// Flag variables
var (
	fields           string
	fieldsArr        []string
	githubOrgToAudit string
)

// reportReposCmd represents the report command
var reportReposCmd = &cobra.Command{
	Use:   "repos",
	Short: "Reports on GitHub repos matching specific criteria",
	Long: `Reports on GitHub repos matching specific criteria

Outputs the fields you specify in a repodata.csv file in the working directory.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// validate flag input
		cmd.Println("Getting data... this might take a minute")
		var err error
		fieldsArr, err = repodata.ValidateInputFields(fields)
		if err != nil {
			cliLogger.Error("Error validating inputs", err)
		}
		cobra.CheckErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		// get all public repos under org
		unfilteredRepos, err := repodata.GetRepos(githubOrgToAudit)
		if err != nil {
			cliLogger.Error(fmt.Sprintf("Error retrieving public repos for the \"%v\" org", githubOrgToAudit), err)
		}
		cobra.CheckErr(err)

		// remove archived repos
		filteredRepos := repodata.FilterRepos(unfilteredRepos)

		// transform repos into a string map
		outputData, err := repodata.Transform(filteredRepos)
		if err != nil {
			cliLogger.Error("Error transforming repo data", err)
		}
		cobra.CheckErr(err)

		t := newTableWriter(cmd.OutOrStdout())
		t.AppendHeader(stringArrayToRow(fieldsArr))

		// Populate rows
		for _, r := range outputData {
			// Filter the row to just contain the fields we care about
			row := make([]interface{}, 0)
			for _, k := range fieldsArr {
				row = append(row, r[k])
			}

			t.AppendRow(row)
		}

		// Pretty-print the table
		t.Render()

		// Now let's render the CSV for backwards compatibility
		csvFile, err := os.Create("repodata.csv")
		if err != nil {
			cliLogger.Error("Error creating CSV of repo data", err)
		}
		cobra.CheckErr(err)

		t.SetOutputMirror(csvFile)
		t.RenderCSV()

		err = csvFile.Close()
		if err != nil {
			cliLogger.Error("Error closing file", err)
		}
		cobra.CheckErr(err)
	},
}

func init() {
	reportCmd.AddCommand(reportReposCmd)

	reportReposCmd.Flags().StringVarP(&fields, "fields", "f", "Name,License,HTMLURL", "Repo attributes you wish to report on")
	reportReposCmd.Flags().StringVar(&githubOrgToAudit, "github-org", "hashicorp", "Sets the target GitHub org who's repos you wish to audit")
}
