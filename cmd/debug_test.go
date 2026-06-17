// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugCmd_Flags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		defValue string
	}{
		{name: "dirPath flag", flagName: "dirPath", defValue: "."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := debugCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %q should exist", tt.flagName)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestDebugCmd_Help(t *testing.T) {
	restoreDebugCmd(t)
	buf := new(bytes.Buffer)
	debugCmd.SetOut(buf)
	debugCmd.SetErr(buf)

	err := debugCmd.Help()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Prints information to help debug issues")
	assert.Contains(t, output, "--dirPath")
	assert.Contains(t, output, "Copywrite Version")
}

func TestDebugCmd_RegisteredUnderRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "debug" {
			found = true
			break
		}
	}
	assert.True(t, found, "debug command should be registered under root")
}

func TestDebugCmd_RunsWithCurrentDir(t *testing.T) {
	// This test focuses on ensuring the PreRun path is safe with default settings.
	initLogger()
	oldDirPath := dirPath
	t.Cleanup(func() { dirPath = oldDirPath })
	dirPath = "."
	require.NotNil(t, debugCmd.PreRun)
	assert.NotPanics(t, func() {
		debugCmd.PreRun(debugCmd, []string{})
	})
}
