// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_determineLicenseCopyrightYears(t *testing.T) {
	currentYear := time.Now().Year()

	// Save and restore conf
	oldYear := conf.Project.CopyrightYear
	defer func() { conf.Project.CopyrightYear = oldYear }()

	tests := []struct {
		name          string
		copyrightYear int
		setupDir      func(t *testing.T) string
		validate      func(t *testing.T, result string)
	}{
		{
			name:          "with configured year matching current year returns single year",
			copyrightYear: currentYear,
			setupDir: func(t *testing.T) string {
				return setupGitRepo(t, time.Now())
			},
			validate: func(t *testing.T, result string) {
				assert.Equal(t, strconv.Itoa(currentYear), result)
			},
		},
		{
			name:          "with configured year earlier than last commit returns range",
			copyrightYear: 2019,
			setupDir: func(t *testing.T) string {
				return setupGitRepo(t, time.Now())
			},
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "2019")
				assert.Contains(t, result, ",")
			},
		},
		{
			name:          "without configured year and non-git dir falls back to current year",
			copyrightYear: 0,
			setupDir: func(t *testing.T) string {
				return t.TempDir()
			},
			validate: func(t *testing.T, result string) {
				assert.Equal(t, strconv.Itoa(currentYear), result)
			},
		},
		{
			name:          "without configured year in git repo detects from git",
			copyrightYear: 0,
			setupDir: func(t *testing.T) string {
				return setupGitRepo(t, time.Now())
			},
			validate: func(t *testing.T, result string) {
				assert.NotEmpty(t, result)
				// Should contain the current year (since the commit is from now)
				assert.Contains(t, result, strconv.Itoa(currentYear))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf.Project.CopyrightYear = tt.copyrightYear
			dir := tt.setupDir(t)
			result := determineLicenseCopyrightYears(dir)
			assert.NotEmpty(t, result)
			tt.validate(t, result)
		})
	}
}

func setupGitRepo(t *testing.T, commitDate time.Time) string {
	t.Helper()
	tmpDir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, out)
	}

	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main"), 0644)
	require.NoError(t, err)

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

func TestLicenseCmd_Flags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		defValue string
	}{
		{name: "dirPath flag", flagName: "dirPath", defValue: "."},
		{name: "plan flag", flagName: "plan", defValue: "false"},
		{name: "year flag", flagName: "year", defValue: "0"},
		{name: "spdx flag", flagName: "spdx", defValue: ""},
		{name: "copyright-holder flag", flagName: "copyright-holder", defValue: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := licenseCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %q should exist", tt.flagName)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestLicenseCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	licenseCmd.SetOut(buf)
	licenseCmd.SetErr(buf)

	err := licenseCmd.Help()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Validates that a LICENSE file is present")
	assert.Contains(t, output, "--dirPath")
	assert.Contains(t, output, "--plan")
}

func Test_determineLicenseCopyrightYears_WithMultipleCommits(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, out)
	}

	// First commit (old)
	testFile := filepath.Join(tmpDir, "old.go")
	err := os.WriteFile(testFile, []byte("package old"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "first commit", "--date", "2019-06-15T00:00:00+00:00")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE=2019-06-15T00:00:00+00:00")
	require.NoError(t, cmd.Run())

	// Second commit (recent but still in the past)
	testFile2 := filepath.Join(tmpDir, "new.go")
	err = os.WriteFile(testFile2, []byte("package new"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Use current year for the second commit so the test is stable
	now := time.Now()
	dateStr := now.Format("2006-01-02T15:04:05-07:00")
	cmd = exec.Command("git", "commit", "-m", "second commit", "--date", dateStr)
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE="+dateStr)
	require.NoError(t, cmd.Run())

	oldYear := conf.Project.CopyrightYear
	defer func() { conf.Project.CopyrightYear = oldYear }()

	conf.Project.CopyrightYear = 2019
	result := determineLicenseCopyrightYears(tmpDir)
	// Should be a range from 2019 to current year
	currentYear := now.Year()
	expected := fmt.Sprintf("2019, %d", currentYear)
	assert.Equal(t, expected, result)
}

func TestLicenseCmd_Run_WithExistingLicense(t *testing.T) {
	tmpDir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())
	}

	// Create a LICENSE file with a copyright format that matches what the command will expect
	currentYear := time.Now().Year()
	licenseContent := fmt.Sprintf("Copyright Test Corp. %d\n\nMPL-2.0 License\n", currentYear)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "LICENSE"), []byte(licenseContent), 0644))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n"), 0644))

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	t.Chdir(tmpDir)

	oldConf := *conf
	t.Cleanup(func() { *conf = oldConf })

	oldPlan := plan
	t.Cleanup(func() { plan = oldPlan })
	plan = false

	origOut, origErr := rootCmd.OutOrStdout(), rootCmd.ErrOrStderr()
	t.Cleanup(func() { rootCmd.SetOut(origOut); rootCmd.SetErr(origErr) })
	origLicOut, origLicErr := licenseCmd.OutOrStdout(), licenseCmd.ErrOrStderr()
	t.Cleanup(func() { licenseCmd.SetOut(origLicOut); licenseCmd.SetErr(origLicErr) })

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	licenseCmd.SetOut(buf)
	licenseCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"license", fmt.Sprintf("--year=%d", currentYear), "--spdx", "MPL-2.0", "--copyright-holder", "Test Corp.", "--dirPath", "."})

	// Exercises the Run path - may succeed or fail depending on copyright format matching
	err := rootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Copyright statement is valid!")
}

func TestLicenseCmd_Run_Plan_MissingLicense(t *testing.T) {
	// cobra.CheckErr inside Run calls os.Exit, so test in subprocess
	if os.Getenv("TEST_LICENSE_PLAN_MISSING") == "1" {
		tmpDir := t.TempDir()

		for _, args := range [][]string{
			{"git", "init"},
			{"git", "config", "user.email", "test@test.com"},
			{"git", "config", "user.name", "Test"},
		} {
			c := exec.Command(args[0], args[1:]...)
			c.Dir = tmpDir
			require.NoError(t, c.Run())
		}

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n"), 0644))
		c := exec.Command("git", "add", ".")
		c.Dir = tmpDir
		require.NoError(t, c.Run())
		c = exec.Command("git", "commit", "-m", "init")
		c.Dir = tmpDir
		require.NoError(t, c.Run())

		t.Chdir(tmpDir)
		rootCmd.SetArgs([]string{"license", "--plan", "--year", "2023", "--spdx", "MPL-2.0"})
		if err := rootCmd.Execute(); err != nil {
			t.Logf("execute: %v", err)
		}
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestLicenseCmd_Run_Plan_MissingLicense", "-test.count=1")
	cmd.Env = append(os.Environ(), "TEST_LICENSE_PLAN_MISSING=1")
	output, err := cmd.CombinedOutput()
	assert.Error(t, err)
	assert.Contains(t, string(output), "missing license file")
}

func TestLicenseCmd_Run_MultipleLicenseFiles(t *testing.T) {
	// cobra.CheckErr inside Run calls os.Exit, so test in subprocess
	if os.Getenv("TEST_LICENSE_MULTIPLE") == "1" {
		tmpDir := t.TempDir()

		for _, args := range [][]string{
			{"git", "init"},
			{"git", "config", "user.email", "test@test.com"},
			{"git", "config", "user.name", "Test"},
		} {
			c := exec.Command(args[0], args[1:]...)
			c.Dir = tmpDir
			require.NoError(t, c.Run())
		}

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "LICENSE"), []byte("license1\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "LICENSE.txt"), []byte("license2\n"), 0644))
		c := exec.Command("git", "add", ".")
		c.Dir = tmpDir
		require.NoError(t, c.Run())
		c = exec.Command("git", "commit", "-m", "init")
		c.Dir = tmpDir
		require.NoError(t, c.Run())

		t.Chdir(tmpDir)
		rootCmd.SetArgs([]string{"license", "--year", "2023", "--spdx", "MPL-2.0"})
		if err := rootCmd.Execute(); err != nil {
			t.Logf("execute: %v", err)
		}
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestLicenseCmd_Run_MultipleLicenseFiles", "-test.count=1")
	cmd.Env = append(os.Environ(), "TEST_LICENSE_MULTIPLE=1")
	output, err := cmd.CombinedOutput()
	assert.Error(t, err)
	assert.Contains(t, string(output), "more than one license file")
}

func TestLicenseCmd_Run_CreatesLicenseFile(t *testing.T) {
	tmpDir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())
	}

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n"), 0644))
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	t.Chdir(tmpDir)

	oldConf := *conf
	defer func() { *conf = oldConf }()
	conf.Project.License = "MPL-2.0"
	conf.Project.CopyrightYear = 2023
	conf.Project.CopyrightHolder = "Test Corp."

	// Reset the global plan variable to avoid pollution from prior tests
	oldPlan := plan
	defer func() { plan = oldPlan }()
	plan = false

	origOut, origErr := rootCmd.OutOrStdout(), rootCmd.ErrOrStderr()
	defer func() { rootCmd.SetOut(origOut); rootCmd.SetErr(origErr) }()
	origLicOut, origLicErr := licenseCmd.OutOrStdout(), licenseCmd.ErrOrStderr()
	defer func() { licenseCmd.SetOut(origLicOut); licenseCmd.SetErr(origLicErr) }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	licenseCmd.SetOut(buf)
	licenseCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"license", "--dirPath", ".", "--year", "2023", "--spdx", "MPL-2.0", "--copyright-holder", "Test Corp."})

	err := rootCmd.Execute()
	require.NoError(t, err)
	_, statErr := os.Stat(filepath.Join(tmpDir, "LICENSE"))
	assert.NoError(t, statErr, "LICENSE file should be created")
}
