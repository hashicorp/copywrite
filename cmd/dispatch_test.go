// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchCmd_Flags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		defValue string
	}{
		{name: "plan flag", flagName: "plan", defValue: "false"},
		{name: "max-attempts flag", flagName: "max-attempts", defValue: fmt.Sprintf("%d", defaultDispatchMaxAttempts)},
		{name: "sleep flag", flagName: "sleep", defValue: fmt.Sprintf("%d", defaultDispatchSleep)},
		{name: "workers flag", flagName: "workers", defValue: fmt.Sprintf("%d", defaultDispatchWorkers)},
		{name: "branch flag", flagName: "branch", defValue: defaultDispatchBranch},
		{name: "batch-id flag", flagName: "batch-id", defValue: defaultDispatchBatchID},
		{name: "workflow flag", flagName: "workflow", defValue: defaultDispatchWorkflow},
		{name: "github-org flag", flagName: "github-org", defValue: defaultDispatchGitHubOrg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := dispatchCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %q should exist", tt.flagName)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestDispatchCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	dispatchCmd.SetOut(buf)
	dispatchCmd.SetErr(buf)
	t.Cleanup(func() {
		dispatchCmd.SetOut(nil)
		dispatchCmd.SetErr(nil)
	})

	err := dispatchCmd.Help()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Dispatches audit jobs")
	assert.Contains(t, output, "--plan")
	assert.Contains(t, output, "--workers")
	assert.Contains(t, output, "--branch")
}

func TestDispatchCmd_RegisteredUnderRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "dispatch" {
			found = true
			break
		}
	}
	assert.True(t, found, "dispatch command should be registered under root")
}

func TestDispatchCmd_FlagShortHands(t *testing.T) {
	tests := []struct {
		name      string
		flagName  string
		shorthand string
	}{
		{name: "sleep shorthand", flagName: "sleep", shorthand: "s"},
		{name: "workers shorthand", flagName: "workers", shorthand: "w"},
		{name: "branch shorthand", flagName: "branch", shorthand: "b"},
		{name: "batch-id shorthand", flagName: "batch-id", shorthand: "i"},
		{name: "workflow shorthand", flagName: "workflow", shorthand: "n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := dispatchCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag)
			assert.Equal(t, tt.shorthand, flag.Shorthand)
		})
	}
}
