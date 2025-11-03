// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package licensecheck

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestHasMatchingCopyright(t *testing.T) {
	AppFs := afero.NewOsFs()
	tempDir := t.TempDir()


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
			caseSensitive: false,
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Valid copyright statement with language headers should pass",
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
			caseSensitive: false,
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "valid uppercase copyright statement should pass",
			caseSensitive: false,
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Valid lowercase copyright statement with case sensitivity on should fail",
			caseSensitive: true,
			expectedValid: false,
			expectedError: nil,
		},
		{
			description:   "valid uppercase copyright statement with case sensitivity on should fail",
			caseSensitive: true,
			expectedValid: false,
			expectedError: nil,
		},
		{
			description: "valid copyright statement on document that has the copyright word elsewhere should pass",

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
			expectedValid: true,
			expectedError: nil,
		},
		{
			description:   "Valid copyright statement with language headers should pass",
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
