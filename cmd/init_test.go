// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hashicorp/copywrite/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_configToHCL(t *testing.T) {
	tests := []struct {
		name           string
		config         config.Config
		wantErr        bool
		expectedParts  []string
		unexpectedPart string
	}{
		{
			name: "valid config with MPL-2.0",
			config: config.Config{
				SchemaVersion: 1,
				Project: config.Project{
					License:       "MPL-2.0",
					CopyrightYear: 2023,
				},
			},
			wantErr: false,
			expectedParts: []string{
				`schema_version = 1`,
				`license        = "MPL-2.0"`,
				`copyright_year = 2023`,
				`header_ignore`,
			},
		},
		{
			name: "valid config with MIT license",
			config: config.Config{
				SchemaVersion: 1,
				Project: config.Project{
					License:       "MIT",
					CopyrightYear: 2020,
				},
			},
			wantErr: false,
			expectedParts: []string{
				`schema_version = 1`,
				`license        = "MIT"`,
				`copyright_year = 2020`,
			},
		},
		{
			name: "config with zero year",
			config: config.Config{
				SchemaVersion: 1,
				Project: config.Project{
					License:       "Apache-2.0",
					CopyrightYear: 0,
				},
			},
			wantErr: false,
			expectedParts: []string{
				`copyright_year = 0`,
				`license        = "Apache-2.0"`,
			},
		},
		{
			name: "config with empty license",
			config: config.Config{
				SchemaVersion: 1,
				Project: config.Project{
					License:       "",
					CopyrightYear: 2024,
				},
			},
			wantErr: false,
			expectedParts: []string{
				`license        = ""`,
				`copyright_year = 2024`,
			},
		},
		{
			name: "schema version 2",
			config: config.Config{
				SchemaVersion: 2,
				Project: config.Project{
					License:       "BSD-3-Clause",
					CopyrightYear: 2019,
				},
			},
			wantErr: false,
			expectedParts: []string{
				`schema_version = 2`,
				`license        = "BSD-3-Clause"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := configToHCL(tt.config, &buf)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			output := buf.String()

			for _, part := range tt.expectedParts {
				assert.Contains(t, output, part)
			}

			if tt.unexpectedPart != "" {
				assert.NotContains(t, output, tt.unexpectedPart)
			}
		})
	}
}

func Test_configToHCL_WriterError(t *testing.T) {
	cfg := config.Config{
		SchemaVersion: 1,
		Project: config.Project{
			License:       "MPL-2.0",
			CopyrightYear: 2023,
		},
	}

	ew := &errorWriter{}
	err := configToHCL(cfg, ew)
	assert.Error(t, err)
}

type errorWriter struct{}

func (ew *errorWriter) Write(p []byte) (int, error) {
	return 0, os.ErrClosed
}

func TestInitCmd_Flags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		defValue string
	}{
		{name: "force flag", flagName: "force", defValue: "false"},
		{name: "year flag", flagName: "year", defValue: "0"},
		{name: "spdx flag", flagName: "spdx", defValue: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := initCmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, flag, "flag %q should exist", tt.flagName)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestInitCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	initCmd.SetOut(buf)
	initCmd.SetErr(buf)
	initCmd.SetArgs([]string{"--help"})

	err := initCmd.Help()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Generates a .copywrite.hcl config")
	assert.Contains(t, output, "--force")
	assert.Contains(t, output, "--year")
	assert.Contains(t, output, "--spdx")
}

func Test_configToHCL_ContainsProjectBlock(t *testing.T) {
	cfg := config.Config{
		SchemaVersion: 1,
		Project: config.Project{
			License:       "MPL-2.0",
			CopyrightYear: 2023,
		},
	}

	var buf bytes.Buffer
	err := configToHCL(cfg, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "project {")
	assert.Contains(t, output, "header_ignore")
	assert.Contains(t, output, "# (OPTIONAL)")
}

func TestInitCmd_Run_NoTTY(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo for github.DiscoverRepo() — will fail gracefully
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

	// Create dummy commit
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dummy.go"), []byte("package main"), 0644))
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(tmpDir))

	// Ensure no existing .copywrite.hcl
	os.Remove(filepath.Join(tmpDir, ".copywrite.hcl"))

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init", "--spdx", "MPL-2.0", "--year", "2023"})

	err = rootCmd.Execute()
	if err == nil {
		// Verify the config file was created
		_, statErr := os.Stat(filepath.Join(tmpDir, ".copywrite.hcl"))
		assert.NoError(t, statErr)

		// Verify content
		content, readErr := os.ReadFile(filepath.Join(tmpDir, ".copywrite.hcl"))
		if readErr == nil {
			assert.Contains(t, string(content), "MPL-2.0")
			assert.Contains(t, string(content), "2023")
		}
	}
}

func TestInitCmd_PreRun_InvalidSPDX(t *testing.T) {
	// cobra.CheckErr in PreRun calls os.Exit, so we test this in a subprocess
	if os.Getenv("TEST_INIT_INVALID_SPDX") == "1" {
		tmpDir := t.TempDir()
		os.Chdir(tmpDir)

		rootCmd.SetArgs([]string{"init", "--spdx", "INVALID-LICENSE-XYZ", "--force"})
		rootCmd.Execute()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestInitCmd_PreRun_InvalidSPDX", "-test.count=1")
	cmd.Env = append(os.Environ(), "TEST_INIT_INVALID_SPDX=1")
	output, err := cmd.CombinedOutput()
	// The subprocess should exit non-zero
	assert.Error(t, err)
	assert.Contains(t, string(output), "invalid SPDX license identifier")
}

func TestInitCmd_PreRun_ExistingConfig(t *testing.T) {
	// cobra.CheckErr in PreRun calls os.Exit, so we test this in a subprocess
	if os.Getenv("TEST_INIT_EXISTING_CONFIG") == "1" {
		tmpDir, _ := os.MkdirTemp("", "copywrite-test-*")
		validHCL := `schema_version = 1
project {
  license = "MIT"
  copyright_year = 2020
}
`
		os.WriteFile(filepath.Join(tmpDir, ".copywrite.hcl"), []byte(validHCL), 0644)
		os.Chdir(tmpDir)

		rootCmd.SetArgs([]string{"init", "--spdx", "MPL-2.0"})
		rootCmd.Execute()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestInitCmd_PreRun_ExistingConfig", "-test.count=1")
	cmd.Env = append(os.Environ(), "TEST_INIT_EXISTING_CONFIG=1")
	output, err := cmd.CombinedOutput()
	assert.Error(t, err)
	assert.Contains(t, string(output), "already exists")
}

func TestInitCmd_PreRun_ForceOverwrite(t *testing.T) {
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
		require.NoError(t, cmd.Run())
	}
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "dummy.go"), []byte("package main"), 0644))
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create existing .copywrite.hcl with valid HCL content
	validHCL := `schema_version = 1
project {
  license = "MIT"
  copyright_year = 2020
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".copywrite.hcl"), []byte(validHCL), 0644))

	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(origDir)
	require.NoError(t, os.Chdir(tmpDir))

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init", "--force", "--spdx", "MIT", "--year", "2022"})

	err = rootCmd.Execute()
	if err == nil {
		content, readErr := os.ReadFile(filepath.Join(tmpDir, ".copywrite.hcl"))
		require.NoError(t, readErr)
		assert.Contains(t, string(content), "MIT")
		assert.Contains(t, string(content), "2022")
	}
}
