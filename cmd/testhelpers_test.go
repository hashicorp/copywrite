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
