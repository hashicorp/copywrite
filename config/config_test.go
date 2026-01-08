// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/knadh/koanf"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func Test_New(t *testing.T) {
	t.Run("Validate default settings", func(t *testing.T) {
		actualOutput, err := New()
		assert.Nil(t, err, "Error must be nil")

		// Validate that a non-nil object was returned
		assert.NotNil(t, actualOutput, "Config object must not be nil")

		// Validate Koanf object exists
		assert.NotNil(t, actualOutput.globalKoanf, "Koanf object should exist")

		// Validate the default value(s)
		assert.Equal(t, 1, actualOutput.SchemaVersion, "Schema Version defaults to 1")
		assert.Equal(t, "IBM Corp.", actualOutput.Project.CopyrightHolder, "Copyright Holder defaults to 'IBM Corp.'")
		assert.Equal(t, "project.copyright_holder -> IBM Corp.\nschema_version -> 1\n", actualOutput.Sprint(), "Koanf object gets updated appropriately with defaults")
	})
}

func Test_LoadConfMap(t *testing.T) {
	mp := map[string]interface{}{
		"schema_version":         12,
		"project.copyright_year": 9001,
		"project.license":        "MPL-2.0",
		"dispatch.ignored_repos": []string{"foo", "bar"},
	}

	// update the running config with any command-line flags
	actualOutput := MustNew()
	err := actualOutput.LoadConfMap(mp)
	assert.Nil(t, err, "Loading should not error")

	expectedOutput := &Config{
		globalKoanf:   koanf.New(delim),
		SchemaVersion: 12,
		Project: Project{
			CopyrightHolder: "IBM Corp.",
			CopyrightYear:   9001,
			License:         "MPL-2.0",
		},
		Dispatch: Dispatch{
			IgnoredRepos: []string{
				"foo",
				"bar",
			},
		},
	}

	// Test schema version
	assert.Equal(t, expectedOutput.SchemaVersion, actualOutput.SchemaVersion, "Schema override should work")

	// Test project struct
	assert.Equal(t, expectedOutput.Project, actualOutput.Project, "Partial Project override should work")

	// Test dispatch struct
	assert.Equal(t, expectedOutput.Dispatch, actualOutput.Dispatch, "Partial Dispatch override should work")
}

func Test_LoadCommandFlags(t *testing.T) {
	// Map command flags to config keys
	mapping := map[string]string{
		`schemaVersion`:   `schema_version`,
		`spdx`:            `project.license`,
		`year`:            `project.copyright_year`,
		`copyrightHolder`: `project.copyright_holder`,
		`ignoredRepos`:    `dispatch.ignored_repos`,
	}

	tests := []struct {
		description         string
		args                []string
		clobberWithDefaults bool
		expectedOutput      *Config
	}{
		{
			description:         "unset schema flag should not override preset value",
			args:                []string{},
			clobberWithDefaults: false,
			expectedOutput: &Config{
				SchemaVersion: 1,
				Project: Project{
					CopyrightHolder: "IBM Corp.",
					CopyrightYear:   9001,
					License:         "MPL-2.0",
				},
				Dispatch: Dispatch{
					IgnoredRepos: []string{"foo", "bar"},
				},
			},
		},
		{
			description:         "unset schema flag should override preset value when overrideWithDefaults is set",
			args:                []string{},
			clobberWithDefaults: true,
			expectedOutput: &Config{
				SchemaVersion: 12,
				Project: Project{
					CopyrightHolder: "IBM Corp.",
					CopyrightYear:   9001,
					License:         "MPL-2.0",
				},
				Dispatch: Dispatch{
					IgnoredRepos: []string{"foo", "bar"},
				},
			},
		},
		{
			description:         "explicitly set flag should override preset value",
			args:                []string{"--schemaVersion=33"},
			clobberWithDefaults: false,
			expectedOutput: &Config{
				SchemaVersion: 33,
				Project: Project{
					CopyrightHolder: "IBM Corp.",
					CopyrightYear:   9001,
					License:         "MPL-2.0",
				},
				Dispatch: Dispatch{
					IgnoredRepos: []string{"foo", "bar"},
				},
			},
		},
		{
			description:         "explicitly set flag should override preset value even if overrideWithDefaults is set",
			args:                []string{"--schemaVersion=33"},
			clobberWithDefaults: true,
			expectedOutput: &Config{
				SchemaVersion: 33,
				Project: Project{
					CopyrightHolder: "IBM Corp.",
					CopyrightYear:   9001,
					License:         "MPL-2.0",
				},
				Dispatch: Dispatch{
					IgnoredRepos: []string{"foo", "bar"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			flags := &pflag.FlagSet{}
			flags.Int("schemaVersion", 12, "Config Schema Version")
			flags.String("spdx", "MPL-2.0", "SPDX License Identifier")
			flags.Int("year", 9001, "Year of copyright")
			flags.String("copyrightHolder", "IBM Corp.", "Copyright Holder")
			flags.StringArray("ignoredRepos", []string{"foo", "bar"}, "repos to ignore")
			err := flags.Parse(tt.args)
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")

			// update the running config with any command-line flags
			actualOutput := MustNew()
			err = actualOutput.LoadCommandFlags(flags, mapping, tt.clobberWithDefaults)
			assert.Nil(t, err, "Loading should not error")

			// Test schema version
			assert.Equal(t, tt.expectedOutput.SchemaVersion, actualOutput.SchemaVersion, "Schema override should work")

			// Test project struct
			assert.Equal(t, tt.expectedOutput.Project, actualOutput.Project, "Partial Project override should work")

			// Test dispatch struct
			assert.Equal(t, tt.expectedOutput.Dispatch, actualOutput.Dispatch, "Partial Dispatch override should work")
		})
	}
}

func Test_LoadConfigFile(t *testing.T) {
	tests := []struct {
		description    string
		inputCfgPath   string
		expectedOutput *Config
	}{
		{
			description:    "Empty file results in empty config",
			inputCfgPath:   "testdata/empty_config.hcl",
			expectedOutput: &Config{},
		},
		{
			description:  "File with schema_version updates config accordingly",
			inputCfgPath: "testdata/config_with_schema_version.hcl",
			expectedOutput: &Config{
				SchemaVersion: 42,
			},
		},
		// Test Project-Related Configuration
		{
			description:  "File with project.copyright_holder populates accordingly",
			inputCfgPath: "testdata/project/copyright_holder_only.hcl",
			expectedOutput: &Config{
				Project: Project{
					CopyrightHolder: "Dummy Corporation",
				},
			},
		},
		{
			description:  "File with project.copyright_year populates accordingly",
			inputCfgPath: "testdata/project/copyright_year_only.hcl",
			expectedOutput: &Config{
				Project: Project{
					CopyrightYear: 9001,
				},
			},
		},
		{
			description:  "File with project.license populates accordingly",
			inputCfgPath: "testdata/project/license_only.hcl",
			expectedOutput: &Config{
				Project: Project{
					License: "NOT_A_VALID_SPDX",
				},
			},
		},
		{
			description:  "File with partial project populates accordingly",
			inputCfgPath: "testdata/project/partial_project.hcl",
			expectedOutput: &Config{
				Project: Project{
					CopyrightYear: 9001,
					License:       "NOT_A_VALID_SPDX",
				},
			},
		},
		{
			description:  "File with full project populates accordingly",
			inputCfgPath: "testdata/project/full_project.hcl",
			expectedOutput: &Config{
				SchemaVersion: 12,
				Project: Project{
					CopyrightYear:   9001,
					CopyrightHolder: "Dummy Corporation",
					License:         "NOT_A_VALID_SPDX",
					HeaderIgnore: []string{
						"asdf.go",
						"*.css",
						"**/vendor/**.go",
					},
					Upstream: "hashicorp/super-secret-private-repo",
				},
			},
		},
		// Test Dispatch-Related Configuration
		{
			description:  "File with full dispatch populates accordingly",
			inputCfgPath: "testdata/dispatch/full_dispatch.hcl",
			expectedOutput: &Config{
				SchemaVersion: 78,
				Dispatch: Dispatch{
					BatchID:          "aZ0-9",
					Branch:           "main",
					GitHubOrgToAudit: "hashicorp-forge",
					IgnoredRepos: []string{
						"org/repo1",
						"org/repo2",
					},
					Sleep:            42,
					MaxAttempts:      3,
					Workers:          12,
					WorkflowFileName: "repair-repo-headers.yml",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			actualOutput := &Config{globalKoanf: koanf.New(delim)}
			err := actualOutput.LoadConfigFile(tt.inputCfgPath)
			assert.Nil(t, err, "Loading should not error")

			// Test schema version
			assert.Equal(t, tt.expectedOutput.SchemaVersion, actualOutput.SchemaVersion, tt.description)

			// Test project struct
			assert.Equal(t, tt.expectedOutput.Project, actualOutput.Project, tt.description)

			// Test dispatch struct
			assert.Equal(t, tt.expectedOutput.Dispatch, actualOutput.Dispatch, tt.description)
		})
	}
}

func Test_Sprint(t *testing.T) {
	tests := []struct {
		description    string
		inputCfgPath   string
		expectedOutput string
	}{
		{
			description:    "Empty file results in empty config",
			inputCfgPath:   "testdata/empty_config.hcl",
			expectedOutput: "",
		},
		{
			description:    "File with schema_version updates config accordingly",
			inputCfgPath:   "testdata/config_with_schema_version.hcl",
			expectedOutput: "schema_version -> 42\n",
		},
		// Test Project-Related Configuration
		{
			description:  "File with full project populates accordingly",
			inputCfgPath: "testdata/project/full_project.hcl",
			expectedOutput: strings.Join([]string{
				"project.copyright_holder -> Dummy Corporation",
				"project.copyright_year -> 9001",
				"project.header_ignore -> [asdf.go *.css **/vendor/**.go]",
				"project.license -> NOT_A_VALID_SPDX",
				"project.upstream -> hashicorp/super-secret-private-repo",
				"schema_version -> 12\n",
			}, "\n"),
		},
		// Test Dispatch-Related Configuration
		{
			description:  "File with full dispatch populates accordingly",
			inputCfgPath: "testdata/dispatch/full_dispatch.hcl",
			expectedOutput: strings.Join([]string{
				"dispatch.batch_id -> aZ0-9",
				"dispatch.branch -> main",
				"dispatch.github_org_to_audit -> hashicorp-forge",
				"dispatch.ignored_repos -> [org/repo1 org/repo2]",
				"dispatch.max_attempts -> 3",
				"dispatch.sleep -> 42",
				"dispatch.workers -> 12",
				"dispatch.workflow_file_name -> repair-repo-headers.yml",
				"schema_version -> 78\n",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			c := &Config{globalKoanf: koanf.New(delim)}
			// TODO: somehow remove this dependency to more purely test just Sprint()
			err := c.LoadConfigFile(tt.inputCfgPath)
			assert.Nil(t, err, "Loading should not error")

			actualOutput := c.Sprint()

			assert.Equal(t, tt.expectedOutput, actualOutput, tt.description)
		})
	}
}

func Test_GetConfigPath(t *testing.T) {
	// test new config without calling Load
	actualOutput := MustNew()
	assert.Equal(t, "", actualOutput.GetConfigPath(), "Unloaded config should return empty string")

	// test new config _after_ calling Load
	cfgPath := "testdata/empty_config.hcl"
	err := actualOutput.LoadConfigFile(cfgPath)
	assert.Nil(t, err, "Loading should not error")

	abs, _ := filepath.Abs(cfgPath)
	assert.Equal(t, abs, actualOutput.GetConfigPath(), "Loaded config should return abs file path")
}

func Test_FormatCopyrightYears(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		description    string
		copyrightYear  int
		expectedOutput string
	}{
		{
			description:    "Copyright year equals current year should return single year",
			copyrightYear:  currentYear,
			expectedOutput: strconv.Itoa(currentYear),
		},
		{
			description:    "Copyright year before current year should return year range",
			copyrightYear:  2023,
			expectedOutput: fmt.Sprintf("2023, %d", currentYear),
		},
		{
			description:    "Old copyright year should return year range",
			copyrightYear:  2018,
			expectedOutput: fmt.Sprintf("2018, %d", currentYear),
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			c := MustNew()
			c.Project.CopyrightYear = tt.copyrightYear

			actualOutput := c.FormatCopyrightYears()

			assert.Equal(t, tt.expectedOutput, actualOutput, tt.description)
		})
	}
}

func Test_FormatCopyrightYears_AutoDetect(t *testing.T) {
	currentYear := time.Now().Year()

	t.Run("Auto-detect from git when copyright_year not set", func(t *testing.T) {
		c := MustNew()
		c.Project.CopyrightYear = 0

		// Set config path to this repo's directory for git detection
		c.absCfgPath = filepath.Join(getCurrentDir(t), ".copywrite.hcl")

		actualOutput := c.FormatCopyrightYears()

		// Should auto-detect and return a year range (this repo was created before 2025)
		// The format should be "YYYY, currentYear" where YYYY < currentYear
		assert.Contains(t, actualOutput, ",", "Should contain year range when auto-detected from git")
		assert.Contains(t, actualOutput, strconv.Itoa(currentYear), "Should contain current year")

		// Parse and validate the detected year
		parts := strings.Split(actualOutput, ", ")
		if len(parts) == 2 {
			detectedYear, err := strconv.Atoi(parts[0])
			assert.Nil(t, err, "First part should be a valid year")
			assert.True(t, detectedYear >= 2020 && detectedYear <= currentYear,
				"Detected year should be reasonable (between 2020 and current year)")
		}
	})

	t.Run("Fallback to current year when git not available", func(t *testing.T) {
		c := MustNew()
		c.Project.CopyrightYear = 0

		// Set config path to non-existent directory (git will fail)
		c.absCfgPath = "/nonexistent/path/.copywrite.hcl"

		actualOutput := c.FormatCopyrightYears()

		// Should fallback to current year only
		assert.Equal(t, strconv.Itoa(currentYear), actualOutput,
			"Should fallback to current year when git detection fails")
	})
}

// Helper function to get current directory
func getCurrentDir(t *testing.T) string {
	dir, err := os.Getwd()
	assert.Nil(t, err, "Should be able to get current directory")
	return dir
}
