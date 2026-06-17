// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// newGitRepo creates a temporary directory, initialises a git repository inside
// it, makes a placeholder initial commit dated commitDate, and returns the path.
func newGitRepo(t *testing.T, commitDate time.Time) string {
	t.Helper()
	tmpDir := t.TempDir()

	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, out)
	}

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".gitkeep"), []byte(""), 0644))
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	dateStr := commitDate.Format("2006-01-02T15:04:05-07:00")
	cmd = exec.Command("git", "commit", "-m", "initial commit", "--date", dateStr)
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE="+dateStr)
	require.NoError(t, cmd.Run())

	return tmpDir
}

// gitAddCommit stages all files and creates a commit in dir.
func gitAddCommit(t *testing.T, dir, message string) {
	t.Helper()
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
}

// restoreDebugCmd saves debugCmd's current output/error writers and schedules
// their restoration via t.Cleanup, preventing tests from leaking writer state
// into each other.
func restoreDebugCmd(t *testing.T) {
	t.Helper()
	origOut, origErr := debugCmd.OutOrStdout(), debugCmd.ErrOrStderr()
	t.Cleanup(func() {
		debugCmd.SetOut(origOut)
		debugCmd.SetErr(origErr)
	})
}

// restoreHeadersCmd saves headersCmd's current output/error writers and
// schedules their restoration via t.Cleanup, preventing tests from leaking
// writer state into each other.
func restoreHeadersCmd(t *testing.T) {
	t.Helper()
	origOut, origErr := headersCmd.OutOrStdout(), headersCmd.ErrOrStderr()
	t.Cleanup(func() {
		headersCmd.SetOut(origOut)
		headersCmd.SetErr(origErr)
	})
}

// restoreReportCmd saves reportCmd's current output/error writers and schedules
// their restoration via t.Cleanup, preventing tests from leaking writer state
// into each other.
func restoreReportCmd(t *testing.T) {
	t.Helper()
	origOut, origErr := reportCmd.OutOrStdout(), reportCmd.ErrOrStderr()
	t.Cleanup(func() {
		reportCmd.SetOut(origOut)
		reportCmd.SetErr(origErr)
	})
}

// restoreReportPRsCmd saves reportPRsCmd's current output/error writers and
// schedules their restoration via t.Cleanup, preventing tests from leaking
// writer state into each other.
func restoreReportPRsCmd(t *testing.T) {
	t.Helper()
	origOut, origErr := reportPRsCmd.OutOrStdout(), reportPRsCmd.ErrOrStderr()
	t.Cleanup(func() {
		reportPRsCmd.SetOut(origOut)
		reportPRsCmd.SetErr(origErr)
	})
}

// restoreReportReposCmd saves reportReposCmd's current output/error writers and
// schedules their restoration via t.Cleanup, preventing tests from leaking
// writer state into each other.
func restoreReportReposCmd(t *testing.T) {
	t.Helper()
	origOut, origErr := reportReposCmd.OutOrStdout(), reportReposCmd.ErrOrStderr()
	t.Cleanup(func() {
		reportReposCmd.SetOut(origOut)
		reportReposCmd.SetErr(origErr)
	})
}

// restoreInitCmd saves initCmd's current output/error writers and schedules
// their restoration via t.Cleanup, preventing tests from leaking writer state
// into each other. It also resets SetArgs to nil on cleanup.
func restoreInitCmd(t *testing.T) {
	t.Helper()
	origOut, origErr := initCmd.OutOrStdout(), initCmd.ErrOrStderr()
	t.Cleanup(func() {
		initCmd.SetOut(origOut)
		initCmd.SetErr(origErr)
		initCmd.SetArgs(nil)
	})
}

// restoreRootCmd saves rootCmd's current output/error writers and schedules
// their restoration via t.Cleanup. It also resets SetArgs to nil on cleanup.
func restoreRootCmd(t *testing.T) {
	t.Helper()
	origOut, origErr := rootCmd.OutOrStdout(), rootCmd.ErrOrStderr()
	t.Cleanup(func() {
		rootCmd.SetOut(origOut)
		rootCmd.SetErr(origErr)
		rootCmd.SetArgs(nil)
	})
}
