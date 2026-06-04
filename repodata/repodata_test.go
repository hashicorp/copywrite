// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package repodata

import (
	"testing"
	"time"

	"github.com/google/go-github/v45/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestTransform(t *testing.T) {
	t.Run("empty array returns empty result", func(t *testing.T) {
		result, err := Transform([]*github.Repository{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("transform repo with string fields", func(t *testing.T) {
		name := "test-repo"
		url := "https://github.com/test/repo"
		repo := &github.Repository{
			Name:    &name,
			HTMLURL: &url,
		}

		result, err := Transform([]*github.Repository{repo})
		require.NoError(t, err)
		require.Len(t, result, 1)

		assert.Equal(t, "test-repo", result[0]["Name"])
		assert.Equal(t, "https://github.com/test/repo", result[0]["HTMLURL"])
	})

	t.Run("transform repo with nil string fields", func(t *testing.T) {
		repo := &github.Repository{
			Name:    nil,
			HTMLURL: nil,
		}

		result, err := Transform([]*github.Repository{repo})
		require.NoError(t, err)
		require.Len(t, result, 1)

		// Nil pointers should be transformed to empty strings
		assert.Equal(t, "", result[0]["Name"])
		assert.Equal(t, "", result[0]["HTMLURL"])
	})

	t.Run("transform repo with license", func(t *testing.T) {
		licenseKey := "mit"
		license := &github.License{
			Key: &licenseKey,
		}
		name := "licensed-repo"
		repo := &github.Repository{
			Name:    &name,
			License: license,
		}

		result, err := Transform([]*github.Repository{repo})
		require.NoError(t, err)
		require.Len(t, result, 1)

		assert.Equal(t, "licensed-repo", result[0]["Name"])
		assert.Equal(t, "mit", result[0]["License"])
	})

	t.Run("transform repo with nil license", func(t *testing.T) {
		name := "unlicensed-repo"
		repo := &github.Repository{
			Name:    &name,
			License: nil,
		}

		result, err := Transform([]*github.Repository{repo})
		require.NoError(t, err)
		require.Len(t, result, 1)

		assert.Equal(t, "unlicensed-repo", result[0]["Name"])
		assert.Equal(t, "", result[0]["License"])
	})

	t.Run("transform repo with timestamp", func(t *testing.T) {
		name := "timestamped-repo"
		testTime := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)
		timestamp := &github.Timestamp{Time: testTime}
		repo := &github.Repository{
			Name:      &name,
			CreatedAt: timestamp,
		}

		result, err := Transform([]*github.Repository{repo})
		require.NoError(t, err)
		require.Len(t, result, 1)

		assert.Equal(t, "timestamped-repo", result[0]["Name"])
		// Timestamp should be converted to string representation
		assert.Equal(t, testTime.String(), result[0]["CreatedAt"])
	})

	t.Run("transform repo with nil timestamp", func(t *testing.T) {
		name := "no-timestamp-repo"
		repo := &github.Repository{
			Name:      &name,
			CreatedAt: nil,
		}

		result, err := Transform([]*github.Repository{repo})
		require.NoError(t, err)
		require.Len(t, result, 1)

		assert.Equal(t, "no-timestamp-repo", result[0]["Name"])
		assert.Equal(t, "", result[0]["CreatedAt"])
	})

	t.Run("transform multiple repos", func(t *testing.T) {
		name1 := "repo-one"
		name2 := "repo-two"
		url1 := "https://github.com/test/one"
		url2 := "https://github.com/test/two"

		repos := []*github.Repository{
			{
				Name:    &name1,
				HTMLURL: &url1,
			},
			{
				Name:    &name2,
				HTMLURL: &url2,
			},
		}

		result, err := Transform(repos)
		require.NoError(t, err)
		require.Len(t, result, 2)

		assert.Equal(t, "repo-one", result[0]["Name"])
		assert.Equal(t, "https://github.com/test/one", result[0]["HTMLURL"])
		assert.Equal(t, "repo-two", result[1]["Name"])
		assert.Equal(t, "https://github.com/test/two", result[1]["HTMLURL"])
	})

	t.Run("transform repo with mixed field types", func(t *testing.T) {
		name := "complex-repo"
		url := "https://github.com/test/complex"
		licenseKey := "apache-2.0"
		license := &github.License{Key: &licenseKey}
		testTime := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
		timestamp := &github.Timestamp{Time: testTime}

		repo := &github.Repository{
			Name:      &name,
			HTMLURL:   &url,
			License:   license,
			CreatedAt: timestamp,
		}

		result, err := Transform([]*github.Repository{repo})
		require.NoError(t, err)
		require.Len(t, result, 1)

		assert.Equal(t, "complex-repo", result[0]["Name"])
		assert.Equal(t, "https://github.com/test/complex", result[0]["HTMLURL"])
		assert.Equal(t, "apache-2.0", result[0]["License"])
		assert.Equal(t, testTime.String(), result[0]["CreatedAt"])
	})

	t.Run("transform repo with all nil fields", func(t *testing.T) {
		repo := &github.Repository{
			Name:      nil,
			HTMLURL:   nil,
			License:   nil,
			CreatedAt: nil,
		}

		result, err := Transform([]*github.Repository{repo})
		require.NoError(t, err)
		require.Len(t, result, 1)

		// All nil fields should become empty strings
		assert.Equal(t, "", result[0]["Name"])
		assert.Equal(t, "", result[0]["HTMLURL"])
		assert.Equal(t, "", result[0]["License"])
		assert.Equal(t, "", result[0]["CreatedAt"])
	})
}
