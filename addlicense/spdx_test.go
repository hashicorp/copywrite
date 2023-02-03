// Copyright (c) HashiCorp, Inc.

package addlicense

import (
	"testing"
)

func TestValidSPDX(t *testing.T) {
	tests := []struct {
		description    string // test case description
		spdxID         string // SPDX ID passed to ValidSPDX()
		expectedOutput bool   // Whether the SPDX ID should be valid or not
	}{
		{
			"MPL-2.0 is a valid SPDX ID",
			"MPL-2.0",
			true,
		},
		{
			"MIT is a valid SPDX ID",
			"MIT",
			true,
		},
		{
			"Non-existent SPDX ID is invalid",
			"asdf323dd7g23f9h38rf978f3h938hf98asdf279hf85gh65323f", // Please don't make this a valid SPDX ID in the future <3
			false,
		},
		{
			"Empty SPDX ID is invalid",
			"",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			actualOutput := ValidSPDX(tt.spdxID)
			if tt.expectedOutput != actualOutput {
				t.Fatalf("ValidSPDX(%q) returned %v, want %v", tt.spdxID, actualOutput, tt.expectedOutput)
			}
		})
	}
}
