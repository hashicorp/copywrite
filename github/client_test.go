// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package github

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mitchellh/go-homedir"
	"golang.org/x/oauth2"
)

// clearAppEnv blanks the three GitHub App env vars so tests start clean.
func clearAppEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ID", "")
	t.Setenv("INSTALLATION_ID", "")
	t.Setenv("APP_PEM", "")
}

// disableHomedirCache disables go-homedir's cache for the duration of a test.
func disableHomedirCache(t *testing.T) {
	t.Helper()
	homedir.DisableCache = true
	t.Cleanup(func() { homedir.DisableCache = false })
}

// unsetGitHubToken removes GITHUB_TOKEN for the duration of a test and
// restores it (or leaves it unset) when the test ends.
func unsetGitHubToken(t *testing.T) {
	t.Helper()
	prev, had := os.LookupEnv("GITHUB_TOKEN")
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	t.Cleanup(func() {
		if had {
			if err := os.Setenv("GITHUB_TOKEN", prev); err != nil {
				t.Errorf("cleanup: failed to restore GITHUB_TOKEN: %v", err)
			}
		} else {
			if err := os.Unsetenv("GITHUB_TOKEN"); err != nil {
				t.Errorf("cleanup: failed to unset GITHUB_TOKEN: %v", err)
			}
		}
	})
}

func TestGHClient_Raw(t *testing.T) {
	client := NewGHClient()
	require.NotNil(t, client)
	assert.NotNil(t, client.Raw())
}

func TestGetGHAppConfig(t *testing.T) {
	tests := []struct {
		name       string
		envVars    map[string]string
		wantExists bool
		wantAppID  int64
		wantInstID int64
		wantPEM    string
	}{
		{
			name:       "no config present",
			envVars:    map[string]string{},
			wantExists: false,
		},
		{
			name: "only APP_ID set",
			envVars: map[string]string{
				"APP_ID": "12345",
			},
			wantExists: false,
		},
		{
			name: "only INSTALLATION_ID set",
			envVars: map[string]string{
				"INSTALLATION_ID": "67890",
			},
			wantExists: false,
		},
		{
			name: "only APP_PEM set",
			envVars: map[string]string{
				"APP_PEM": "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
			},
			wantExists: false,
		},
		{
			name: "APP_ID and INSTALLATION_ID set but no PEM",
			envVars: map[string]string{
				"APP_ID":          "12345",
				"INSTALLATION_ID": "67890",
			},
			wantExists: false,
		},
		{
			name: "APP_ID and APP_PEM set but no INSTALLATION_ID",
			envVars: map[string]string{
				"APP_ID":  "12345",
				"APP_PEM": "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
			},
			wantExists: false,
		},
		{
			name: "all three set - config exists",
			envVars: map[string]string{
				"APP_ID":          "12345",
				"INSTALLATION_ID": "67890",
				"APP_PEM":         "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
			},
			wantExists: true,
			wantAppID:  12345,
			wantInstID: 67890,
			wantPEM:    "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
		},
		{
			name: "APP_PEM with escaped newlines",
			envVars: map[string]string{
				"APP_ID":          "11111",
				"INSTALLATION_ID": "22222",
				"APP_PEM":         "-----BEGIN RSA PRIVATE KEY-----\\nMIIE\\ntest\\n-----END RSA PRIVATE KEY-----",
			},
			wantExists: true,
			wantAppID:  11111,
			wantInstID: 22222,
			wantPEM:    "-----BEGIN RSA PRIVATE KEY-----\nMIIE\ntest\n-----END RSA PRIVATE KEY-----",
		},
		{
			name: "APP_ID is zero (invalid)",
			envVars: map[string]string{
				"APP_ID":          "0",
				"INSTALLATION_ID": "67890",
				"APP_PEM":         "some-pem-data",
			},
			wantExists: false,
		},
		{
			name: "INSTALLATION_ID is zero (invalid)",
			envVars: map[string]string{
				"APP_ID":          "12345",
				"INSTALLATION_ID": "0",
				"APP_PEM":         "some-pem-data",
			},
			wantExists: false,
		},
		{
			name: "APP_PEM is empty string",
			envVars: map[string]string{
				"APP_ID":          "12345",
				"INSTALLATION_ID": "67890",
				"APP_PEM":         "",
			},
			wantExists: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clean env before each test
			t.Setenv("APP_ID", "")
			t.Setenv("INSTALLATION_ID", "")
			t.Setenv("APP_PEM", "")

			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			cc, exists := getGHAppConfig()
			assert.Equal(t, tc.wantExists, exists)

			if tc.wantExists {
				assert.Equal(t, tc.wantAppID, cc.appID)
				assert.Equal(t, tc.wantInstID, cc.instID)
				assert.Equal(t, tc.wantPEM, cc.appPEM)
			}
		})
	}
}

// NOTE: This test mutates homedir.DisableCache (global state)
// and must NOT use t.Parallel().
func TestGetGitHubCLIConfig(t *testing.T) {
	disableHomedirCache(t)
	tests := []struct {
		name        string
		configData  string
		wantToken   string
		wantExists  bool
		setupHome   bool
		noConfigDir bool
	}{
		{
			name: "valid config with oauth_token",
			configData: `github.com:
    user: octocat
    oauth_token: gho_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    git_protocol: https
`,
			wantToken:  "gho_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			wantExists: true,
			setupHome:  true,
		},
		{
			name: "config without oauth_token",
			configData: `github.com:
    user: octocat
    git_protocol: https
`,
			wantToken:  "",
			wantExists: false,
			setupHome:  true,
		},
		{
			name:       "empty config file",
			configData: "",
			wantToken:  "",
			wantExists: false,
			setupHome:  true,
		},
		{
			name: "config with different host only",
			configData: `github.enterprise.com:
    user: octocat
    oauth_token: gho_enterprise_token
    git_protocol: https
`,
			wantToken:  "",
			wantExists: false,
			setupHome:  true,
		},
		{
			name:        "no config directory exists",
			configData:  "",
			wantToken:   "",
			wantExists:  false,
			setupHome:   true,
			noConfigDir: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupHome {
				tmpHome := t.TempDir()
				t.Setenv("HOME", tmpHome)

				if !tc.noConfigDir {
					configDir := filepath.Join(tmpHome, ".config", "gh")
					err := os.MkdirAll(configDir, 0755)
					require.NoError(t, err)

					configPath := filepath.Join(configDir, "hosts.yml")
					err = os.WriteFile(configPath, []byte(tc.configData), 0644)
					require.NoError(t, err)
				}
			}

			token, exists := getGitHubCLIConfig()
			assert.Equal(t, tc.wantExists, exists)
			assert.Equal(t, tc.wantToken, token)
		})
	}
}

func TestNewGHClient_WithGitHubToken(t *testing.T) {
	clearAppEnv(t)

	t.Setenv("GITHUB_TOKEN", "ghp_test_token_12345")

	client := NewGHClient()
	require.NotNil(t, client)
	assert.NotNil(t, client.gh)
	assert.NotNil(t, client.Raw())
	_, ok := client.Raw().Client().Transport.(*oauth2.Transport)
	assert.True(t, ok, "expected oauth2 transport for GITHUB_TOKEN client")
}

func TestNewGHClient_UnauthenticatedFallback(t *testing.T) {
	disableHomedirCache(t)
	clearAppEnv(t)
	// Ensure no GITHUB_TOKEN — must unset, not empty, because NewGHClient uses os.LookupEnv
	unsetGitHubToken(t)

	// Set HOME to a temp dir with no gh config
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	client := NewGHClient()
	require.NotNil(t, client)
	assert.NotNil(t, client.gh)
	assert.NotNil(t, client.Raw())
	assert.Nil(t, client.Raw().Client().Transport, "expected unauthenticated client: transport should be nil")
}

func TestNewGHClient_WithGHCLIConfig(t *testing.T) {
	disableHomedirCache(t)
	clearAppEnv(t)
	// Ensure no GITHUB_TOKEN — must unset, not empty, because NewGHClient uses os.LookupEnv
	unsetGitHubToken(t)

	// Set up gh CLI config
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	configDir := filepath.Join(tmpHome, ".config", "gh")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configData := `github.com:
    user: testuser
    oauth_token: gho_testtoken123456
    git_protocol: https
`
	err = os.WriteFile(filepath.Join(configDir, "hosts.yml"), []byte(configData), 0644)
	require.NoError(t, err)

	client := NewGHClient()
	require.NotNil(t, client)
	assert.NotNil(t, client.gh)
	assert.NotNil(t, client.Raw())
	_, ok := client.Raw().Client().Transport.(*oauth2.Transport)
	assert.True(t, ok, "expected oauth2 transport for gh CLI config client")
}

func TestNewGHClient_WithInvalidGHAppConfig(t *testing.T) {
	// Set invalid PEM to trigger the ghinstallation error path
	t.Setenv("APP_ID", "12345")
	t.Setenv("INSTALLATION_ID", "67890")
	t.Setenv("APP_PEM", "invalid-pem-data")

	// This will attempt to create ghinstallation transport with invalid PEM
	// The function logs an error but still returns a client
	client := NewGHClient()
	require.NotNil(t, client)
	assert.Nil(t, client.Raw().Client().Transport, "expected nil transport when GH App PEM is invalid")
}

func TestGetGHAppConfig_WithDotEnvFile(t *testing.T) {
	// Use t.Chdir so the working directory is restored automatically and
	// the test is safe to run in parallel with other packages.
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// os.Unsetenv is required here because t.Setenv("", "") would cause
	// koanf's env provider to override .env values with an empty string.
	// This test must NOT be run with t.Parallel().
	unsetAppEnv := func(t *testing.T) {
		t.Helper()
		for _, key := range []string{"APP_ID", "INSTALLATION_ID", "APP_PEM"} {
			if err := os.Unsetenv(key); err != nil {
				t.Errorf("failed to unset %s: %v", key, err)
			}
		}
		t.Cleanup(func() {
			for _, key := range []string{"APP_ID", "INSTALLATION_ID", "APP_PEM"} {
				if err := os.Unsetenv(key); err != nil {
					t.Errorf("cleanup: failed to unset %s: %v", key, err)
				}
			}
		})
	}

	tests := []struct {
		name       string
		envContent string
		wantExists bool
		wantAppID  int64
		wantInstID int64
	}{
		{
			name:       "valid .env with all fields",
			envContent: "APP_ID=99999\nINSTALLATION_ID=88888\nAPP_PEM=-----BEGIN RSA PRIVATE KEY-----\\ndata\\n-----END RSA PRIVATE KEY-----\n",
			wantExists: true,
			wantAppID:  99999,
			wantInstID: 88888,
		},
		{
			name:       "incomplete .env file",
			envContent: "APP_ID=11111\n",
			wantExists: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unsetAppEnv(t)

			err := os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(tc.envContent), 0644)
			require.NoError(t, err)

			cc, exists := getGHAppConfig()
			assert.Equal(t, tc.wantExists, exists)
			if tc.wantExists {
				assert.Equal(t, tc.wantAppID, cc.appID)
				assert.Equal(t, tc.wantInstID, cc.instID)
			}
		})
	}
}
