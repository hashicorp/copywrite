package licensecheck

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestHasMatchingCopyright(t *testing.T) {
	AppFs := afero.NewOsFs()
	tempDir := t.TempDir()

	desiredCopyrightString := "Copyright (c) 2022 HashiCorp, Inc."

	cases := []struct {
		description   string
		fileContents  string
		caseSensitive bool
		expectedValid bool
		expectedError error
	}{
		{
			description:   "Missing copyright statement should fail",
			fileContents:  "",
			caseSensitive: false,
			expectedValid: false,
			expectedError: nil,
		},
		{
			description:   "Valid copyright statement should pass",
			fileContents:  "Copyright (c) 2022 HashiCorp, Inc.",
			caseSensitive: false,
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Valid copyright statement with language headers should pass",
			fileContents:  "#!/bin/bash\nCopyright (c) 2022 HashiCorp, Inc.",
			caseSensitive: false,
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Malformed copyright statement without symbol should fail",
			fileContents:  "Copyright 2022 HashiCorp, Inc.",
			caseSensitive: false,
			expectedValid: false,
			expectedError: nil,
		},
		{
			description:   "Malformed copyright statement without year should fail",
			fileContents:  "Copyright (c) HashiCorp, Inc.",
			caseSensitive: false,
			expectedValid: false,
			expectedError: nil,
		},
		{
			description:   "Malformed copyright statement without holder should fail",
			fileContents:  "Copyright (c) 2022",
			caseSensitive: false,
			expectedValid: false,
			expectedError: nil,
		},
		{
			description:   "Malformed copyright statement with year range should fail",
			fileContents:  "Copyright 1995-2022 HashiCorp, Inc.",
			caseSensitive: false,
			expectedValid: false,
			expectedError: nil,
		},
		{
			description:   "Valid lowercase copyright statement should pass",
			fileContents:  "copyright (c) 2022 hashicorp, inc.",
			caseSensitive: false,
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "valid uppercase copyright statement should pass",
			fileContents:  "COPYRIGHT (C) 2022 HASHICORP, INC.",
			caseSensitive: false,
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Valid lowercase copyright statement with case sensitivity on should fail",
			fileContents:  "copyright (c) 2022 hashicorp, inc.",
			caseSensitive: true,
			expectedValid: false,
			expectedError: nil,
		},
		{
			description:   "valid uppercase copyright statement with case sensitivity on should fail",
			fileContents:  "COPYRIGHT (C) 2022 HASHICORP, INC.",
			caseSensitive: true,
			expectedValid: false,
			expectedError: nil,
		},
		{
			description: "valid copyright statement on document that has the copyright word elsewhere should pass",
			fileContents: `Copyright (c) 2022 HashiCorp, Inc.

			Apache License
			Version 2.0, January 2004
			http://www.apache.org/licenses/

			TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION

			1. Definitions.

			"License" shall mean the terms and conditions for use, reproduction,
			and distribution as defined by Sections 1 through 9 of this document.

			"Licensor" shall mean the copyright owner or entity authorized by
			the copyright owner that is granting the License.`,
			caseSensitive: false,
			expectedValid: true,
			expectedError: nil,
		},
		{
			description: "missing statement on document that has the copyright word elsewhere should fail",
			fileContents: `Apache License
			Version 2.0, January 2004
			http://www.apache.org/licenses/

			TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION

			1. Definitions.

			"License" shall mean the terms and conditions for use, reproduction,
			and distribution as defined by Sections 1 through 9 of this document.

			"Licensor" shall mean the copyright owner or entity authorized by
			the copyright owner that is granting the License.`,
			caseSensitive: false,
			expectedValid: false,
			expectedError: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			f, _ := afero.TempFile(AppFs, tempDir, "")
			_ = afero.WriteFile(AppFs, f.Name(), []byte(tt.fileContents), 0644)
			// run test
			actualValid, err := HasMatchingCopyright(f.Name(), desiredCopyrightString, tt.caseSensitive)
			assert.ErrorIs(t, err, tt.expectedError, tt.description)
			assert.Equal(t, tt.expectedValid, actualValid, tt.description)
		})
	}
}

// hasCopyright only tests if a copyright statement exists, not if the contents are valid
func TestHasCopyright(t *testing.T) {
	AppFs := afero.NewOsFs()
	tempDir := t.TempDir()

	cases := []struct {
		description   string
		fileContents  string
		expectedValid bool
		expectedError error
	}{
		{
			description:   "Missing copyright statement should fail",
			fileContents:  "",
			expectedValid: false,
			expectedError: nil,
		},
		{
			description:   "Missing copyright statement with language headers should fail",
			fileContents:  "#!/bin/bash",
			expectedValid: false,
			expectedError: nil,
		},
		{
			description:   "Valid copyright statement should pass",
			fileContents:  "Copyright (c) 2022 HashiCorp, Inc.",
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Valid copyright statement with language headers should pass",
			fileContents:  "#!/bin/bash\nCopyright (c) 2022 HashiCorp, Inc.",
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Malformed copyright statement without symbol should pass",
			fileContents:  "Copyright 2022 HashiCorp, Inc.",
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Malformed copyright statement without year should pass",
			fileContents:  "Copyright (c) HashiCorp, Inc.",
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Malformed copyright statement without holder should pass",
			fileContents:  "Copyright (c) 2022",
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Malformed copyright statement with year range should pass",
			fileContents:  "Copyright 1995-2022 HashiCorp, Inc.",
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Valid lowercase copyright statement should pass",
			fileContents:  "copyright 2022 hashicorp, inc.",
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "valid uppercase copyright statement should pass",
			fileContents:  "COPYRIGHT 2022 HASHICORP, INC.",
			expectedValid: true,
			expectedError: nil,
		},
		{
			description: "valid copyright statement on document that has the copyright word elsewhere should pass",
			fileContents: `Copyright (c) 2022 HashiCorp, Inc.

			Apache License
			Version 2.0, January 2004
			http://www.apache.org/licenses/

			TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION

			1. Definitions.

			"License" shall mean the terms and conditions for use, reproduction,
			and distribution as defined by Sections 1 through 9 of this document.

			"Licensor" shall mean the copyright owner or entity authorized by
			the copyright owner that is granting the License.`,
			expectedValid: true,
			expectedError: nil,
		},
		{
			description: "missing statement on document that has the copyright word elsewhere should fail",
			fileContents: `Apache License
			Version 2.0, January 2004
			http://www.apache.org/licenses/

			TERMS AND CONDITIONS FOR USE, REPRODUCTION, AND DISTRIBUTION

			1. Definitions.

			"License" shall mean the terms and conditions for use, reproduction,
			and distribution as defined by Sections 1 through 9 of this document.

			"Licensor" shall mean the copyright owner or entity authorized by
			the copyright owner that is granting the License.`,
			expectedValid: false,
			expectedError: nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			f, _ := afero.TempFile(AppFs, tempDir, "")
			_ = afero.WriteFile(AppFs, f.Name(), []byte(tt.fileContents), 0644)
			// run test
			actualValid, err := HasCopyright(f.Name())
			assert.ErrorIs(t, err, tt.expectedError, tt.description)
			assert.Equal(t, tt.expectedValid, actualValid, tt.description)
		})
	}
}

func TestHasMatchingCopyright_ErrorHandling(t *testing.T) {
	t.Run("error on non-existent file", func(t *testing.T) {
		hasCopyright, err := HasMatchingCopyright("/nonexistent/file.txt", "Copyright", false)
		assert.NotNil(t, err)
		assert.False(t, hasCopyright)
	})
}

func TestHasMatchingCopyright_EdgeCases(t *testing.T) {
	AppFs := afero.NewOsFs()
	tempDir := t.TempDir()

	t.Run("file exactly 300 bytes with copyright at end", func(t *testing.T) {
		// Create a file exactly 300 bytes where "Copyright" appears at byte 290
		padding := make([]byte, 290)
		for i := range padding {
			padding[i] = 'A'
		}
		content := string(padding) + "Copyright!"

		f, _ := afero.TempFile(AppFs, tempDir, "")
		_ = afero.WriteFile(AppFs, f.Name(), []byte(content), 0644)

		hasCopyright, err := HasMatchingCopyright(f.Name(), "Copyright", false)
		assert.Nil(t, err)
		assert.True(t, hasCopyright)
	})

	t.Run("file less than 300 bytes", func(t *testing.T) {
		content := "Short file with Copyright notice"

		f, _ := afero.TempFile(AppFs, tempDir, "")
		_ = afero.WriteFile(AppFs, f.Name(), []byte(content), 0644)

		hasCopyright, err := HasMatchingCopyright(f.Name(), "Copyright", false)
		assert.Nil(t, err)
		assert.True(t, hasCopyright)
	})

	t.Run("file larger than 300 bytes with copyright after header", func(t *testing.T) {
		// Create content > 300 bytes with copyright appearing after byte 300
		header := make([]byte, 350)
		for i := range header {
			header[i] = 'X'
		}
		content := string(header) + "\nCopyright notice here"

		f, _ := afero.TempFile(AppFs, tempDir, "")
		_ = afero.WriteFile(AppFs, f.Name(), []byte(content), 0644)

		// Should not find copyright since it's after the 300-byte header check
		hasCopyright, err := HasMatchingCopyright(f.Name(), "Copyright", false)
		assert.Nil(t, err)
		assert.False(t, hasCopyright)
	})

	t.Run("empty search string", func(t *testing.T) {
		f, _ := afero.TempFile(AppFs, tempDir, "")
		_ = afero.WriteFile(AppFs, f.Name(), []byte("Some content"), 0644)

		// Empty string should always be found
		hasCopyright, err := HasMatchingCopyright(f.Name(), "", false)
		assert.Nil(t, err)
		assert.True(t, hasCopyright)
	})

	t.Run("search string longer than file", func(t *testing.T) {
		f, _ := afero.TempFile(AppFs, tempDir, "")
		_ = afero.WriteFile(AppFs, f.Name(), []byte("Short"), 0644)

		hasCopyright, err := HasMatchingCopyright(f.Name(), "This is a very long copyright statement that is longer than the file content", false)
		assert.Nil(t, err)
		assert.False(t, hasCopyright)
	})
}

func TestHasCopyright_ErrorHandling(t *testing.T) {
	t.Run("error on non-existent file", func(t *testing.T) {
		hasCopyright, err := HasCopyright("/nonexistent/file.txt")
		assert.NotNil(t, err)
		assert.False(t, hasCopyright)
	})
}
