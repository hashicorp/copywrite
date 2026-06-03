// Copyright IBM Corp. 2023, 2025
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
		_ = afero.WriteFile(AppFs, filePath, []byte(""), 0644)

		header := "Copyright (c) 2023 Test Corp"
		err := AddHeader(filePath, header)
		assert.Nil(t, err)

		// Read file and verify header was added
		content, _ := afero.ReadFile(AppFs, filePath)
		assert.Contains(t, string(content), header)
		// Should have double newline after header
		assert.Contains(t, string(content), header+"\n\n")
	})

	t.Run("add header to file with existing content", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.txt")
		originalContent := "This is the original file content"
		_ = afero.WriteFile(AppFs, filePath, []byte(originalContent), 0644)

		header := "Copyright (c) 2023 Test Corp"
		err := AddHeader(filePath, header)
		assert.Nil(t, err)

		// Read file and verify header was prepended
		content, _ := afero.ReadFile(AppFs, filePath)
		assert.Contains(t, string(content), header)
		assert.Contains(t, string(content), originalContent)
		// Header should come before original content
		headerIdx := lo.IndexOf([]byte(string(content)), []byte(header)[0])
		contentIdx := lo.IndexOf([]byte(string(content)), []byte(originalContent)[0])
		assert.Less(t, headerIdx, contentIdx)
	})

	t.Run("add multi-line header", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.txt")
		_ = afero.WriteFile(AppFs, filePath, []byte("Original content"), 0644)

		header := "Copyright (c) 2023 Test Corp\nSPDX-License-Identifier: MPL-2.0"
		err := AddHeader(filePath, header)
		assert.Nil(t, err)

		content, _ := afero.ReadFile(AppFs, filePath)
		assert.Contains(t, string(content), "Copyright (c) 2023 Test Corp")
		assert.Contains(t, string(content), "SPDX-License-Identifier: MPL-2.0")
		assert.Contains(t, string(content), "Original content")
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
		fileExists, _ := afero.Exists(AppFs, licensePath)
		assert.True(t, fileExists)

		// Verify content contains MPL-2.0 license text
		content, _ := afero.ReadFile(AppFs, licensePath)
		assert.Contains(t, string(content), "Mozilla Public License")
	})

	t.Run("create LICENSE file with Apache-2.0", func(t *testing.T) {
		tempDir := t.TempDir()
		licensePath, err := AddLicenseFile(tempDir, "Apache-2.0")
		assert.Nil(t, err)

		content, _ := afero.ReadFile(AppFs, licensePath)
		assert.Contains(t, string(content), "Apache License")
	})

	t.Run("create LICENSE file with MIT", func(t *testing.T) {
		tempDir := t.TempDir()
		licensePath, err := AddLicenseFile(tempDir, "MIT")
		assert.Nil(t, err)

		content, _ := afero.ReadFile(AppFs, licensePath)
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
		_ = AppFs.MkdirAll(subDir, 0755)
		_ = afero.WriteFile(AppFs, filepath.Join(subDir, "LICENSE"), []byte("sublicense"), 0644)

		// FindLicenseFiles should only find files in the top-level directory, not subdirs
		result, err := FindLicenseFiles(tempDir)
		assert.Nil(t, err)
		assert.Equal(t, []string{}, result)
	})
}
