// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package licensecheck

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/samber/lo"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func createTempFiles(t *testing.T, fileNames []string) (dirPath string, filePaths []string) {
	AppFs := afero.NewOsFs()
	tempDir := t.TempDir()
	// create file
	filePaths = lo.Map(fileNames, func(fileName string, i int) string {
		filePath := filepath.Join(tempDir, fileName)
		_ = afero.WriteFile(AppFs, filePath, []byte("Bob Loblaw's Law Blog"), 0644)
		return filePath
	})

	return tempDir, filePaths
}

func TestEnsureCorrectName(t *testing.T) {
	AppFs := afero.NewOsFs()

	cases := []struct {
		description   string
		filesToCreate []string
	}{
		{
			description:   "Correctly named file should be left alone",
			filesToCreate: []string{"LICENSE"},
		},
		{
			description:   "License file with .txt extension should be renamed",
			filesToCreate: []string{"LICENSE.txt"},
		},
		{
			description:   "License file with .md extension should be renamed",
			filesToCreate: []string{"LICENSE.md"},
		},
		{
			description:   "Lowercase file name should be renamed",
			filesToCreate: []string{"license"},
		},
		{
			description:   "Lowercase file name with .txt extension should be renamed",
			filesToCreate: []string{"license.txt"},
		},
		{
			description:   "Lowercase file name with .md extension should be renamed",
			filesToCreate: []string{"license.md"},
		},
		{
			description:   "Oddly cased file without extension should be renamed",
			filesToCreate: []string{"LiCeNsE"},
		},
		{
			description:   "Oddly cased file with .txt extension should be renamed",
			filesToCreate: []string{"LiCeNsE.TxT"},
		},
		{
			description:   "Oddly cased file with .md extension should be renamed",
			filesToCreate: []string{"LiCeNsE.Md"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			tempDir, filePaths := createTempFiles(t, tt.filesToCreate)
			// run test
			_, err := EnsureCorrectName(filePaths[0])
			assert.Nil(t, err)
			// validate file was renamed successfully
			fileExists, err := afero.Exists(AppFs, filepath.Join(tempDir, "LICENSE"))
			assert.True(t, fileExists)
			assert.Nil(t, err)
		})
	}
}

func TestAddHeader(t *testing.T) {
	// stub
	t.Skip()
}

func TestAddLicenseFile(t *testing.T) {
	// stub
	t.Skip()
}

func sortSlice(input *[]string) {
	sort.Slice(*input, func(i, j int) bool {
		return (*input)[i] < (*input)[j]
	})
}

func TestFindLicenseFiles(t *testing.T) {

	cases := []struct {
		description    string
		input          []string
		expectedOutput []string
	}{
		{
			description:    "Empty directory should have no matches",
			input:          []string{},
			expectedOutput: []string{},
		},
		{
			description:    "Uppercase file without extension is matched",
			input:          []string{"LICENSE"},
			expectedOutput: []string{"LICENSE"},
		},
		{
			description:    "Uppercase file with .txt extension is matched",
			input:          []string{"LICENSE.txt"},
			expectedOutput: []string{"LICENSE.txt"},
		},
		{
			description:    "Uppercase file with .md extension is matched",
			input:          []string{"LICENSE.md"},
			expectedOutput: []string{"LICENSE.md"},
		},
		{
			description:    "Multiple licenses with various extensions are matched",
			input:          []string{"LICENSE", "LICENSE.txt", "LICENSE.md"},
			expectedOutput: []string{"LICENSE", "LICENSE.txt", "LICENSE.md"},
		},
		{
			description:    "Matches are case-insensitive",
			input:          []string{"LiCenSe", "LICenSe.TXT", "liCense.mD"},
			expectedOutput: []string{"LiCenSe", "LICenSe.TXT", "liCense.mD"},
		},
		{
			description:    "Matches are case-insensitive",
			input:          []string{"LiCenSe", "LICenSe.TXT", "liCense.mD"},
			expectedOutput: []string{"LiCenSe", "LICenSe.TXT", "liCense.mD"},
		},
		{
			description:    "Don't match files that are prefixed with other stuff",
			input:          []string{"coollicense", "coollicense.txt", "coollicense.md"},
			expectedOutput: []string{},
		},
		{
			description:    "Don't match files with non-standard extensions",
			input:          []string{"LICENSE.", "LICENSE.asdf", "LICENSE.csv", "LICENSE.txta", "LICENSE.mdx"},
			expectedOutput: []string{},
		},
		{
			description:    "Don't match directories",
			input:          []string{"LICENSE", "license/blah.txt"},
			expectedOutput: []string{"LICENSE"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			tempDir, _ := createTempFiles(t, tt.input)
			// run test
			actualOutput, err := FindLicenseFiles(tempDir)
			// validate file was renamed successfully
			expectedOutputPaths := lo.Map(tt.expectedOutput, func(p string, _ int) string {
				return filepath.Join(tempDir, p)
			})

			// sort both actual and expected output, as no guarantees are given on file ordering
			sortSlice(&expectedOutputPaths)
			sortSlice(&actualOutput)

			assert.Equal(t, expectedOutputPaths, actualOutput, tt.description)
			assert.Nil(t, err)
		})
	}
}
