// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package licensecheck

import (
	"bytes"
	"os"
)

// HasCopyright reports whether or not a file contains a copyright statement
// It makes no promises as to the validity of the copyright statement, however!
// If you wish to validate the contents of the statement, use hasValidCopyright
func HasCopyright(filePath string) (bool, error) {
	// just check if the word "copyright" exists in the header
	// TODO: maybe further check the formation of the copyright statement, such as
	// ensuring a holder exists, etc.
	return HasMatchingCopyright(filePath, "copyright", false)
}

// HasMatchingCopyright takes an explicit copyright statement and validates that
// a given file contains that string in the header (first 1k chars)
func HasMatchingCopyright(filePath string, copyrightStatement string, caseSensitive bool) (bool, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	// Check the first 300 characters
	n := 300
	if len(b) < n {
		n = len(b)
	}

	expected := []byte(copyrightStatement)
	header := b[:n]
	if !caseSensitive {
		header = bytes.ToLower(header)
		expected = bytes.ToLower(expected)
	}
	return bytes.Contains(header, expected), nil
}
