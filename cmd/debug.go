// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"context"
	"os"
	"path/filepath"

	"github.com/hashicorp/copywrite/github"
	"github.com/hashicorp/go-hclog"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mergestat/timediff"
	"github.com/spf13/cobra"
)

// debugCmd represents the debug command
var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Prints env-specific debug information about copywrite",
	Long: `Prints information to help debug issues, including:
- Copywrite Version
- Running configuration
- Current GitHub repo (if one is detected)
- GitHub authentication status`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// Change directory if needed
		if dirPath != "." {
			err := os.Chdir(dirPath)
			cobra.CheckErr(err)
		}

		// Let's forcibly enable trace-level logging
		cliLogger.SetLevel(hclog.Trace)
	},
	Run: func(cmd *cobra.Command, args []string) {
		title := func(t string) {
			escaped := colorize(t, text.FgCyan, text.Bold)
			cmd.Println(escaped)
		}

		//
		// Print version info
		//
		title("Copywrite Version:")
		version := GetVersion()
		cmd.Printf("%v\n\n", version)

		//
		// Print working directory info
		//
		title("Working Directory:")
		if dirPath != "." {
			cmd.Print("The working directory was overwritten with the --dirPath flag\n")
		}
		absDirPath, _ := filepath.Abs(dirPath)
		cmd.Printf("Directory path: %v\n\n", absDirPath)

		//
		// Print info relating to any configuration file found
		//
		title("Copywrite Configuration File:")
		path := conf.GetConfigPath()
		cmd.Printf("Configuration file path: %s\n", path)
		if _, err := os.Stat(path); err == nil {
			cmd.Print("✔️ Config file exists\n\n")
		} else {
			cmd.Print("❌ File does not exist\n\n")
		}

		//
		// Print running config
		//
		title("Running Config:")
		runningConfigString := conf.Sprint()
		cmd.Printf("%v\n", runningConfigString)

		//
		// Print GitHub Actions/CI Information
		//
		title("GitHub Actions:")
		if gha.IsGHA() {
			cmd.Print("Current execution environment is GitHub Actions\n\n")
		} else {
			cmd.Print("Current execution environment is NOT GitHub Actions\n\n")
		}

		//
		// Print any GitHub repo that is discovered
		//
		title("Current GitHub Repo:")
		repo, err := github.DiscoverRepo()
		if err != nil {
			cmd.Println(err)
		} else {
			cmd.Printf("GitHub Org:\t%v\n", repo.Owner)
			cmd.Printf("GitHub Repo:\t%v\n", repo.Name)
		}
		cmd.Println()

		//
		// Attempt to auth to GitHub and print any relevant info
		//
		title("Attempting GitHub Authentication:")
		ghc := github.NewGHClient().Raw()

		user, _, _ := ghc.Users.Get(context.Background(), "")
		cmd.Printf("Running as authenticated user: %v (@%v)\n", user.GetName(), user.GetLogin())

		rateLimits, _, _ := ghc.RateLimits(context.Background())
		cmd.Printf("GitHub API rate limits: %v/%v remaining\n", rateLimits.Core.Remaining, rateLimits.Core.Limit)
		cmd.Printf("GitHub API rate limits will reset at: %v (%v)\n", rateLimits.Core.Reset, timediff.TimeDiff(rateLimits.Core.Reset.Time))
	},
}

func init() {
	rootCmd.AddCommand(debugCmd)

	// These flags are only locally relevant
	debugCmd.Flags().StringVarP(&dirPath, "dirPath", "d", ".", "Path to the directory in which you wish to introspect")
}
