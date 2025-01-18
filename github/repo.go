// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package github

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v45/github"
)

// GHRepo is a repo
type GHRepo struct {
	Owner string
	Name  string
}

// DiscoverRepo attempts to find if the current directory is related to a
// GitHub repo and, if so, what the organization name and repo name are
//
// This function will return an error if more than one GitHub repos are
// associated with the given folder. This can happen if multiple git upstreams
// defined.
func DiscoverRepo() (GHRepo, error) {
	repo, err := repository.Current()
	if err != nil {
		return GHRepo{}, fmt.Errorf("unable to determine if the current directory relates to a GitHub repo: %v", err)
	}

	return GHRepo{
		Name:  repo.Name,
		Owner: repo.Owner,
	}, nil
}

// GetRepoCreationYear takes in a repo and uses the GitHub API to determine the
// year it was created.
//
// This is typically used to infer when the original copyright date is, but it
// should be noted that certain circumstances can cause the value returned may
// be unsuitable for use as the original copyright date. Specifically, it is
// possible for an older project to be recreated in a newer repo, or for the
// repo to be made public at a later date, at which point the year of creation
// and year of copyright may differ.
func GetRepoCreationYear(client *github.Client, repo GHRepo) (int, error) {
	data, _, err := client.Repositories.Get(context.Background(), repo.Owner, repo.Name)
	if err != nil {
		return 0, err
	}

	year := data.CreatedAt.Year()
	if year == 0 {
		return 0, fmt.Errorf("year returned from GitHub API is invalid \"%v\"", year)
	}

	return year, nil
}
