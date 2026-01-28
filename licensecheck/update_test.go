// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package licensecheck

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCopyrightLine(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		lineNum      int
		expectedInfo *CopyrightInfo
		expectNil    bool
	}{
		{
			name:    "Simple copyright with single year",
			line:    "// Copyright IBM Corp. 2023",
			lineNum: 1,
			expectedInfo: &CopyrightInfo{
				LineNumber:   1,
				OriginalLine: "// Copyright IBM Corp. 2023",
				Holder:       "IBM Corp.",
				StartYear:    2023,
				EndYear:      2023,
				Prefix:       "// ",
				TrailingText: "",
			},
		},
		{
			name:    "Copyright with year range",
			line:    "// Copyright IBM Corp. 2022, 2025",
			lineNum: 1,
			expectedInfo: &CopyrightInfo{
				LineNumber:   1,
				OriginalLine: "// Copyright IBM Corp. 2022, 2025",
				Holder:       "IBM Corp.",
				StartYear:    2022,
				EndYear:      2025,
				Prefix:       "// ",
				TrailingText: "",
			},
		},
		{
			name:    "Copyright with (c) symbol",
			line:    "# Copyright (c) HashiCorp, Inc. 2020",
			lineNum: 2,
			expectedInfo: &CopyrightInfo{
				LineNumber:   2,
				OriginalLine: "# Copyright (c) HashiCorp, Inc. 2020",
				Holder:       "HashiCorp, Inc.",
				StartYear:    2020,
				EndYear:      2020,
				Prefix:       "# ",
				TrailingText: "",
			},
		},
		{
			name:    "Copyright with trailing text",
			line:    "/* Copyright IBM Corp. 2023 - All rights reserved */",
			lineNum: 1,
			expectedInfo: &CopyrightInfo{
				LineNumber:   1,
				OriginalLine: "/* Copyright IBM Corp. 2023 - All rights reserved */",
				Holder:       "IBM Corp.",
				StartYear:    2023,
				EndYear:      2023,
				Prefix:       "/* ",
				TrailingText: " - All rights reserved */",
			},
		},
		{
			name:      "Line without copyright",
			line:      "// This is just a comment",
			lineNum:   1,
			expectNil: true,
		},
		{
			name:    "Copyright without year (holder only)",
			line:    "// Copyright IBM Corp.",
			lineNum: 1,
			expectedInfo: &CopyrightInfo{
				LineNumber:   1,
				OriginalLine: "// Copyright IBM Corp.",
				Holder:       "IBM Corp.",
				StartYear:    0,
				EndYear:      0,
				Prefix:       "// ",
				TrailingText: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCopyrightLine(tt.line, tt.lineNum, "file.go")

			if tt.expectNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedInfo.LineNumber, result.LineNumber)
			assert.Equal(t, tt.expectedInfo.OriginalLine, result.OriginalLine)
			assert.Equal(t, tt.expectedInfo.Holder, result.Holder)
			assert.Equal(t, tt.expectedInfo.StartYear, result.StartYear)
			assert.Equal(t, tt.expectedInfo.EndYear, result.EndYear)
			assert.Equal(t, tt.expectedInfo.Prefix, result.Prefix)
			assert.Equal(t, tt.expectedInfo.TrailingText, result.TrailingText)
		})
	}
}

func TestExtractCommentPrefix(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		expectedPrefix string
	}{
		{
			name:           "Double slash comment",
			line:           "// Copyright IBM Corp.",
			expectedPrefix: "// ",
		},
		{
			name:           "Double slash without space",
			line:           "//Copyright IBM Corp.",
			expectedPrefix: "//",
		},
		{
			name:           "Hash comment",
			line:           "# Copyright IBM Corp.",
			expectedPrefix: "# ",
		},
		{
			name:           "Star comment",
			line:           "* Copyright IBM Corp.",
			expectedPrefix: "* ",
		},
		{
			name:           "Block comment start",
			line:           "/* Copyright IBM Corp.",
			expectedPrefix: "/* ",
		},
		{
			name:           "Indented comment",
			line:           "  // Copyright IBM Corp.",
			expectedPrefix: "  // ",
		},
		{
			name:           "Tab indented comment",
			line:           "\t# Copyright IBM Corp.",
			expectedPrefix: "\t# ",
		},
		{
			name:           "No comment prefix",
			line:           "Copyright IBM Corp.",
			expectedPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCommentPrefix(tt.line)
			assert.Equal(t, tt.expectedPrefix, result)
		})
	}
}

func TestParseYearFromGitOutput(t *testing.T) {
	tests := []struct {
		name         string
		output       []byte
		useFirstLine bool
		expectedYear int
		expectError  bool
	}{
		{
			name:         "Single year - first line",
			output:       []byte("2023\n"),
			useFirstLine: true,
			expectedYear: 2023,
		},
		{
			name:         "Multiple years - first line",
			output:       []byte("2020\n2021\n2022\n2023\n"),
			useFirstLine: true,
			expectedYear: 2020,
		},
		{
			name:         "Empty output",
			output:       []byte(""),
			useFirstLine: true,
			expectError:  true,
		},
		{
			name:         "Invalid year",
			output:       []byte("invalid\n"),
			useFirstLine: true,
			expectError:  true,
		},
		{
			name:         "Whitespace handling",
			output:       []byte("  2022  \n"),
			useFirstLine: true,
			expectedYear: 2022,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			year, err := parseYearFromGitOutput(tt.output, tt.useFirstLine)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedYear, year)
		})
	}
}

func TestExtractAllCopyrightInfo(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	fileContent := `// Copyright IBM Corp. 2020, 2023

package main

// Some other comment
// Copyright HashiCorp, Inc. 2019

func main() {
	// Not a copyright
}
`

	err := os.WriteFile(testFile, []byte(fileContent), 0644)
	require.NoError(t, err)

	copyrights, err := extractAllCopyrightInfo(testFile)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(copyrights), 2, "Should find at least 2 copyright statements")

	// First copyright
	assert.Equal(t, 1, copyrights[0].LineNumber)
	assert.Equal(t, "IBM Corp.", copyrights[0].Holder)
	assert.Equal(t, 2020, copyrights[0].StartYear)
	assert.Equal(t, 2023, copyrights[0].EndYear)

	// Find the HashiCorp copyright (should be second or later)
	var hashicorpFound bool
	for _, c := range copyrights {
		if c.Holder == "HashiCorp, Inc." {
			assert.Equal(t, 2019, c.StartYear)
			assert.Equal(t, 2019, c.EndYear)
			hashicorpFound = true
			break
		}
	}
	assert.True(t, hashicorpFound, "Should find HashiCorp copyright")
}

func TestExtractCopyrightInfo(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	fileContent := `// Copyright IBM Corp. 2020, 2023
// SPDX-License-Identifier: MPL-2.0

package main
`

	err := os.WriteFile(testFile, []byte(fileContent), 0644)
	require.NoError(t, err)

	copyrights, err := extractAllCopyrightInfo(testFile)
	require.NoError(t, err)
	require.NotEmpty(t, copyrights)

	info := copyrights[0]
	assert.Equal(t, 1, info.LineNumber)
	assert.Equal(t, "IBM Corp.", info.Holder)
	assert.Equal(t, 2020, info.StartYear)
	assert.Equal(t, 2023, info.EndYear)
}

func TestExtractCopyrightInfo_NoCopyright(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	fileContent := `// Just a regular comment
package main
`

	err := os.WriteFile(testFile, []byte(fileContent), 0644)
	require.NoError(t, err)

	copyrights, err := extractAllCopyrightInfo(testFile)
	require.NoError(t, err)
	assert.Empty(t, copyrights)
}

func TestUpdateCopyrightHeader(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name             string
		initialContent   string
		targetHolder     string
		configYear       int
		forceCurrentYear bool
		expectModified   bool
		expectedContent  string
	}{
		{
			name: "Update end year when outdated",
			initialContent: `// Copyright IBM Corp. 2022, 2023
package main
`,
			targetHolder:     "IBM Corp.",
			configYear:       2022,
			forceCurrentYear: true,
			expectModified:   true,
			expectedContent: `// Copyright IBM Corp. 2022, ` + string(rune(currentYear/1000+48)) + string(rune((currentYear/100)%10+48)) + string(rune((currentYear/10)%10+48)) + string(rune(currentYear%10+48)) + `
package main
`,
		},
		{
			name: "Update start year when different from config",
			initialContent: `// Copyright IBM Corp. 2023
package main
`,
			targetHolder:   "IBM Corp.",
			configYear:     2020,
			expectModified: true,
			// Since we don't have git history in this test and forceCurrentYear is false,
			// the end year should NOT update, only the start year.
			expectedContent: `// Copyright IBM Corp. 2020, 2023
package main
`,
		},
		{
			name: "No update needed",
			initialContent: `// Copyright IBM Corp. ` + string(rune(currentYear/1000+48)) + string(rune((currentYear/100)%10+48)) + string(rune((currentYear/10)%10+48)) + string(rune(currentYear%10+48)) + `
package main
`,
			targetHolder:   "IBM Corp.",
			configYear:     currentYear,
			expectModified: false,
		},
		{
			name: "Wrong holder - no update",
			initialContent: `// Copyright HashiCorp, Inc. 2020
package main
`,
			targetHolder:   "IBM Corp.",
			configYear:     2022,
			expectModified: false,
		},
		{
			name: "No copyright - no update",
			initialContent: `package main
`,
			targetHolder:   "IBM Corp.",
			configYear:     2022,
			expectModified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.go")

			err := os.WriteFile(testFile, []byte(tt.initialContent), 0644)
			require.NoError(t, err)

			modified, err := UpdateCopyrightHeader(testFile, tt.targetHolder, tt.configYear, tt.forceCurrentYear)
			require.NoError(t, err)
			assert.Equal(t, tt.expectModified, modified)

			if tt.expectModified && tt.expectedContent != "" {
				content, err := os.ReadFile(testFile)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedContent, string(content))
			}
		})
	}
}

func TestParseCopyrightLine_UnprefixedLicense(t *testing.T) {
	line := "Copyright IBM Corp. 2018, 2025"
	info := parseCopyrightLine(line, 1, "LICENSE")
	require.NotNil(t, info)
	assert.Equal(t, "IBM Corp.", info.Holder)
	assert.Equal(t, 2018, info.StartYear)
	assert.Equal(t, 2025, info.EndYear)
	assert.Equal(t, "", info.Prefix)
}

func TestNeedsUpdate(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name              string
		fileContent       string
		targetHolder      string
		configYear        int
		forceCurrentYear  bool
		expectNeedsUpdate bool
	}{
		{
			name: "Needs update - outdated end year",
			fileContent: `// Copyright IBM Corp. 2022, 2023
package main
`,
			targetHolder:      "IBM Corp.",
			configYear:        2022,
			forceCurrentYear:  true,
			expectNeedsUpdate: true,
		},
		{
			name: "Needs update - different start year",
			fileContent: `// Copyright IBM Corp. 2023
package main
`,
			targetHolder:      "IBM Corp.",
			configYear:        2020,
			expectNeedsUpdate: true,
		},
		{
			name: "No update needed - current",
			fileContent: `// Copyright IBM Corp. ` + string(rune(currentYear/1000+48)) + string(rune((currentYear/100)%10+48)) + string(rune((currentYear/10)%10+48)) + string(rune(currentYear%10+48)) + `
package main
`,
			targetHolder:      "IBM Corp.",
			configYear:        currentYear,
			expectNeedsUpdate: false,
		},
		{
			name: "Wrong holder - no update",
			fileContent: `// Copyright HashiCorp, Inc. 2020
package main
`,
			targetHolder:      "IBM Corp.",
			configYear:        2022,
			expectNeedsUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.go")

			err := os.WriteFile(testFile, []byte(tt.fileContent), 0644)
			require.NoError(t, err)

			needsUpdate, err := NeedsUpdate(testFile, tt.targetHolder, tt.configYear, tt.forceCurrentYear)
			require.NoError(t, err)
			assert.Equal(t, tt.expectNeedsUpdate, needsUpdate)
		})
	}
}

func TestUpdateCopyrightHeader_SkipCopywriteConfig(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, ".copywrite.hcl")

	fileContent := `// Copyright IBM Corp. 2020
schema_version = 1
`

	err := os.WriteFile(testFile, []byte(fileContent), 0644)
	require.NoError(t, err)

	modified, err := UpdateCopyrightHeader(testFile, "IBM Corp.", 2022, false)
	require.NoError(t, err)
	assert.False(t, modified, "Should skip .copywrite.hcl file")
}

func TestNeedsUpdate_SkipCopywriteConfig(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, ".copywrite.hcl")

	fileContent := `// Copyright IBM Corp. 2020
schema_version = 1
`

	err := os.WriteFile(testFile, []byte(fileContent), 0644)
	require.NoError(t, err)

	needsUpdate, err := NeedsUpdate(testFile, "IBM Corp.", 2022, false)
	require.NoError(t, err)
	assert.False(t, needsUpdate, "Should skip .copywrite.hcl file")
}

func TestIsGenerated(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		expectSkip bool
	}{
		{
			name: "Go generated file",
			content: `// Code generated by protoc-gen-go. DO NOT EDIT.
package main

func main() {}
`,
			expectSkip: true,
		},
		{
			name: "Cargo raze generated file",
			content: `DO NOT EDIT! Replaced on runs of cargo-raze

[package]
name = "test"
`,
			expectSkip: true,
		},
		{
			name: "Terraform init generated file",
			content: `# This file is maintained automatically by "terraform init".

provider "aws" {}
`,
			expectSkip: true,
		},
		{
			name: "Regular file",
			content: `// Copyright IBM Corp. 2023
package main

func main() {}
`,
			expectSkip: false,
		},
		{
			name: "File with 'generated' in comment but not a marker",
			content: `// This file was generated by hand
package main
`,
			expectSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGenerated([]byte(tt.content))
			assert.Equal(t, tt.expectSkip, result)
		})
	}
}

func TestHasSpecialFirstLine(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		filePath string
		expected bool
	}{
		{
			name:     "Shell script with shebang",
			content:  "#!/bin/bash\necho 'hello'\n",
			filePath: "test.sh",
			expected: true,
		},
		{
			name:     "XML with declaration",
			content:  "<?xml version=\"1.0\"?>\n<root></root>\n",
			filePath: "test.xml",
			expected: true,
		},
		{
			name:     "PHP file",
			content:  "<?php\necho 'hello';\n",
			filePath: "test.php",
			expected: true,
		},
		{
			name:     "Ruby with encoding",
			content:  "# encoding: utf-8\nputs 'hello'\n",
			filePath: "test.rb",
			expected: true,
		},
		{
			name:     "Dockerfile with directive",
			content:  "# syntax=docker/dockerfile:1\nFROM ubuntu\n",
			filePath: "Dockerfile",
			expected: true,
		},
		{
			name:     "Regular Go file",
			content:  "package main\n\nfunc main() {}\n",
			filePath: "test.go",
			expected: false,
		},
		{
			name:     "File with copyright header",
			content:  "// Copyright IBM Corp. 2023\npackage main\n",
			filePath: "test.go",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasSpecialFirstLine([]byte(tt.content), tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateCopyrightHeader_SkipsGeneratedFiles(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "generated.go")

	fileContent := `// Code generated by protoc-gen-go. DO NOT EDIT.
// Copyright IBM Corp. 2020

package main

func main() {}
`

	err := os.WriteFile(testFile, []byte(fileContent), 0644)
	require.NoError(t, err)

	modified, err := UpdateCopyrightHeader(testFile, "IBM Corp.", 2023, false)
	require.NoError(t, err)
	assert.False(t, modified, "Should skip generated files")

	// Verify file content wasn't modified
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, fileContent, string(content))
}

func TestNeedsUpdate_SkipsGeneratedFiles(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "generated.go")

	fileContent := `// Code generated by protoc-gen-go. DO NOT EDIT.
// Copyright IBM Corp. 2020

package main
`

	err := os.WriteFile(testFile, []byte(fileContent), 0644)
	require.NoError(t, err)

	needsUpdate, err := NeedsUpdate(testFile, "IBM Corp.", 2023, false)
	require.NoError(t, err)
	assert.False(t, needsUpdate, "Should skip generated files")
}

func TestParseCopyrightLine_StrictCopyrightCheck(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		expectNil bool
	}{
		{
			name:      "Valid: Starts with Copyright",
			line:      "// Copyright IBM Corp. 2020",
			expectNil: false,
		},
		{
			name:      "Invalid: Copyright mentioned but not at start",
			line:      "// This file has a different copyright holder - IBM Corp.",
			expectNil: true,
		},
		{
			name:      "Invalid: Copyright in middle of sentence",
			line:      "// See copyright notice in LICENSE file",
			expectNil: true,
		},
		{
			name:      "Valid: Copyright with leading whitespace after comment",
			line:      "//   Copyright IBM Corp. 2020",
			expectNil: false,
		},
		{
			name:      "Invalid: No Copyright word at all",
			line:      "// IBM Corp. 2020",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCopyrightLine(tt.line, 1, "file.go")
			if tt.expectNil {
				assert.Nil(t, result, "Should return nil for non-copyright lines")
			} else {
				assert.NotNil(t, result, "Should parse valid copyright lines")
			}
		})
	}
}

func TestExtractCommentPrefix_AllFormats(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		expectedPrefix string
	}{
		{"C++ style", "// Copyright", "// "},
		{"Shell/Python style", "# Copyright", "# "},
		{"Block comment start", "/* Copyright", "/* "},
		{"XML/HTML comment", "<!-- Copyright", "<!-- "},
		{"Lisp style", ";; Copyright", ";; "},
		{"Erlang style", "% Copyright", "% "},
		{"Haskell/SQL style", "-- Copyright", "-- "},
		{"Handlebars style", "{{! Copyright", "{{! "},
		{"OCaml style", "(** Copyright", "(** "},
		{"EJS template style", "<%/* Copyright", "<%/* "},
		{"JSDoc style", "/** Copyright", "/** "},
		{"Block continuation", "* Copyright", "* "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCommentPrefix(tt.line)
			assert.Equal(t, tt.expectedPrefix, result)
		})
	}
}

func TestParseCopyrightLine_InlineComment(t *testing.T) {
	line := "var x := 1 // Copyright IBM Corp. 2023"
	info := parseCopyrightLine(line, 1, "file.go")
	require.NotNil(t, info)
	assert.Equal(t, "IBM Corp.", info.Holder)
	assert.Equal(t, 2023, info.StartYear)
	assert.Equal(t, 2023, info.EndYear)
	assert.Greater(t, info.PrefixIndex, 0)
	assert.Contains(t, info.Prefix, "//")
}

func TestUpdateCopyrightHeader_InlineCommentPreserved(t *testing.T) {
	currentYear := time.Now().Year()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "inline.go")

	// code before inline comment should be preserved
	initial := "var x := 1 // Copyright IBM Corp. 2022, 2023\n"
	err := os.WriteFile(testFile, []byte(initial), 0644)
	require.NoError(t, err)

	modified, err := UpdateCopyrightHeader(testFile, "IBM Corp.", 2022, true)
	require.NoError(t, err)
	assert.True(t, modified)

	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	expected := "var x := 1 // Copyright IBM Corp. 2022, " + strconv.Itoa(currentYear) + "\n"
	assert.Equal(t, expected, string(content))
}

func TestUpdateCopyrightHeader_WrongHolder(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	fileContent := `// Copyright HashiCorp, Inc. 2020
// SPDX-License-Identifier: MPL-2.0

package main
`

	err := os.WriteFile(testFile, []byte(fileContent), 0644)
	require.NoError(t, err)

	modified, err := UpdateCopyrightHeader(testFile, "IBM Corp.", 2023, false)
	require.NoError(t, err)
	assert.False(t, modified, "Should not update different copyright holder")

	// Verify file wasn't modified
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, fileContent, string(content))
}

func TestCalculateYearUpdates(t *testing.T) {
	currentYear := time.Now().Year()

	t.Run("Update start year when canonical differs", func(t *testing.T) {
		info := &CopyrightInfo{StartYear: 2023, EndYear: 2023}
		shouldUpdate, newStart, newEnd := calculateYearUpdates(
			info, 2020, 2023, currentYear, false,
		)
		assert.True(t, shouldUpdate)
		assert.Equal(t, 2020, newStart)
		assert.Equal(t, 2023, newEnd)
	})

	t.Run("No update when already current", func(t *testing.T) {
		info := &CopyrightInfo{StartYear: 2020, EndYear: currentYear}
		shouldUpdate, _, _ := calculateYearUpdates(
			info, 2020, currentYear, currentYear, false,
		)
		assert.False(t, shouldUpdate)
	})

	t.Run("Force current year updates end year", func(t *testing.T) {
		info := &CopyrightInfo{StartYear: 2020, EndYear: currentYear - 1}
		shouldUpdate, newStart, newEnd := calculateYearUpdates(
			info, 2020, currentYear-1, currentYear, true,
		)
		assert.True(t, shouldUpdate)
		assert.Equal(t, 2020, newStart)
		assert.Equal(t, currentYear, newEnd)
	})

	t.Run("No years uses config and force updates end", func(t *testing.T) {
		info := &CopyrightInfo{StartYear: 0, EndYear: 0}
		shouldUpdate, newStart, newEnd := calculateYearUpdates(
			info, 2022, 0, currentYear, true,
		)
		assert.True(t, shouldUpdate)
		assert.Equal(t, 2022, newStart)
		assert.Equal(t, currentYear, newEnd)
	})
}
