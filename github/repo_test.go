// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	// "os/exec"
	"testing"
	"time"

	gogithub "github.com/google/go-github/v45/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRepoCreationYear(t *testing.T) {
	tests := []struct {
		name        string
		repo        GHRepo
		handler     func(w http.ResponseWriter, r *http.Request)
		wantYear    int
		wantErr     bool
		errContains string
	}{
		{
			name: "successful retrieval of creation year",
			repo: GHRepo{Owner: "hashicorp", Name: "copywrite"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "/repos/hashicorp/copywrite")
				assert.Equal(t, http.MethodGet, r.Method)

				repoData := &gogithub.Repository{
					CreatedAt: &gogithub.Timestamp{Time: time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC)},
				}
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(repoData))
			},
			wantYear: 2021,
			wantErr:  false,
		},
		{
			name: "repo created in 2023",
			repo: GHRepo{Owner: "org", Name: "project"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "/repos/org/project")

				repoData := &gogithub.Repository{
					CreatedAt: &gogithub.Timestamp{Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)},
				}
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(repoData))
			},
			wantYear: 2023,
			wantErr:  false,
		},
		{
			name: "API returns error",
			repo: GHRepo{Owner: "nonexistent", Name: "repo"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = fmt.Fprint(w, `{"message": "Not Found"}`)
			},
			wantYear:    0,
			wantErr:     true,
			errContains: "",
		},
		{
			name: "API returns malformed JSON",
			repo: GHRepo{Owner: "owner", Name: "repo"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{invalid json`)
			},
			wantYear: 0,
			wantErr:  true,
		},
		{
			name: "API returns server error",
			repo: GHRepo{Owner: "owner", Name: "repo"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprint(w, `{"message": "Internal Server Error"}`)
			},
			wantYear: 0,
			wantErr:  true,
		},
		{
			name: "empty owner and name",
			repo: GHRepo{Owner: "", Name: ""},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = fmt.Fprint(w, `{"message": "Not Found"}`)
			},
			wantYear: 0,
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/", tc.handler)
			server := httptest.NewServer(mux)
			defer server.Close()

			client, err := gogithub.NewEnterpriseClient(server.URL+"/", server.URL+"/", nil)
			require.NoError(t, err)

			year, err := GetRepoCreationYear(client, tc.repo)

			if tc.wantErr {
				assert.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantYear, year)
		})
	}
}

func TestDiscoverRepo_NotInGitRepo(t *testing.T) {
	// Use t.Chdir for process-safe directory change in tests
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	_, err := DiscoverRepo()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to determine if the current directory relates to a GitHub repo:")
}
