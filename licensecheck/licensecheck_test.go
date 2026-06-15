// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package licensecheck

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTempFiles(t *testing.T, fileNames []string) (dirPath string, filePaths []string) {
	AppFs := afero.NewOsFs()
	tempDir := t.TempDir()
	// create file
	filePaths = lo.Map(fileNames, func(fileName string, i int) string {
		filePath := filepath.Join(tempDir, fileName)
		err := AppFs.MkdirAll(filepath.Dir(filePath), 0755)
		require.NoError(t, err)
		err = afero.WriteFile(AppFs, filePath, []byte("Bob Loblaw's Law Blog"), 0644)
		require.NoError(t, err)
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

func TestEnsureCorrectName_ErrorHandling(t *testing.T) {
	t.Run("error when file does not exist", func(t *testing.T) {
		_, err := EnsureCorrectName("/nonexistent/path/license.txt")
		assert.NotNil(t, err)
	})
}

func TestAddHeader(t *testing.T) {
	AppFs := afero.NewOsFs()

	t.Run("add header to empty file", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.txt")
		err := afero.WriteFile(AppFs, filePath, []byte(""), 0644)
		require.NoError(t, err)

		header := "Copyright (c) 2023 Test Corp"
		err = AddHeader(filePath, header)
		require.NoError(t, err)

		// Read file and verify header was added
		content, err := afero.ReadFile(AppFs, filePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), header)
		// Should have double newline after header, with nothing following it
		assert.Equal(t, header+"\n\n", string(content), "file should contain only the header followed by a blank line with nothing after")
	})

	t.Run("add header to file with existing content", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.txt")
		originalContent := "This is the original file content"
		err := afero.WriteFile(AppFs, filePath, []byte(originalContent), 0644)
		require.NoError(t, err)

		header := "Copyright (c) 2023 Test Corp"
		err = AddHeader(filePath, header)
		require.NoError(t, err)

		// Read file and verify header was prepended
		content, err := afero.ReadFile(AppFs, filePath)
		require.NoError(t, err)
		// File must start with the exact header followed by a blank line
		assert.True(t, strings.HasPrefix(string(content), header+"\n\n"),
			"file should start with the exact header followed by a blank line")
		assert.True(t, strings.HasSuffix(string(content), originalContent),
			"original content should follow the header unchanged")
	})

	t.Run("add multi-line header", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.txt")
		originalContent := "This is the original file content"
		err := afero.WriteFile(AppFs, filePath, []byte(originalContent), 0644)
		require.NoError(t, err)

		header := "Copyright (c) 2023 Test Corp\nSPDX-License-Identifier: MPL-2.0"
		err = AddHeader(filePath, header)
		require.NoError(t, err)

		content, err := afero.ReadFile(AppFs, filePath)
		require.NoError(t, err)
		expectedPrefix := header + "\n\n"
		assert.True(t, strings.HasPrefix(string(content), expectedPrefix),
			"file should start with the multi-line header followed by a blank line")
		assert.True(t, strings.HasSuffix(string(content), originalContent),
			"original content should follow the header unchanged")
		assert.Equal(t, expectedPrefix+originalContent, string(content),
			"file should contain only the header and original content with nothing in between")
	})

	t.Run("error on non-existent file", func(t *testing.T) {
		header := "Copyright (c) 2023 Test Corp"
		err := AddHeader("/nonexistent/path/file.txt", header)
		assert.NotNil(t, err)
	})
}

func TestAddLicenseFile(t *testing.T) {
	AppFs := afero.NewOsFs()

	t.Run("create LICENSE file with MPL-2.0", func(t *testing.T) {
		tempDir := t.TempDir()
		licensePath, err := AddLicenseFile(tempDir, "MPL-2.0")
		assert.Nil(t, err)
		assert.NotEmpty(t, licensePath)

		// Verify file exists
		fileExists, err := afero.Exists(AppFs, licensePath)
		require.NoError(t, err)
		assert.True(t, fileExists)

		// Verify content contains MPL-2.0 license text
		content, err := afero.ReadFile(AppFs, licensePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Mozilla Public License")
	})

	t.Run("create LICENSE file with Apache-2.0", func(t *testing.T) {
		tempDir := t.TempDir()
		licensePath, err := AddLicenseFile(tempDir, "Apache-2.0")
		assert.Nil(t, err)

		content, err := afero.ReadFile(AppFs, licensePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Apache License")
	})

	t.Run("create LICENSE file with MIT", func(t *testing.T) {
		tempDir := t.TempDir()
		licensePath, err := AddLicenseFile(tempDir, "MIT")
		assert.Nil(t, err)

		content, err := afero.ReadFile(AppFs, licensePath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Permission is hereby granted")
	})

	t.Run("error on unknown SPDX ID", func(t *testing.T) {
		tempDir := t.TempDir()
		licensePath, err := AddLicenseFile(tempDir, "UNKNOWN-LICENSE-99")
		assert.NotNil(t, err)
		assert.Empty(t, licensePath)
		assert.Contains(t, err.Error(), "unknown SPDX license ID")
	})

	t.Run("error on invalid directory path", func(t *testing.T) {
		licensePath, err := AddLicenseFile("/nonexistent/invalid/path", "MPL-2.0")
		assert.NotNil(t, err)
		assert.Empty(t, licensePath)
	})

	t.Run("returned path is absolute", func(t *testing.T) {
		tempDir := t.TempDir()
		licensePath, err := AddLicenseFile(tempDir, "MPL-2.0")
		assert.Nil(t, err)
		assert.True(t, filepath.IsAbs(licensePath))
	})

	t.Run("file is named LICENSE", func(t *testing.T) {
		tempDir := t.TempDir()
		licensePath, err := AddLicenseFile(tempDir, "Apache-2.0")
		assert.Nil(t, err)
		assert.Equal(t, "LICENSE", filepath.Base(licensePath))
	})
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
			input:          []string{"LICENSE", "subdir/blah.txt"},
			expectedOutput: []string{"LICENSE"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			tempDir, _ := createTempFiles(t, tt.input)
			// run test
			actualOutput, err := FindLicenseFiles(tempDir)
			require.NoError(t, err)
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

func TestFindLicenseFiles_ErrorHandling(t *testing.T) {
	t.Run("returns empty slice when directory doesn't exist", func(t *testing.T) {
		// When the directory doesn't exist, glob returns empty without error
		result, err := FindLicenseFiles("/nonexistent/directory/path")
		assert.Nil(t, err)
		assert.Equal(t, []string{}, result)
	})

	t.Run("handles subdirectories correctly", func(t *testing.T) {
		AppFs := afero.NewOsFs()
		tempDir := t.TempDir()

		// Create a subdirectory with a LICENSE file
		subDir := filepath.Join(tempDir, "subdir")
		err := AppFs.MkdirAll(subDir, 0755)
		require.NoError(t, err)
		err = afero.WriteFile(AppFs, filepath.Join(subDir, "LICENSE"), []byte("sublicense"), 0644)
		require.NoError(t, err)

		// FindLicenseFiles should only find files in the top-level directory, not subdirs
		result, err := FindLicenseFiles(tempDir)
		require.NoError(t, err)
		assert.Equal(t, []string{}, result)
	})
}
