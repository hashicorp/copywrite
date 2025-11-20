// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

var csv bool

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Performs a variety of reporting tasks",
	Long:  `Use the audit subcommands to retrieve reports such as unlicensed repos, outstanding pull requests, and more`,
	// Run function is omitted, as this command exists only to house subcommands
}

func init() {
	rootCmd.AddCommand(reportCmd)
}
