// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package licensecheck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samber/lo"
)

// EnsureCorrectName fixes a malformed license file name and returns the
// new (corrected) file path
// E.g., "license.txt" --> "LICENSE"
func EnsureCorrectName(filePath string) (string, error) {
	dir, _ := filepath.Split(filePath)
	desiredPath := filepath.Join(dir, "LICENSE")
	if desiredPath != filePath {
		fmt.Printf("Found improperly named file \"%s\". Renaming to \"%s\"", filePath, desiredPath)
		err := os.Rename(filePath, filepath.Join(dir, "LICENSE"))
		if err != nil {
			return "", fmt.Errorf("Unable to rename file \"%s\". Full error context: %s", filePath, err)
		}
	} else {
		fmt.Printf("Validated file: %s\n", filePath)
	}

	return desiredPath, nil
}

// AddHeader prepends a given string to a file. It will automatically handle
// newline characters
func AddHeader(filePath string, header string) error {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	b = append([]byte(header+"\n\n"), b...)
	return os.WriteFile(filePath, b, 0644)
}

// AddLicenseFile creates a file named "LICENSE" in the target directory
// pre-populated with license text based on the SPDX Identifier you supply.
// Returns the fully qualified path to the license file it created
//
// NOTE: this function will NOT add a copyright statement for you. You must
// manually call AddHeader() afterward if you wish to have copyright headers
func AddLicenseFile(dirPath string, spdxID string) (string, error) {
	template, exists := licenseTemplate[spdxID]
	if !exists {
		validOptions := strings.Join(lo.Keys(licenseTemplate), ", ")
		return "", fmt.Errorf("Failed to add license file, unknown SPDX license ID: %s. The following options are supported at this time: %s", spdxID, validOptions)
	}

	destinationPath, err := filepath.Abs(filepath.Join(dirPath, "LICENSE"))
	if err != nil {
		return "", err
	}

	err = os.WriteFile(destinationPath, []byte(template), 0644)
	if err != nil {
		return "", err
	}
	return destinationPath, nil
}

// FindLicenseFiles returns a list of filepaths for licenses in a given directory
func FindLicenseFiles(dirPath string) ([]string, error) {
	// find all files in the supplied dirPath (1-level deep only)
	files, err := filepath.Glob(fmt.Sprintf("%s/*", dirPath))

	if err != nil {
		return []string{}, err
	}

	// filter without case sensitivity for LICENSE, LICENSE.txt, and LICENSE.md
	r := regexp.MustCompile(`^(?i)(license.md|license.txt|license)$`)

	matches := lo.Filter(files, func(f string, _ int) bool {
		_, file := filepath.Split(f)
		return r.MatchString(file)
	})

	return matches, nil
}
