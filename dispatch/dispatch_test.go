// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package dispatch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v45/github"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMockServer is a helper function to mock the GitHub API endpoint.
func setupMockServer(t *testing.T) (*github.Client, *http.ServeMux, *httptest.Server) {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	client := github.NewClient(server.Client())
	u, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = u
	return client, mux, server
}

func TestWaitRunFinished(t *testing.T) {
	baseOpts := Options{
		SecondsBetweenPolls: 0, // 0 speeds up the tests
		MaxAttempts:         2,
		Logger:              hclog.NewNullLogger(),
		GitHubOwner:         "testOrg",
		GitHubRepo:          "testRepo",
	}

	t.Run("short circuit completed", func(t *testing.T) {
		client, _, server := setupMockServer(t)
		defer server.Close()

		run := github.WorkflowRun{
			ID:     github.Int64(1),
			Name:   github.String("test-run"),
			Status: github.String("completed"),
		}

		err := WaitRunFinished(client, baseOpts, run)
		assert.NoError(t, err)
	})

	t.Run("polls and completes", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		run := github.WorkflowRun{
			ID:     github.Int64(2),
			Name:   github.String("test-run"),
			Status: github.String("queued"),
		}

		calls := 0
		mux.HandleFunc("/repos/testOrg/testRepo/actions/runs/2", func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				if _, err := fmt.Fprint(w, `{"id": 2, "status": "in_progress"}`); err != nil {
					t.Errorf("mock write failed: %v", err)
				}
			} else {
				if _, err := fmt.Fprint(w, `{"id": 2, "status": "completed"}`); err != nil {
					t.Errorf("mock write failed: %v", err)
				}
			}
		})

		err := WaitRunFinished(client, baseOpts, run)
		assert.NoError(t, err)
		assert.Equal(t, 2, calls)
	})

	t.Run("unrepairable state", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		run := github.WorkflowRun{
			ID:     github.Int64(3),
			Name:   github.String("test-run"),
			Status: github.String("queued"),
		}

		mux.HandleFunc("/repos/testOrg/testRepo/actions/runs/3", func(w http.ResponseWriter, r *http.Request) {
			if _, err := fmt.Fprint(w, `{"id": 3, "status": "failed"}`); err != nil {
				t.Errorf("mock write failed: %v", err)
			}
		})

		err := WaitRunFinished(client, baseOpts, run)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unrepairable state")
	})

	t.Run("timeout", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		run := github.WorkflowRun{
			ID:     github.Int64(4),
			Name:   github.String("test-run"),
			Status: github.String("queued"),
		}

		mux.HandleFunc("/repos/testOrg/testRepo/actions/runs/4", func(w http.ResponseWriter, r *http.Request) {
			if _, err := fmt.Fprint(w, `{"id": 4, "status": "in_progress"}`); err != nil {
				t.Errorf("mock write failed: %v", err)
			}
		})

		err := WaitRunFinished(client, baseOpts, run)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})

	t.Run("api error", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		run := github.WorkflowRun{
			ID:     github.Int64(5),
			Name:   github.String("test-run"),
			Status: github.String("queued"),
		}

		mux.HandleFunc("/repos/testOrg/testRepo/actions/runs/5", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		err := WaitRunFinished(client, baseOpts, run)
		assert.Error(t, err)
	})
}

func TestFindRun(t *testing.T) {
	baseOpts := Options{
		SecondsBetweenPolls: 0, // 0 speeds up the tests
		MaxAttempts:         2,
		Logger:              hclog.NewNullLogger(),
		GitHubOwner:         "testOrg",
		GitHubRepo:          "testRepo",
		BranchRef:           "main",
		WorkflowFileName:    "test.yml",
	}

	t.Run("finds on first attempt", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		mux.HandleFunc("/repos/testOrg/testRepo/actions/workflows/test.yml/runs", func(w http.ResponseWriter, r *http.Request) {
			if _, err := fmt.Fprint(w, `{"workflow_runs": [{"id": 1, "name": "my-run", "status": "queued"}]}`); err != nil {
				t.Errorf("mock write failed: %v", err)
			}
		})

		run, err := FindRun(client, baseOpts, "my-run")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), *run.ID)
	})

	t.Run("polls and finds", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		calls := 0
		mux.HandleFunc("/repos/testOrg/testRepo/actions/workflows/test.yml/runs", func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				if _, err := fmt.Fprint(w, `{"workflow_runs": []}`); err != nil {
					t.Errorf("mock write failed: %v", err)
				}
			} else {
				if _, err := fmt.Fprint(w, `{"workflow_runs": [{"id": 2, "name": "my-run-2", "status": "queued"}]}`); err != nil {
					t.Errorf("mock write failed: %v", err)
				}
			}
		})

		run, err := FindRun(client, baseOpts, "my-run-2")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), *run.ID)
		assert.Equal(t, 2, calls)
	})

	t.Run("timeout", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		mux.HandleFunc("/repos/testOrg/testRepo/actions/workflows/test.yml/runs", func(w http.ResponseWriter, r *http.Request) {
			if _, err := fmt.Fprint(w, `{"workflow_runs": []}`); err != nil {
				t.Errorf("mock write failed: %v", err)
			}
		})

		_, err := FindRun(client, baseOpts, "my-run-3")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})

	t.Run("api error", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		mux.HandleFunc("/repos/testOrg/testRepo/actions/workflows/test.yml/runs", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		_, err := FindRun(client, baseOpts, "my-run-4")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error attempting to find")
	})
}

func TestWorker(t *testing.T) {
	baseOpts := Options{
		SecondsBetweenPolls: 0,
		MaxAttempts:         2,
		Logger:              hclog.NewNullLogger(),
		GitHubOwner:         "testOrg",
		GitHubRepo:          "testRepo",
		BatchID:             "batch123",
		WorkflowFileName:    "test.yml",
		BranchRef:           "main",
	}

	t.Run("success", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		mux.HandleFunc("/repos/testOrg/testRepo/actions/workflows/test.yml/dispatches", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		mux.HandleFunc("/repos/testOrg/testRepo/actions/workflows/test.yml/runs", func(w http.ResponseWriter, r *http.Request) {
			if _, err := fmt.Fprint(w, `{"workflow_runs": [{"id": 1, "name": "batch123: Audit test-repo", "status": "queued"}]}`); err != nil {
				t.Errorf("mock write failed: %v", err)
			}
		})
		mux.HandleFunc("/repos/testOrg/testRepo/actions/runs/1", func(w http.ResponseWriter, r *http.Request) {
			if _, err := fmt.Fprint(w, `{"id": 1, "status": "completed"}`); err != nil {
				t.Errorf("mock write failed: %v", err)
			}
		})

		jobs := make(chan string, 1)
		results := make(chan Result, 1)
		jobs <- "test-repo"
		close(jobs)

		Worker(client, baseOpts, 1, jobs, results)

		res := <-results
		assert.True(t, res.Success)
		assert.NoError(t, res.Error)
		assert.Equal(t, "test-repo", res.Name)
	})

	t.Run("dispatch fails", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		mux.HandleFunc("/repos/testOrg/testRepo/actions/workflows/test.yml/dispatches", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		jobs := make(chan string, 1)
		results := make(chan Result, 1)
		jobs <- "test-repo"
		close(jobs)

		Worker(client, baseOpts, 1, jobs, results)

		res := <-results
		assert.False(t, res.Success)
		assert.Error(t, res.Error)
	})

	t.Run("find fails", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		// Dispatch works, but the run search throws an error
		mux.HandleFunc("/repos/testOrg/testRepo/actions/workflows/test.yml/dispatches", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		mux.HandleFunc("/repos/testOrg/testRepo/actions/workflows/test.yml/runs", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		jobs := make(chan string, 1)
		results := make(chan Result, 1)
		jobs <- "test-repo"
		close(jobs)

		Worker(client, baseOpts, 1, jobs, results)

		res := <-results
		assert.False(t, res.Success)
		assert.Error(t, res.Error)
		assert.Contains(t, res.Error.Error(), "error attempting to find")
	})

	t.Run("wait fails with unrepairable state", func(t *testing.T) {
		client, mux, server := setupMockServer(t)
		defer server.Close()

		// Dispatch succeeds
		mux.HandleFunc("/repos/testOrg/testRepo/actions/workflows/test.yml/dispatches", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		// FindRun succeeds
		mux.HandleFunc("/repos/testOrg/testRepo/actions/workflows/test.yml/runs", func(w http.ResponseWriter, r *http.Request) {
			if _, err := fmt.Fprint(w, `{"workflow_runs": [{"id": 10, "name": "batch123: Audit test-repo", "status": "queued"}]}`); err != nil {
				t.Errorf("mock write failed: %v", err)
			}
		})
		// WaitRunFinished hits an unrepairable state
		mux.HandleFunc("/repos/testOrg/testRepo/actions/runs/10", func(w http.ResponseWriter, r *http.Request) {
			if _, err := fmt.Fprint(w, `{"id": 10, "status": "failed"}`); err != nil {
				t.Errorf("mock write failed: %v", err)
			}
		})

		jobs := make(chan string, 1)
		results := make(chan Result, 1)
		jobs <- "test-repo"
		close(jobs)

		Worker(client, baseOpts, 1, jobs, results)

		res := <-results
		assert.False(t, res.Success)
		assert.Error(t, res.Error)
		assert.Contains(t, res.Error.Error(), "unrepairable state")
		assert.Equal(t, "test-repo", res.Name)
	})
}
