// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package licensecheck

import (
	"os"
	"path/filepath"
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
			result := parseCopyrightLine(tt.line, tt.lineNum)

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

	copyrights, err := ExtractAllCopyrightInfo(testFile)
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

	info, err := ExtractCopyrightInfo(testFile)
	require.NoError(t, err)
	require.NotNil(t, info)

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

	info, err := ExtractCopyrightInfo(testFile)
	require.NoError(t, err)
	assert.Nil(t, info)
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
			expectedContent: `// Copyright IBM Corp. 2020, ` + string(rune(currentYear/1000+48)) + string(rune((currentYear/100)%10+48)) + string(rune((currentYear/10)%10+48)) + string(rune(currentYear%10+48)) + `
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
