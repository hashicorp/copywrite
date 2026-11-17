// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Git ignore state is collected once per headers invocation and shared by both
// header-processing passes.
// gitIgnoredFiles contains paths reported by Git as ignored for a scan root.
// Git remains the source of truth so repository, info/exclude, and global rules
// all have identical semantics to the user's Git commands.
type gitIgnoredFiles struct {
	paths       map[string]struct{}
	nestedRepos map[string]struct{}
}

// newGitIgnoredFiles loads Git's standard ignore sources and identifies nested
// repositories that should be treated as scan boundaries.
func newGitIgnoredFiles(scanRoot string) (*gitIgnoredFiles, error) {
	// Ask Git to apply repository, info/exclude, and configured global excludes.
	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard", "-z")
	cmd.Dir = scanRoot
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("unable to read Git ignored files: %w", err)
	}

	ignored := &gitIgnoredFiles{
		paths:       make(map[string]struct{}),
		nestedRepos: make(map[string]struct{}),
	}

	// Store exact ignored paths so negated Git rules remain effective.
	for _, path := range splitNUL(output) {
		if path == "" {
			continue
		}
		ignored.paths[normalizeGitPath(strings.TrimSuffix(path, "/"))] = struct{}{}
	}

	// Find nested repositories and keep the parent walk from entering them.
	err = filepath.WalkDir(scanRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == scanRoot {
			return nil
		}
		if entry.Name() == ".git" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		gitPath := filepath.Join(path, ".git")
		if _, statErr := os.Lstat(gitPath); statErr == nil {
			ignored.nestedRepos[normalizeGitPath(path)] = struct{}{}
			if entry.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to find nested Git repositories: %w", err)
	}
	return ignored, nil
}

// splitNUL parses Git's NUL-delimited output without treating path contents as
// separators. This keeps paths containing spaces or newlines intact.
func splitNUL(data []byte) []string {
	parts := strings.Split(string(data), "\x00")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		return parts[:len(parts)-1]
	}
	return parts
}

// IsIgnored reports whether Git explicitly identified path as ignored.
func (g *gitIgnoredFiles) IsIgnored(path string) bool {
	if g == nil {
		return false
	}
	_, ok := g.paths[normalizeGitPath(path)]
	return ok
}

// IsNestedRepo reports whether path is the root of a nested Git repository.
func (g *gitIgnoredFiles) IsNestedRepo(path string) bool {
	if g == nil {
		return false
	}
	_, ok := g.nestedRepos[normalizeGitPath(path)]
	return ok
}

// normalizeGitPath uses Git's slash-separated path representation on all OSes.
func normalizeGitPath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}
