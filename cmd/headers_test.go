// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/copywrite/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initGitRepo initializes a git repository in dir with a dummy commit.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
	}
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

// withTestConfig saves the current global conf, applies modifications via fn,
// and restores the original config when the test completes.
func withTestConfig(t *testing.T, fn func(c *config.Config)) {
	t.Helper()
	oldConf := *conf
	t.Cleanup(func() { *conf = oldConf })
	fn(conf)
}

func TestHeadersCmd_Flags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		defValue string
	}{
		{name: "dirPath flag", flagName: "dirPath", defValue: "."},
		{name: "plan flag", flagName: "plan", defValue: "false"},
		{name: "spdx flag", flagName: "spdx", defValue: ""},
		{name: "copyright-holder flag", flagName: "copyright-holder", defValue: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := headersCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %q should exist", tt.flagName)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestHeadersCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	headersCmd.SetOut(buf)
	headersCmd.SetErr(buf)

	err := headersCmd.Help()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Recursively checks for all files")
	assert.Contains(t, output, "--plan")
	assert.Contains(t, output, "--spdx")
	assert.Contains(t, output, "--dirPath")
}

func Test_updateExistingHeaders_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create a dummy commit so git log works
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dummy.txt"), []byte("dummy"), 0644))
	gitAddCommit(t, tmpDir, "init")

	t.Chdir(tmpDir)

	withTestConfig(t, func(c *config.Config) {
		c.Project.CopyrightHolder = "Test Corp."
		c.Project.CopyrightYear = 2023
		c.Project.IgnoreYear1 = false
	})

	buf := new(bytes.Buffer)
	headersCmd.SetOut(buf)

	count, anyUpdated, licensePath := updateExistingHeaders(headersCmd, []string{}, true)
	assert.Equal(t, 0, count)
	assert.False(t, anyUpdated)
	assert.Empty(t, licensePath)
}

func Test_updateExistingHeaders_WithLicenseFile(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create LICENSE file
	licenseContent := "Copyright Test Corp. 2023\nMIT License\n"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "LICENSE"), []byte(licenseContent), 0644))
	gitAddCommit(t, tmpDir, "init")

	t.Chdir(tmpDir)

	withTestConfig(t, func(c *config.Config) {
		c.Project.CopyrightHolder = "Test Corp."
		c.Project.CopyrightYear = 2023
	})

	buf := new(bytes.Buffer)
	headersCmd.SetOut(buf)

	_, _, licensePath := updateExistingHeaders(headersCmd, []string{}, true)
	assert.Equal(t, "LICENSE", licensePath)
}

func Test_updateExistingHeaders_SkipsIgnoredPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create files in vendor directory
	vendorDir := filepath.Join(tmpDir, "vendor")
	require.NoError(t, os.MkdirAll(vendorDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(vendorDir, "lib.go"), []byte("// Copyright Test Corp. 2019\npackage vendor"), 0644))

	// Create a regular file with a start year that differs from configYear
	// so that the update is detected via start-year mismatch (Condition 1)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("// Copyright Test Corp. 2019\npackage main"), 0644))
	gitAddCommit(t, tmpDir, "init")

	t.Chdir(tmpDir)

	withTestConfig(t, func(c *config.Config) {
		c.Project.CopyrightHolder = "Test Corp."
		c.Project.CopyrightYear = 2020
		c.Project.IgnoreYear1 = false
	})

	buf := new(bytes.Buffer)
	headersCmd.SetOut(buf)

	ignoredPatterns := []string{"vendor/**"}
	count, _, _ := updateExistingHeaders(headersCmd, ignoredPatterns, true)
	// Only main.go should be counted; the vendor file must be excluded
	assert.Equal(t, 1, count)
}

func Test_updateLicenseFile_EmptyPath(t *testing.T) {
	buf := new(bytes.Buffer)
	headersCmd.SetOut(buf)

	// Should return immediately with no output
	updateLicenseFile(headersCmd, "", true, false)
	assert.Empty(t, buf.String())
}

func Test_updateLicenseFile_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create a LICENSE file with copyright
	licenseContent := "Copyright Test Corp. 2020\n\nMIT License text here\n"
	licensePath := filepath.Join(tmpDir, "LICENSE")
	require.NoError(t, os.WriteFile(licensePath, []byte(licenseContent), 0644))
	gitAddCommit(t, tmpDir, "init")

	t.Chdir(tmpDir)

	withTestConfig(t, func(c *config.Config) {
		c.Project.CopyrightHolder = "Test Corp."
		c.Project.CopyrightYear = 2020
	})

	buf := new(bytes.Buffer)
	headersCmd.SetOut(buf)

	updateLicenseFile(headersCmd, licensePath, true, true)
	// In dry run, file should not be modified
	content, err := os.ReadFile(licensePath)
	require.NoError(t, err)
	assert.Equal(t, licenseContent, string(content))
}

func Test_updateLicenseFile_UpdatesYear(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name            string
		initialContent  string
		configHolder    string
		configYear      int
		expectedContent string
	}{
		{
			name:            "single year updated to range with current year",
			initialContent:  "Copyright Test Corp. 2023\n\nLicense text\n",
			configHolder:    "Test Corp.",
			configYear:      2023,
			expectedContent: fmt.Sprintf("Copyright Test Corp. 2023, %d\n\nLicense text\n", currentYear),
		},
		{
			name:            "existing range end year updated to current year",
			initialContent:  "Copyright Test Corp. 2023, 2025\n\nLicense text\n",
			configHolder:    "Test Corp.",
			configYear:      2023,
			expectedContent: fmt.Sprintf("Copyright Test Corp. 2023, %d\n\nLicense text\n", currentYear),
		},
		{
			name:            "config year changes start year and end year updates to current",
			initialContent:  fmt.Sprintf("Copyright Test Corp. 2023, %d\n\nLicense text\n", currentYear),
			configHolder:    "Test Corp.",
			configYear:      2024,
			expectedContent: fmt.Sprintf("Copyright Test Corp. 2024, %d\n\nLicense text\n", currentYear),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			initGitRepo(t, tmpDir)

			licensePath := filepath.Join(tmpDir, "LICENSE")
			require.NoError(t, os.WriteFile(licensePath, []byte(tt.initialContent), 0644))
			gitAddCommit(t, tmpDir, "init")

			t.Chdir(tmpDir)

			withTestConfig(t, func(c *config.Config) {
				c.Project.CopyrightHolder = tt.configHolder
				c.Project.CopyrightYear = tt.configYear
			})

			buf := new(bytes.Buffer)
			headersCmd.SetOut(buf)

			// anyFileUpdated=true forces current year update
			updateLicenseFile(headersCmd, licensePath, true, false)

			content, err := os.ReadFile(licensePath)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedContent, string(content))
		})
	}
}

func TestHeadersCmd_PlanMode_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create a .copywrite.hcl so the command doesn't fail on config load
	configContent := `schema_version = 1

project {
  license        = "MPL-2.0"
  copyright_year = 2023
  copyright_holder = "Test Corp."
  header_ignore = []
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".copywrite.hcl"), []byte(configContent), 0644))

	// Create a Go source file that ALREADY has a proper copyright header with current year
	goFile := filepath.Join(tmpDir, "main.go")
	goContent := fmt.Sprintf("// Copyright (c) Test Corp. 2023, %d\n// SPDX-License-Identifier: MPL-2.0\n\npackage main\n\nfunc main() {}\n", time.Now().Year())
	require.NoError(t, os.WriteFile(goFile, []byte(goContent), 0644))
	gitAddCommit(t, tmpDir, "init")

	t.Chdir(tmpDir)

	// Reset dirPath to avoid stale state from other tests
	oldDirPath := dirPath
	defer func() { dirPath = oldDirPath }()
	dirPath = "."

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	headersCmd.SetOut(buf)
	headersCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"headers", "--plan", "--spdx", "MPL-2.0", "--copyright-holder", "Test Corp."})

	_ = rootCmd.Execute()
	output := buf.String()
	assert.Contains(t, output, "dry-run mode")
}

func TestHeadersCmd_Run_WithEmptyLicense(t *testing.T) {
	// When no --spdx is specified, addlicense.Run in plan mode may still find a missing
	// header and cobra.CheckErr will call os.Exit. Use subprocess testing pattern.
	if os.Getenv("TEST_HEADERS_EMPTY_LICENSE") == "1" {
		tmpDir := t.TempDir()
		initGitRepo(t, tmpDir)

		goContent := fmt.Sprintf("// Copyright (c) Test Corp. 2023, %d\n\npackage main\n\nfunc main() {}\n", time.Now().Year())
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goContent), 0644))
		gitAddCommit(t, tmpDir, "init")

		t.Chdir(tmpDir)
		rootCmd.SetArgs([]string{"headers", "--plan", "--copyright-holder", "Test Corp."})
		if err := rootCmd.Execute(); err != nil {
			t.Logf("execute: %v", err)
		}
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHeadersCmd_Run_WithEmptyLicense", "-test.count=1")
	cmd.Env = append(os.Environ(), "TEST_HEADERS_EMPTY_LICENSE=1")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	assert.Contains(t, outputStr, "--spdx flag was not specified")
	_ = err
}

func TestHeadersCmd_Run_WithHeaderIgnore(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create a .copywrite.hcl with header_ignore
	configContent := `schema_version = 1

project {
  license        = "MPL-2.0"
  copyright_year = 2023
  copyright_holder = "Test Corp."
  header_ignore = [
    "vendor/**",
    "**autogen**",
  ]
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".copywrite.hcl"), []byte(configContent), 0644))

	goFile := filepath.Join(tmpDir, "main.go")
	goContent := fmt.Sprintf("// Copyright (c) Test Corp. 2023, %d\n// SPDX-License-Identifier: MPL-2.0\n\npackage main\n\nfunc main() {}\n", time.Now().Year())
	require.NoError(t, os.WriteFile(goFile, []byte(goContent), 0644))

	// Create files that match header_ignore patterns (should not get headers)
	vendorDir := filepath.Join(tmpDir, "vendor")
	require.NoError(t, os.MkdirAll(vendorDir, 0755))
	vendorFile := filepath.Join(vendorDir, "lib.go")
	vendorContent := "package lib\n\nfunc Lib() {}\n"
	require.NoError(t, os.WriteFile(vendorFile, []byte(vendorContent), 0644))

	autogenFile := filepath.Join(tmpDir, "foo_autogen_bar.go")
	autogenContent := "package main\n\nfunc Generated() {}\n"
	require.NoError(t, os.WriteFile(autogenFile, []byte(autogenContent), 0644))
	gitAddCommit(t, tmpDir, "init")

	t.Chdir(tmpDir)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	headersCmd.SetOut(buf)
	headersCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"headers", "--spdx", "MPL-2.0", "--copyright-holder", "Test Corp."})

	_ = rootCmd.Execute()

	output := buf.String()

	// Verify all header_ignore patterns are reported in output
	expectedPatterns := []string{"vendor/**", "**autogen**"}
	for _, pattern := range expectedPatterns {
		assert.Contains(t, output, pattern, "expected ignore pattern %q in output", pattern)
	}

	// Verify ignored files were NOT modified (no copyright header added)
	vendorActual, err := os.ReadFile(vendorFile)
	require.NoError(t, err)
	assert.Equal(t, vendorContent, string(vendorActual), "vendor file should not have been modified")

	autogenActual, err := os.ReadFile(autogenFile)
	require.NoError(t, err)
	assert.Equal(t, autogenContent, string(autogenActual), "autogen file should not have been modified")
}

func Test_updateLicenseFile_ReadOnlyFile(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create a LICENSE file with an outdated year
	licenseContent := "Copyright Test Corp. 2020\n\nLicense text\n"
	licensePath := filepath.Join(tmpDir, "LICENSE")
	require.NoError(t, os.WriteFile(licensePath, []byte(licenseContent), 0644))
	gitAddCommit(t, tmpDir, "init")

	// Make the LICENSE file read-only so the write fails
	require.NoError(t, os.Chmod(licensePath, 0444))
	t.Cleanup(func() {
		// Restore permissions so TempDir cleanup succeeds
		_ = os.Chmod(licensePath, 0644)
	})

	t.Chdir(tmpDir)

	withTestConfig(t, func(c *config.Config) {
		c.Project.CopyrightHolder = "Test Corp."
		c.Project.CopyrightYear = 2020
	})

	buf := new(bytes.Buffer)
	headersCmd.SetOut(buf)

	// Should not panic; the error is silently handled
	updateLicenseFile(headersCmd, licensePath, true, false)

	// File should remain unchanged since write was not possible
	content, err := os.ReadFile(licensePath)
	require.NoError(t, err)
	assert.Equal(t, licenseContent, string(content))
}

func Test_updateLicenseFile_NonExistentPath(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create a dummy file and commit so git operations work
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dummy.txt"), []byte("x"), 0644))
	gitAddCommit(t, tmpDir, "init")

	t.Chdir(tmpDir)

	withTestConfig(t, func(c *config.Config) {
		c.Project.CopyrightHolder = "Test Corp."
		c.Project.CopyrightYear = 2020
	})

	buf := new(bytes.Buffer)
	headersCmd.SetOut(buf)

	// Pass a path that does not exist — should not panic
	updateLicenseFile(headersCmd, filepath.Join(tmpDir, "NONEXISTENT"), true, false)
	assert.Empty(t, buf.String())
}

func Test_updateLicenseFile_NoUpdateWhenNotDirty(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	currentYear := time.Now().Year()
	// LICENSE already has the current year — no update needed
	licenseContent := fmt.Sprintf("Copyright Test Corp. 2020, %d\n\nLicense text\n", currentYear)
	licensePath := filepath.Join(tmpDir, "LICENSE")
	require.NoError(t, os.WriteFile(licensePath, []byte(licenseContent), 0644))
	gitAddCommit(t, tmpDir, "init")

	t.Chdir(tmpDir)

	withTestConfig(t, func(c *config.Config) {
		c.Project.CopyrightHolder = "Test Corp."
		c.Project.CopyrightYear = 2020
	})

	buf := new(bytes.Buffer)
	headersCmd.SetOut(buf)

	// anyFileUpdated=false means we don't force current-year update
	updateLicenseFile(headersCmd, licensePath, false, false)

	// File should remain unchanged
	content, err := os.ReadFile(licensePath)
	require.NoError(t, err)
	assert.Equal(t, licenseContent, string(content))
}
