// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"path/filepath"
	"strings"
	"testing"

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
		"schema_version":          12,
		"project.copyright_year":  9001,
		"project.copyright_year1": 6001,
		"project.copyright_year2": 7001,
		"project.license":         "MPL-2.0",
		"dispatch.ignored_repos":  []string{"foo", "bar"},
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
			CopyrightYear1:  6001,
			CopyrightYear2:  7001,
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
		`schemaVersion`: `schema_version`,
		`spdx`:          `project.license`,
		`year`:          `project.copyright_year`,
		`year1`:         `project.copyright_year1`,
		`year2`:         `project.copyright_year2`,
		`ignoredRepos`:  `dispatch.ignored_repos`,
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
					CopyrightYear1:  6001,
					CopyrightYear2:  7001,
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
					CopyrightYear1:  6001,
					CopyrightYear2:  7001,
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
					CopyrightYear1:  6001,
					CopyrightYear2:  7001,
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
					CopyrightYear1:  6001,
					CopyrightYear2:  7001,
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
			flags.Int("year1", 6001, "First year of copyright")
			flags.Int("year2", 7001, "Second year of copyright")
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
					CopyrightYear:  9001,
					CopyrightYear1: 6001,
					CopyrightYear2: 7001,
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
					CopyrightYear:  9001,
					CopyrightYear1: 6001,
					CopyrightYear2: 7001,
					License:        "NOT_A_VALID_SPDX",
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
					CopyrightYear1:  6001,
					CopyrightYear2:  7001,
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
				"project.copyright_year1 -> 6001",
				"project.copyright_year2 -> 7001",
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
