// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package addlicense

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// HeaderInfo represents parsed copyright header information
type HeaderInfo struct {
	FullMatch      string // The complete matched header text
	HasCopyright   bool   // Whether "Copyright" keyword exists
	HasCSymbol     bool   // Whether "(c)" symbol exists
	Year           string // Year or year range (e.g., "2020" or "2020,2025")
	Holder         string // Copyright holder name
	AdditionalText string // Any text after the holder (e.g., "All rights reserved")
	StartPos       int    // Start position in file
	EndPos         int    // End position in file
	Organization   string // Detected organization (IBM, Hashicorp, or Other)
}

// Regular expressions to match different copyright header formats
var (
	// Matches: Copyright (c) 2020 Hashicorp Inc.
	//          Copyright 2020 Hashicorp Inc.
	//          Copyright Hashicorp Inc.
	//          Copyright (c) Hashicorp Inc.
	//          Copyright (c) Hashicorp, Inc.
	//          Copyright IBM Corp. 2020,2025
	//          Copyright (c) IBM Corp. 2020,2025 All rights reserved
	copyrightRegex = regexp.MustCompile(
		`(?i)Copyright\s*(?:\(c\))?\s*(?:(?:IBM\s+Corp\.?|Hashicorp(?:(?:\s+|,\s*)Inc\.?)?)\s*)?` +
			`(?:(\d{4}(?:\s*[,-]\s*\d{4})?))?\s*` +
			`(?:(IBM\s+Corp\.?|Hashicorp(?:(?:\s+|,\s*)Inc\.?)?))?\s*` +
			`(?:(\d{4}(?:\s*[,-]\s*\d{4})?))?\s*` +
			`(.*)`,
	)

	// More specific patterns for better matching
	ibmHeaderRegex = regexp.MustCompile(
		`(?i)Copyright\s*(?:\(c\))?\s*IBM\s+Corp\.?\s*(?:(\d{4}(?:\s*[,-]\s*\d{4})?))?(.*)`,
	)

	hashicorpHeaderRegex = regexp.MustCompile(
		`(?i)Copyright\s*(?:\(c\))?\s*(?:(\d{4}(?:\s*[,-]\s*\d{4})?)\s*)?Hashicorp(?:(?:\s+|,\s*)Inc\.?)?(?:\s*(\d{4}(?:\s*[,-]\s*\d{4})?))?(.*)`,
	)

	otherOrgRegex = regexp.MustCompile(
		`(?i)Copyright\s*(?:\(c\))?\s*(?:\d{4}(?:\s*[,-]\s*\d{4})?\s*)?([A-Z][^\d\n]{2,})`,
	)
)

// ParseCopyrightHeader extracts copyright information from a header line
func ParseCopyrightHeader(line string) *HeaderInfo {
	line = strings.TrimSpace(line)

	// Check for other organizations first to avoid false positives
	if otherOrgRegex.MatchString(line) &&
		!strings.Contains(strings.ToLower(line), "ibm") &&
		!strings.Contains(strings.ToLower(line), "hashicorp") {
		matches := otherOrgRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			return &HeaderInfo{
				FullMatch:    line,
				HasCopyright: true,
				Organization: "Other",
			}
		}
	}

	// Try IBM format
	if matches := ibmHeaderRegex.FindStringSubmatch(line); matches != nil {
		info := &HeaderInfo{
			FullMatch:    matches[0],
			HasCopyright: true,
			HasCSymbol:   strings.Contains(line, "(c)"),
			Holder:       "IBM Corp.",
			Organization: "IBM",
		}
		if len(matches) > 1 && matches[1] != "" {
			info.Year = strings.ReplaceAll(matches[1], " ", "")
			info.Year = strings.ReplaceAll(info.Year, "-", ",")
		}
		if len(matches) > 2 && matches[2] != "" {
			info.AdditionalText = strings.TrimSpace(matches[2])
		}
		return info
	}

	// Try Hashicorp format
	if matches := hashicorpHeaderRegex.FindStringSubmatch(line); matches != nil {
		info := &HeaderInfo{
			FullMatch:    matches[0],
			HasCopyright: true,
			HasCSymbol:   strings.Contains(line, "(c)"),
			Holder:       "Hashicorp",
			Organization: "Hashicorp",
		}
		// Year can be in position 1 or 2
		if len(matches) > 1 && matches[1] != "" {
			info.Year = strings.ReplaceAll(matches[1], " ", "")
			info.Year = strings.ReplaceAll(info.Year, "-", ",")
		} else if len(matches) > 2 && matches[2] != "" {
			info.Year = strings.ReplaceAll(matches[2], " ", "")
			info.Year = strings.ReplaceAll(info.Year, "-", ",")
		}
		if len(matches) > 3 && matches[3] != "" {
			info.AdditionalText = strings.TrimSpace(matches[3])
		}
		return info
	}

	return nil
}

// FindAllCopyrightHeaders finds all copyright headers in file content
func FindAllCopyrightHeaders(content []byte) []*HeaderInfo {
	var headers []*HeaderInfo
	lines := bytes.Split(content, []byte("\n"))

	pos := 0
	for i, line := range lines {
		lineStr := string(line)

		// Only check first 50 lines for performance
		if i > 50 {
			break
		}

		if info := ParseCopyrightHeader(lineStr); info != nil {
			info.StartPos = pos
			info.EndPos = pos + len(line)
			headers = append(headers, info)
		}

		pos += len(line) + 1 // +1 for newline
	}

	return headers
}

// GetFileModificationYear gets the year when the file was last modified using git
func GetFileModificationYear(filePath string) (int, error) {
	// Try git first
	// Use the current working directory for git command
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	cmd := exec.Command("git", "log", "-1", "--format=%ad", "--date=format:%Y", "--", filePath)
	// Don't set cmd.Dir - use current working directory

	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		yearStr := strings.TrimSpace(string(output))
		if year, err := strconv.Atoi(yearStr); err == nil {
			return year, nil
		}
	}

	// Fallback to filesystem modification time
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return 0, err
	}

	return fileInfo.ModTime().Year(), nil
}

// FormatYearRange creates a year range string for IBM format
func FormatYearRange(startYear int, endYear int) string {
	if startYear == 0 && endYear == 0 {
		return fmt.Sprintf("%d", time.Now().Year())
	}
	if startYear == 0 {
		return fmt.Sprintf("%d", endYear)
	}
	if endYear == 0 || startYear == endYear {
		return fmt.Sprintf("%d", startYear)
	}
	return fmt.Sprintf("%d, %d", startYear, endYear)
}

// ParseYearRange extracts start and end year from a year string
func ParseYearRange(yearStr string) (startYear int, endYear int) {
	yearStr = strings.TrimSpace(yearStr)
	if yearStr == "" {
		return 0, 0
	}

	// Handle formats: "2020", "2020,2025", "2020-2025", "2020, 2025"
	yearStr = strings.ReplaceAll(yearStr, " ", "")
	yearStr = strings.ReplaceAll(yearStr, "-", ",")

	parts := strings.Split(yearStr, ",")
	if len(parts) >= 1 {
		if y, err := strconv.Atoi(parts[0]); err == nil {
			startYear = y
		}
	}
	if len(parts) >= 2 {
		if y, err := strconv.Atoi(parts[1]); err == nil {
			endYear = y
		}
	} else {
		endYear = startYear
	}

	return startYear, endYear
}

// GenerateUpdatedHeader creates the updated header text
func GenerateUpdatedHeader(info *HeaderInfo, newHolder string, newYear string, preserveFormat bool) string {
	var result strings.Builder

	result.WriteString("Copyright ")
	result.WriteString(newHolder)
	result.WriteString(" ")
	result.WriteString(newYear)

	if info.AdditionalText != "" {
		result.WriteString(" ")
		result.WriteString(info.AdditionalText)
	}

	return result.String()
}

// ShouldUpdateHeader determines if a header needs updating
func ShouldUpdateHeader(info *HeaderInfo, currentYear int, fileModYear int, configStartYear int) bool {
	if info == nil {
		return false
	}

	// Skip non-IBM/Hashicorp headers
	if info.Organization != "IBM" && info.Organization != "Hashicorp" {
		return false
	}

	// Always update Hashicorp headers
	if info.Organization == "Hashicorp" {
		return true
	}

	// For IBM headers, update if:
	// 1. File was modified in current year and header doesn't reflect it, OR
	// 2. Header's start year doesn't match the configured copyright year
	if info.Organization == "IBM" {
		startYear, endYear := ParseYearRange(info.Year)

		// If file was modified in current year and header doesn't reflect it
		if fileModYear == currentYear && endYear != currentYear {
			return true
		}

		// If start year doesn't match config year
		if configStartYear > 0 && startYear != configStartYear {
			return true
		}
	}

	return false
}

// CheckIfHeaderNeedsUpdate checks if a file's headers need updating without modifying the file
func CheckIfHeaderNeedsUpdate(path string, data LicenseData) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	// Skip generated files
	if isGenerated(content) {
		return false, nil
	}

	headers := FindAllCopyrightHeaders(content)
	if len(headers) == 0 {
		return false, nil
	}

	// Get file modification year
	fileModYear, err := GetFileModificationYear(path)
	if err != nil {
		fileModYear = time.Now().Year()
	}

	currentYear := time.Now().Year()
	
	// Extract config start year from data.Year
	configStartYear, _ := ParseYearRange(data.Year)
	
	// Check if any header needs updating
	for _, header := range headers {
		if ShouldUpdateHeader(header, currentYear, fileModYear, configStartYear) {
			return true, nil
		}
	}

	return false, nil
}

// UpdateFileHeaders updates all copyright headers in a file in-place
func UpdateFileHeaders(path string, fmode os.FileMode, tmpl *template.Template, data LicenseData) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	// Skip generated files
	if isGenerated(content) {
		return false, nil
	}

	headers := FindAllCopyrightHeaders(content)
	if len(headers) == 0 {
		// No headers found, let the regular addLicense handle it
		return false, nil
	}

	// Get file modification year
	fileModYear, err := GetFileModificationYear(path)
	if err != nil {
		fileModYear = time.Now().Year()
	}

	currentYear := time.Now().Year()

	// Extract config start year from data.Year (comes from FormatCopyrightYears)
	configStartYear, _ := ParseYearRange(data.Year)

	needsUpdate := false
	anyUpdated := false

	// Check if any header needs updating
	for _, header := range headers {
		if ShouldUpdateHeader(header, currentYear, fileModYear, configStartYear) {
			needsUpdate = true
			break
		}
	}

	if !needsUpdate {
		return false, nil
	}

	// Do in-place replacement of copyright lines only
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	for i, line := range lines {
		lineInfo := ParseCopyrightHeader(strings.TrimSpace(line))
		if lineInfo == nil {
			continue
		}

		if !ShouldUpdateHeader(lineInfo, currentYear, fileModYear, configStartYear) {
			continue
		}

		// Determine the new copyright text
		var newCopyrightText string

		if lineInfo.Organization == "Hashicorp" {
			// Convert Hashicorp to IBM
			startYear, _ := ParseYearRange(data.Year)
			if startYear == 0 {
				startYear = fileModYear
			}
			// Use currentYear since we're modifying the file now
			yearRange := FormatYearRange(startYear, currentYear)
			newCopyrightText = fmt.Sprintf("Copyright %s %s", data.Holder, yearRange)
		} else if lineInfo.Organization == "IBM" {
			// Update IBM year range
			// data.Year contains the config start year, use it
			startYear, _ := ParseYearRange(data.Year)
			yearRange := FormatYearRange(startYear, currentYear)
			newCopyrightText = fmt.Sprintf("Copyright %s %s", data.Holder, yearRange)
		}

		// Preserve additional text if present
		if lineInfo.AdditionalText != "" {
			newCopyrightText += " " + lineInfo.AdditionalText
		}

		// Replace the line preserving comment style
		oldLine := line
		newLine := line

		// Detect comment style and replace
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			// Go/C++ style comment
			indent := line[:strings.Index(line, "//")]
			newLine = indent + "// " + newCopyrightText
		} else if strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Shell/Python style comment
			indent := line[:strings.Index(line, "#")]
			newLine = indent + "# " + newCopyrightText
		} else if strings.Contains(line, "/*") {
			// C style block comment
			indent := line[:strings.Index(line, "/*")]
			newLine = indent + "/* " + newCopyrightText + " */"
		} else {
			// Plain text (e.g., LICENSE files)
			// Preserve leading whitespace
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(trimmed), "copyright") {
				indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				newLine = indent + newCopyrightText
			}
		}

		if oldLine != newLine {
			lines[i] = newLine
			anyUpdated = true
		}
	}

	if !anyUpdated {
		return false, nil
	}

	// Write the updated content back
	newContent := strings.Join(lines, "\n")
	err = os.WriteFile(path, []byte(newContent), fmode)
	if err != nil {
		return false, err
	}

	return true, nil
}
