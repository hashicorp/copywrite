// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetRootCmd restores rootCmd output and args to defaults after a test.
func resetRootCmd(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
}

func TestRootCmd_VersionFlag(t *testing.T) {
	resetRootCmd(t)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, GetVersion())
}

func TestRootCmd_HelpFlag(t *testing.T) {
	resetRootCmd(t)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Copywrite provides utilities")
	assert.Contains(t, output, "copywrite")
}

func TestRootCmd_UnknownCommand(t *testing.T) {
	resetRootCmd(t)
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"nonexistentcommand"})

	err := rootCmd.Execute()
	assert.Error(t, err)
}

func TestRootCmd_ConfigFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, flag)
	assert.Equal(t, ".copywrite.hcl", flag.DefValue)
}

func Test_initConfig_FileNotExist(t *testing.T) {
	oldCfgPath := cfgPath
	defer func() { cfgPath = oldCfgPath }()

	cfgPath = "/tmp/nonexistent_copywrite_test_config_12345.hcl"
	initConfig()
}

func Test_initConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".copywrite.hcl")

	content := []byte(`schema_version = 1

project {
  license        = "MPL-2.0"
  copyright_year = 2023
}
`)
	err := os.WriteFile(configPath, content, 0644)
	require.NoError(t, err)

	oldCfgPath := cfgPath
	defer func() { cfgPath = oldCfgPath }()

	cfgPath = configPath
	initConfig()
}

func Test_initLogger_DefaultLevel(t *testing.T) {
	t.Setenv("RUNNER_DEBUG", "")
	t.Setenv("COPYWRITE_LOG_LEVEL", "")

	initLogger()
	require.NotNil(t, cliLogger)
}

func Test_initLogger_RunnerDebug(t *testing.T) {
	t.Setenv("RUNNER_DEBUG", "1")
	t.Setenv("COPYWRITE_LOG_LEVEL", "")

	initLogger()
	require.NotNil(t, cliLogger)
}

func Test_initLogger_CopywriteLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{name: "trace level", level: "TRACE"},
		{name: "debug level", level: "DEBUG"},
		{name: "info level", level: "INFO"},
		{name: "warn level", level: "WARN"},
		{name: "error level", level: "ERROR"},
		{name: "invalid level defaults to NoLevel", level: "INVALID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("COPYWRITE_LOG_LEVEL", tt.level)
			t.Setenv("RUNNER_DEBUG", "")

			initLogger()
			require.NotNil(t, cliLogger)
		})
	}
}

func Test_initLogger_CopywriteLogLevelOverridesRunnerDebug(t *testing.T) {
	t.Setenv("RUNNER_DEBUG", "1")
	t.Setenv("COPYWRITE_LOG_LEVEL", "ERROR")

	initLogger()
	require.NotNil(t, cliLogger)
}
