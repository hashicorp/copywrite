// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/hcl"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/spf13/pflag"
)

var (
	// Use a period for delimiting sections of the config, e.g.:
	// project.copyright_year or dispatch.branch
	delim = "."
)

// Project represents data needed for copyright and licensing statements inside
// a specific project/repo
type Project struct {
	CopyrightYear   int      `koanf:"copyright_year"`
	CopyrightYear1  int      `koanf:"copyright_year1"`
	CopyrightYear2  int      `koanf:"copyright_year2"`
	CopyrightHolder string   `koanf:"copyright_holder"`
	HeaderIgnore    []string `koanf:"header_ignore"`
	License         string   `koanf:"license"`

	// Upstream is optional and only used if a given repo pulls from another
	Upstream string `koanf:"upstream"`
}

// Dispatch represents data needed by the `copywrite dispatch` command, and is
// used to control ignored repos, concurrency, and other information
type Dispatch struct {
	// A unique identifier for the current batch of workflow runs
	BatchID string `koanf:"batch_id"`

	// The GitHub Branch to base workflow runs off of
	Branch string `koanf:"branch"`

	// The GitHub Organization who's repositories you want to audit
	GitHubOrgToAudit string `koanf:"github_org_to_audit"`

	// A list of repos that should be exempted from scans.
	// Repo names must be fully-qualified (i.e., include the org name), like so:
	// "hashicorp/copywrite"
	IgnoredRepos []string `koanf:"ignored_repos"`

	// Sleep time in seconds between polling operations
	Sleep int `koanf:"sleep"`

	// maxAttempts is the maximum number of times a worker will check if a
	// workflow is finished (sleeping between each attempt) before timing out
	MaxAttempts int `koanf:"max_attempts"`

	// The number of concurrent workers in the worker pool
	Workers int `koanf:"workers"`

	// The workflow file name to be used when triggering GitHub Actions jobs
	WorkflowFileName string `koanf:"workflow_file_name"`
}

// Config is a struct representing the data from a well-defined config file
type Config struct {
	SchemaVersion int      `koanf:"schema_version"`
	Project       Project  `koanf:"project"`
	Dispatch      Dispatch `koanf:"dispatch"`

	// Global koanf instance
	globalKoanf *koanf.Koanf

	// Stores the absolute path of a .copywrite.hcl config object, if it exists
	absCfgPath string
}

// New returns a Config object initialized with default values
func New() (*Config, error) {
	k := koanf.New(delim)
	c := &Config{
		globalKoanf: k,
	}

	// Preload default config values
	defaults := map[string]interface{}{
		"schema_version":           1,
		"project.copyright_holder": "IBM Corp.",
	}
	err := c.LoadConfMap(defaults)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// MustNew returns a Config object initialized with default values
// and panics if that is not possible
func MustNew() *Config {
	c, err := New()
	if err != nil {
		panic(err)
	}

	return c
}

// LoadConfMap updates the running config with a key-value map, where
// keys are delimited configuration key references.
//
// Example mapping:
//
//	map[string]interface{}{
//		"schema_version":         2,
//		"project.copyright_year": 2022,
//		"project.license":        "MPL-2.0",
//		"dispatch.ignored_repos": []string{"foo", "bar"},
//	}
func (c *Config) LoadConfMap(mp map[string]interface{}) error {
	err := c.globalKoanf.Load(confmap.Provider(mp, delim), nil)
	if err != nil {
		return err
	}

	// Update the global config object with the new new
	err = c.globalKoanf.Unmarshal("", &c)
	if err != nil {
		return err
	}

	return nil
}

// LoadCommandFlags updates the running config with any command-line flags
// based on a mapping of flag names to config keys
//
// Example mapping (flag name: config key):
//
//	mapping := map[string]string{
//		`license`: `project.license`,
//		`year`:    `project.copyright_year`,
//	}
//
// Merge Behavior:
// If a configuration value already exists (e.g., from previously reading a
// .copywrite.hcl config file), those values will only be overwritten by default
// flag values if clobberWithDefaults is true. If it is false, only values from
// flags the user explicitly sets will be transferred to the configuration.
//
// Default flag options will be always be loaded if no value was previously set
// in the running configuration.
func (c *Config) LoadCommandFlags(flagSet *pflag.FlagSet, mapping map[string]string, clobberWithDefaults bool) error {
	// a new/blank koanf.New(delim) is used if we want to load all default flag
	// values, even if that would mean clobbering an already set config value.
	// If we wish to flip that behavior, we pass in the config's Koanf object
	// instead so that no clobbering exists.
	ko := c.globalKoanf
	if clobberWithDefaults {
		ko = koanf.New(delim)
	}

	// Parse out flag values
	p := posflag.ProviderWithFlag(flagSet, delim, ko, func(f *pflag.Flag) (string, interface{}) {
		// Transform the key name based on the provided mapping
		key := mapping[f.Name]

		// Retrieve the flag value
		val := posflag.FlagVal(flagSet, f)

		return key, val
	})

	// Load up the new values into the global Koanf instance
	err := c.globalKoanf.Load(p, nil)
	if err != nil {
		return err
	}

	// Update the global config object with the new new
	err = c.globalKoanf.Unmarshal("", &c)
	if err != nil {
		return err
	}

	return nil
}

// LoadConfigFile takes a path to an HCL config file and
// merges it with the running config
//
// Example HCL config:
//
//	schema_version = 1
//	project {
//		copyright_year = 2022
//		license        = "MPL-2.0"
//	}
func (c *Config) LoadConfigFile(cfgPath string) error {
	abs, err := filepath.Abs(cfgPath)
	if err != nil {
		return fmt.Errorf("unable to determine config path: %w", err)
	}
	c.absCfgPath = abs

	// If a config file exists, let's load it
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("config file doesn't exist: %w", err)
	}

	// Load HCL config.
	err = c.globalKoanf.Load(file.Provider(abs), hcl.Parser(true))
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	// Attempt to suss out a Config struct
	err = c.globalKoanf.Unmarshal("", &c)
	if err != nil {
		return fmt.Errorf("unable to unmarshal config: %w", err)
	}

	return nil
}

// Sprint returns a textual version of the current running config.
// The string is newline-delimited and contains alphabetical key -> value pairs
func (c *Config) Sprint() string {
	return c.globalKoanf.Sprint()
}

// GetConfigPath returns the absolute path of the last loaded HCL config.
// If LoadConfigFile() has not been called, it will return an empty string.
func (c *Config) GetConfigPath() string {
	return c.absCfgPath
}
