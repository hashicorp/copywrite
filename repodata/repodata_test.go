// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package repodata

import (
	"testing"

	"github.com/google/go-github/v45/github"
	"github.com/stretchr/testify/assert"
)

// to help with making archived and non archived repos for testing
func makeArchivedRepo(flag bool) *github.Repository {
	repo := new(github.Repository)
	repo.Archived = &flag
	return repo
}

func makeNilArchivedRepo() *github.Repository {
	repo := new(github.Repository)
	repo.Archived = nil
	return repo
}

func TestFilterRepos(t *testing.T) {

	cases := []struct {
		description    string
		actualresult   []*github.Repository
		expectedresult []*github.Repository
	}{
		{
			description:    "archived repo should be removed",
			actualresult:   FilterRepos([]*github.Repository{makeArchivedRepo(true)}),
			expectedresult: []*github.Repository{},
		},
		{
			description:    "non archived repo should still remain",
			actualresult:   FilterRepos([]*github.Repository{makeArchivedRepo(false)}),
			expectedresult: []*github.Repository{makeArchivedRepo(false)},
		},
		{
			description:    "archived repo should be gone, non archived repo should stay",
			actualresult:   FilterRepos([]*github.Repository{makeArchivedRepo(true), makeArchivedRepo(false)}),
			expectedresult: []*github.Repository{makeArchivedRepo(false)},
		},
		{
			description:    "repo struct missing the archived key should still remain",
			actualresult:   FilterRepos([]*github.Repository{makeNilArchivedRepo()}),
			expectedresult: []*github.Repository{makeNilArchivedRepo()},
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			assert.Equal(t, tt.expectedresult, tt.actualresult, tt.description)
		})
	}

}

func TestValidateInputFields(t *testing.T) {
	cases := []struct {
		description    string
		inputString    string
		expectedresult []string
	}{
		{
			description:    "default flag values all exist",
			inputString:    "Name,   HTMLURL, License,         CreatedAt",
			expectedresult: []string{"Name", "HTMLURL", "License", "CreatedAt"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			actualResult, err := ValidateInputFields(tt.inputString)
			assert.Equal(t, tt.expectedresult, actualResult, tt.description)
			assert.Nil(t, err)
		})
	}

	// test errors
	errorCases := []struct {
		description    string
		inputString    string
		expectedresult string
	}{
		{
			description:    "data type does not exist in struct",
			inputString:    "Name,   HTMLURL, License,         Dave",
			expectedresult: "Data type Dave does not exist in repository struct",
		},
		{
			description:    "data type isn't supported",
			inputString:    "Name,   HTMLURL, License,         ForksCount",
			expectedresult: "Data type ForksCount is currently not supported",
		},
	}

	for _, tt := range errorCases {
		t.Run(tt.description, func(t *testing.T) {
			actualResult, err := ValidateInputFields(tt.inputString)
			assert.Equal(t, tt.expectedresult, err.Error(), tt.description)
			assert.Equal(t, []string{}, actualResult, "should return empty after error")
		})
	}
}
