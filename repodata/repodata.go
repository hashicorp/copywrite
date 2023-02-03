// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package repodata

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/google/go-github/v45/github"
	gh "github.com/hashicorp/copywrite/github"
	"github.com/hashicorp/go-hclog"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
)

// GetRepos retrieves the repo data and places it into an array
func GetRepos(githubOrganization string) ([]*github.Repository, error) {
	client := gh.NewGHClient().Raw()

	// list public repositories for org
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100}, // 100 is the max page size
		Type:        "public",
	}

	// pagination to always retrieve the exact number of repos and all metadata regarding them
	var allRepos []*github.Repository
	for {
		repos, current, err := client.Repositories.ListByOrg(context.Background(), githubOrganization, opt)
		if err != nil {
			hclog.L().Error(err.Error())

			return []*github.Repository{}, err
		}

		// append to the master list of repos
		allRepos = append(allRepos, repos...)

		// check if no more pages before continuing pagination
		if current.NextPage == 0 {
			break
		}
		opt.Page = current.NextPage
	}
	return allRepos, nil
}

// FilterRepos returns a new array of repo structs that only has non-archived repos
func FilterRepos(repos []*github.Repository) []*github.Repository {
	predicate := func(v *github.Repository, i int) bool {
		// Repo structs occasionally don't have the `Archived` key set. In these
		// cases, default to including the repo as it is categorically not archived
		return v.Archived == nil || !*v.Archived
	}
	return lo.Filter(repos, predicate)
}

// Transform takes in an array of repo structs and transforms it into an array of repo maps with attributes as strings
func Transform(repos []*github.Repository) ([]map[string]interface{}, error) {
	// place all the metaData types into the csvData array
	var structRepos []map[string]interface{}
	for _, repo := range repos {
		//turn the repo struct into a map and append
		repomap := map[string]interface{}{}
		err := mapstructure.Decode(repo, &repomap)
		if err != nil {
			return []map[string]interface{}{}, err
		}

		// Transform values into strings for easier parsing
		for _, value := range lo.Keys(repomap) {
			//type assertion to index into the map and deference pointer value
			pointer := repomap[value]
			data := ""

			//pointer will never be nil, but the underlying value may be
			if !reflect.ValueOf(pointer).IsNil() {
				switch pointer := pointer.(type) {
				case *string:
					data = *pointer
				case *github.License:
					data = *pointer.Key
				case *github.Timestamp: // time will never be nil
					data = pointer.Time.String()
				default:
				}
			}
			repomap[value] = data
		}
		structRepos = append(structRepos, repomap)
	}

	return structRepos, nil
}

// ValidateInputFields takes the module input flag string, splits it by comma, and then checks to make sure each data type exists in the Repository struct
func ValidateInputFields(fields string) ([]string, error) {
	//split by comma and trim whitespace
	values := strings.Split(fields, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}

	// convert to map
	base := new(github.Repository)
	repomap := map[string]interface{}{}
	err := mapstructure.Decode(base, &repomap)
	if err != nil {
		return []string{}, err
	}

	for _, value := range values {
		//make sure the data type exists in the struct
		_, exist := repomap[value]
		if !exist {
			hclog.L().Error("Data type does not exist in repository struct", "type", value)
			return []string{}, errors.New("Data type " + value + " does not exist in repository struct")
		}

		// if the data type is not currently supported
		switch repomap[value].(type) {
		case *string:
		case *github.License:
		case *github.Timestamp:
		default:
			return []string{}, errors.New("Data type " + value + " is currently not supported")
		}
	}

	return values, nil
}
