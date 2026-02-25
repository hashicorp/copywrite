// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package github

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v45/github"
	"github.com/hashicorp/go-hclog"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/oauth2"
)

var logger = hclog.L()

// GHClient is a wrapper to access Github Client's API endpoints easily
type GHClient struct {
	gh *github.Client
}

// GHClientConfig is the configuration portion of the GH client (mostly for GH App)
type GHClientConfig struct {
	appID  int64
	instID int64
	appPEM string
}

// Raw is a util function to access the Github Client directly
func (c *GHClient) Raw() *github.Client {
	return c.gh
}

// getGHAppConfig looks for Github App configurations and sets them appropriately.
// if configuration is not found, return false
func getGHAppConfig() (cc GHClientConfig, exists bool) {
	k := koanf.New(".")
	var err error

	// Start by loading any .env file that exists
	err = k.Load(file.Provider(".env"), dotenv.Parser())
	if err != nil {
		logger.Debug(fmt.Sprintf("Error reading .env configuration file: %v", err))
	}

	// If environment variable are present, give preference to them
	// (this will overwrite values for any keys that also exist in the .env file)
	err = k.Load(env.Provider("", ".", nil), nil)
	if err != nil {
		logger.Debug(fmt.Sprintf("Error reading environment variables: %v", err))
	}

	cc.appID = k.Int64("APP_ID")
	cc.instID = k.Int64("INSTALLATION_ID")
	cc.appPEM = strings.ReplaceAll(k.String("APP_PEM"), "\\n", "\n")

	if cc.appPEM != "" && cc.appID != 0 && cc.instID != 0 {
		return cc, true
	}

	// Nothing worked
	logger.Debug("Problem with retrieving Github App identifiers, skipping GHApp configuration")
	return GHClientConfig{}, false
}

// getGitHubCLIConfig attempts to find a GitHub CLI (gh) configuration in the
// user's home directory. If it encounters any problems doing so, or if the
// configuration is missing/malformed, it will exit early with exists = false
func getGitHubCLIConfig() (token string, exists bool) {
	// Use "/" as the delimiter instead of "." because the GH CLI uses "." in YAML
	// key names, such as "github.com:"
	var k = koanf.New("/")

	errorString := "Unable to retrieve GitHub authentication via gh CLI config"

	configPath, err := homedir.Expand("~/.config/gh/hosts.yml")
	if err != nil {
		return "", false
	}

	// Config file is in the following format:
	// ───────┬───────────────────────────────────────────────────────────
	//        │ File: /Users/octocat/.config/gh/hosts.yml
	// ───────┼───────────────────────────────────────────────────────────
	//    1   │ github.com:
	//    2   │     user: octocat
	//    3   │     oauth_token: gho_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
	//    4   │     git_protocol: https

	err = k.Load(file.Provider(configPath), yaml.Parser())
	if err != nil {
		logger.Debug(fmt.Sprintf("%v: %v", errorString, err))
		return "", false
	}

	// We only care about github.com as the host right now, not self-hosted GitHub
	token = k.String("github.com/oauth_token")
	if token != "" {
		return token, true
	}

	logger.Debug(errorString)
	return "", false
}

// NewGHClient uses the copyright Github App for client requests
func NewGHClient() *GHClient {

	// Shared transport to reuse TCP connections and default to background context
	tr := http.DefaultTransport
	ctx := context.Background()

	// First, let's see if we can use GitHub App creds
	// This serves the use case of running as `hashicorp-copywrite[bot]` for
	// automatically scanning repos on a periodic basis.
	if cc, exists := getGHAppConfig(); exists {
		itr, err := ghinstallation.New(tr, cc.appID, cc.instID, []byte(cc.appPEM))
		if err != nil {
			logger.Error("Problem instantiating GH App transport")
		}

		client := github.NewClient(&http.Client{Transport: itr})
		logger.Info("Successfully established GH App client, requests will be made from hashicorp-copywrite.")
		return &GHClient{gh: client}
	}

	// If GitHub App creds can't be found or are malformed, fall back to using
	// the `GITHUB_TOKEN` environment variable.
	// This is a common use case for per-repo GitHub Actions, as an example
	if token, exists := os.LookupEnv("GITHUB_TOKEN"); exists {
		logger.Info("Using discovered Github PAT")

		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)
		return &GHClient{gh: github.NewClient(tc)}
	}

	// Fallback to seeing if the user happens to have the GitHub CLI tool (gh)
	// installed, at which point we can examine its config and extract a token
	if token, exists := getGitHubCLIConfig(); exists {
		logger.Info("Using discovered GitHub CLI Config Token")

		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)
		return &GHClient{gh: github.NewClient(tc)}
	}

	// If all else fails, fallback to an unauthenticated client
	// This only gives access to public information
	logger.Info("No Github auth credentials found, using unauthenticated GH Client")
	return &GHClient{gh: github.NewClient(nil)}
}
