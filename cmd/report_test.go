// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportCmd_RegisteredUnderRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "report" {
			found = true
			break
		}
	}
	assert.True(t, found, "report command should be registered under root")
}

func TestReportCmd_Help(t *testing.T) {
	restoreReportCmd(t)
	buf := new(bytes.Buffer)
	reportCmd.SetOut(buf)
	reportCmd.SetErr(buf)

	err := reportCmd.Help()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "audit subcommands")
}

func TestReportCmd_HasSubcommands(t *testing.T) {
	cmds := reportCmd.Commands()
	names := make([]string, 0, len(cmds))
	for _, c := range cmds {
		names = append(names, c.Use)
	}
	assert.Contains(t, names, "prs")
	assert.Contains(t, names, "repos")
}

func TestReportPRsCmd_Flags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		defValue string
	}{
		{name: "csv flag", flagName: "csv", defValue: "false"},
		{name: "author flag", flagName: "author", defValue: "app/hashicorp-copywrite"},
		{name: "status flag", flagName: "status", defValue: "open"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := reportPRsCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %q should exist", tt.flagName)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestReportPRsCmd_Help(t *testing.T) {
	restoreReportPRsCmd(t)
	buf := new(bytes.Buffer)
	reportPRsCmd.SetOut(buf)
	reportPRsCmd.SetErr(buf)

	err := reportPRsCmd.Help()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Lists all unmerged compliance pull requests")
	assert.Contains(t, output, "--csv")
	assert.Contains(t, output, "--author")
	assert.Contains(t, output, "--status")
}

func TestReportReposCmd_Flags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		defValue string
	}{
		{name: "fields flag", flagName: "fields", defValue: "Name,License,HTMLURL"},
		{name: "github-org flag", flagName: "github-org", defValue: "hashicorp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := reportReposCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %q should exist", tt.flagName)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestReportReposCmd_Help(t *testing.T) {
	restoreReportReposCmd(t)
	buf := new(bytes.Buffer)
	reportReposCmd.SetOut(buf)
	reportReposCmd.SetErr(buf)

	err := reportReposCmd.Help()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Reports on GitHub repos matching specific criteria")
	assert.Contains(t, output, "--fields")
	assert.Contains(t, output, "--github-org")
	assert.Contains(t, output, "repodata.csv")
}

func TestReportReposCmd_FlagShortHands(t *testing.T) {
	flag := reportReposCmd.Flags().Lookup("fields")
	require.NotNil(t, flag)
	assert.Equal(t, "f", flag.Shorthand)
}
