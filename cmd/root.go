// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"errors"
	"os"

	"github.com/hashicorp/copywrite/config"
	"github.com/hashicorp/copywrite/github/actions"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
)

var (
	// Relative path to the Copywrite HCL config, defaults to .copywrite.hcl
	cfgPath string

	// This is the global configuration struct you should use to reference anything
	// from the .copywrite.hcl conf
	conf = config.MustNew()

	// This is a global instance of the GitHub Actions core helper library
	gha = actions.New(rootCmd.OutOrStdout())

	// Named subsystem logger for copywrite-cli commands
	cliLogger hclog.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "copywrite",
	Short: "Utilities for managing copyright headers and license files",
	Long: `Copywrite provides utilities for managing copyright headers and license
files in HashiCorp repos.

You can use it to report on what licenses repos are using, add LICENSE files,
and add or validate the presence of copyright headers on source code files.`,
	Version: GetVersion(),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		// Attempt to publish a GitHub error annotation (if in GHA) before exiting
		gha.Error(actions.Annotation{Message: err.Error()})
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initLogger)

	// Let's group together the most commonly used commands in the help section
	rootCmd.AddGroup(&cobra.Group{
		ID:    "common",
		Title: "Common Commands:",
	})

	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", ".copywrite.hcl", "config file")

	// Let's make sure Cobra doesn't default to stderr
	rootCmd.SetOut(os.Stdout)
}

func initConfig() {
	// Load the .copywrite.hcl config file into the running config
	err := conf.LoadConfigFile(cfgPath)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	cobra.CheckErr(err)
}

func initLogger() {
	// Valid levels list: https://pkg.go.dev/github.com/hashicorp/go-hclog#Level
	logLevel := hclog.DefaultLevel

	// If we're running in GitHub Actions and runner debugging is enabled, let's
	// default to debug logging just to be extra friendly
	if os.Getenv("RUNNER_DEBUG") == "1" {
		logLevel = hclog.Debug
	}

	// If the `COPYWRITE_LOG_LEVEL` environment variable is explicitly set, let's
	// attempt to coerce the result into a proper level. If no matching level can
	// be found, hclog.LevelFromString() defaults to the "NoLevel" (a good thing)
	levelEnv, levelSet := os.LookupEnv("COPYWRITE_LOG_LEVEL")
	if levelSet {
		logLevel = hclog.LevelFromString(levelEnv)
	}

	hclog.Default().Named("cli")
	cliLogger = hclog.New(&hclog.LoggerOptions{
		Name:   "cli",
		Level:  logLevel,
		Color:  hclog.AutoColor,
		Output: os.Stdout,
	})
}
