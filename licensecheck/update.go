// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package licensecheck

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CopyrightInfo holds parsed copyright information from a file
type CopyrightInfo struct {
	LineNumber   int
	OriginalLine string
	Holder       string
	StartYear    int
	EndYear      int
	Prefix       string // Comment prefix (e.g., "// ", "# ")
	TrailingText string // Any text after the years
}

// ExtractAllCopyrightInfo extracts all copyright information from a file
func ExtractAllCopyrightInfo(filePath string) ([]*CopyrightInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var copyrights []*CopyrightInfo

	// Scan entire file for all copyright statements
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check if line contains "copyright"
		if strings.Contains(strings.ToLower(line), "copyright") {
			info := parseCopyrightLine(line, lineNum)
			if info != nil {
				copyrights = append(copyrights, info)
			}
		}
	}

	return copyrights, scanner.Err()
}

// ExtractCopyrightInfo extracts the first copyright information from a file (for compatibility)
func ExtractCopyrightInfo(filePath string) (*CopyrightInfo, error) {
	copyrights, err := ExtractAllCopyrightInfo(filePath)
	if err != nil {
		return nil, err
	}
	if len(copyrights) == 0 {
		return nil, nil
	}
	return copyrights[0], nil
}

// parseCopyrightLine extracts copyright details from a line
func parseCopyrightLine(line string, lineNum int) *CopyrightInfo {
	// Extract comment prefix
	prefix := extractCommentPrefix(line)

	// Get the content after the prefix
	contentStart := len(prefix)
	if contentStart >= len(line) {
		return nil
	}
	content := line[contentStart:]

	// Must contain "copyright"
	if !strings.Contains(strings.ToLower(content), "copyright") {
		return nil
	}

	info := &CopyrightInfo{
		LineNumber:   lineNum,
		OriginalLine: line,
		Prefix:       prefix,
	}

	// Remove "Copyright" and optional (c) from the beginning
	re := regexp.MustCompile(`(?i)^copyright\s*(?:\(c\))?\s*`)
	afterCopyright := re.ReplaceAllString(content, "")
	afterCopyright = strings.TrimSpace(afterCopyright)

	// Strategy: Find all 4-digit years in the line
	yearPattern := regexp.MustCompile(`\b(\d{4})\b`)
	yearMatches := yearPattern.FindAllStringIndex(afterCopyright, -1)

	if len(yearMatches) == 0 {
		// No year found, everything is the holder
		info.Holder = strings.TrimSpace(afterCopyright)
		return info
	}

	// Find the last occurrence of years (which should be the copyright years)
	// Look for patterns like "YYYY" or "YYYY, YYYY" or "YYYY-YYYY"
	lastYearIdx := yearMatches[len(yearMatches)-1]

	// Extract years - check if there's a year before the last one (start year)
	if len(yearMatches) >= 2 {
		// Check if the previous year is close to the last year (within 20 chars)
		prevYearIdx := yearMatches[len(yearMatches)-2]
		between := afterCopyright[prevYearIdx[1]:lastYearIdx[0]]

		// If only separators between them, treat as start and end year
		if strings.TrimSpace(strings.Trim(between, "-, ")) == "" {
			startYearStr := afterCopyright[prevYearIdx[0]:prevYearIdx[1]]
			if year, err := strconv.Atoi(startYearStr); err == nil {
				info.StartYear = year
			}
		}
	}

	// Extract the last year (end year or only year)
	endYearStr := afterCopyright[lastYearIdx[0]:lastYearIdx[1]]
	if year, err := strconv.Atoi(endYearStr); err == nil {
		info.EndYear = year
		if info.StartYear == 0 {
			info.StartYear = year
		}
	}

	// Everything before the first year (or before the pair of years) is the holder
	holderEndIdx := yearMatches[0][0]
	if len(yearMatches) >= 2 && info.StartYear != 0 {
		holderEndIdx = yearMatches[len(yearMatches)-2][0]
	}

	holder := strings.TrimSpace(afterCopyright[:holderEndIdx])
	info.Holder = holder

	// Everything after the last year is trailing text - preserve it exactly
	if lastYearIdx[1] < len(afterCopyright) {
		trailing := afterCopyright[lastYearIdx[1]:]
		if trailing != "" {
			info.TrailingText = trailing
		}
	}

	return info
}

// extractCommentPrefix extracts comment markers from the beginning of a line
func extractCommentPrefix(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	leadingSpace := line[:len(line)-len(trimmed)]

	// Check for common comment prefixes
	commentPrefixes := []string{"// ", "//", "# ", "#", "* ", "*", "/* "}

	for _, prefix := range commentPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return leadingSpace + prefix
		}
	}

	return leadingSpace
}

// parseYearFromGitOutput parses the year from git command output
func parseYearFromGitOutput(output []byte, useFirstLine bool) (int, error) {
	yearStr := strings.TrimSpace(string(output))
	if yearStr == "" {
		return 0, fmt.Errorf("no commits found")
	}

	// For commands with multiple lines, extract the first line if requested
	if useFirstLine && strings.Contains(yearStr, "\n") {
		lines := strings.Split(yearStr, "\n")
		if len(lines) > 0 {
			yearStr = strings.TrimSpace(lines[0])
		}
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return 0, err
	}

	return year, nil
}

// GetFileLastCommitYear returns the year of the last commit that modified a file
func GetFileLastCommitYear(filePath string) (int, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return 0, err
	}

	cmd := exec.Command("git", "log", "-1", "--format=%ad", "--date=format:%Y", "--", filepath.Base(absPath))
	cmd.Dir = filepath.Dir(absPath)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	return parseYearFromGitOutput(output, false)
}

// GetRepoFirstCommitYear returns the year of the first commit in the repository
func GetRepoFirstCommitYear(workingDir string) (int, error) {
	cmd := exec.Command("git", "log", "--reverse", "--format=%ad", "--date=format:%Y")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	return parseYearFromGitOutput(output, true)
}

// GetRepoLastCommitYear returns the year of the last commit in the repository
func GetRepoLastCommitYear(workingDir string) (int, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%ad", "--date=format:%Y")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	return parseYearFromGitOutput(output, false)
}

// UpdateCopyrightHeader updates all copyright headers in a file if needed
// If forceCurrentYear is true, forces end year to current year regardless of git history
// Returns true if the file was modified
func UpdateCopyrightHeader(filePath string, targetHolder string, configYear int, forceCurrentYear bool) (bool, error) {
	// Skip .copywrite.hcl config file
	if filepath.Base(filePath) == ".copywrite.hcl" {
		return false, nil
	}

	// Extract all copyright statements in the file
	copyrights, err := ExtractAllCopyrightInfo(filePath)
	if err != nil {
		return false, err
	}

	if len(copyrights) == 0 {
		// No copyright headers found
		return false, nil
	}

	currentYear := time.Now().Year()

	// Get last commit year once for the file
	lastCommitYear, _ := GetFileLastCommitYear(filePath)

	// If configYear is 0, try to auto-detect from repo's first commit
	canonicalStartYear := configYear
	if canonicalStartYear == 0 {
		if repoFirstYear, err := GetRepoFirstCommitYear(filepath.Dir(filePath)); err == nil && repoFirstYear > 0 {
			canonicalStartYear = repoFirstYear
		}
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(content), "\n")
	modified := false

	// Process each copyright statement
	for _, info := range copyrights {
		// Check if holder matches target (case-insensitive partial match)
		if !strings.Contains(strings.ToLower(info.Holder), strings.ToLower(targetHolder)) {
			continue
		}

		shouldUpdate := false
		newStartYear := info.StartYear
		newEndYear := info.EndYear

		// Condition 1: Update start year if canonical year differs from file's start year
		if canonicalStartYear > 0 && info.StartYear != canonicalStartYear {
			newStartYear = canonicalStartYear
			shouldUpdate = true
		}

		// Condition 2: Check if file was modified after the copyright end year OR we're making any update
		if lastCommitYear > info.EndYear || shouldUpdate {
			// File was modified after copyright end year (or will be modified by us), update end year
			if info.EndYear < currentYear {
				newEndYear = currentYear
				shouldUpdate = true
			}
		}

		// Condition 3: Force current year if requested (e.g., for LICENSE when other files updated)
		if forceCurrentYear && info.EndYear < currentYear {
			newEndYear = currentYear
			shouldUpdate = true
		}

		if !shouldUpdate {
			continue
		}

		if info.LineNumber < 1 || info.LineNumber > len(lines) {
			continue
		}

		// Reconstruct the copyright line preserving format and trailing text
		var yearStr string
		if newStartYear == newEndYear {
			yearStr = fmt.Sprintf("%d", newEndYear)
		} else {
			yearStr = fmt.Sprintf("%d, %d", newStartYear, newEndYear)
		}

		// Build new line: prefix + "Copyright " + holder + " " + years + trailing
		newLine := fmt.Sprintf("%sCopyright %s %s", info.Prefix, info.Holder, yearStr)
		if info.TrailingText != "" {
			newLine += info.TrailingText
		}

		lines[info.LineNumber-1] = newLine
		modified = true
	}

	if !modified {
		return false, nil
	}

	// Write back
	newContent := strings.Join(lines, "\n")
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return false, err
	}

	return true, nil
}

// NeedsUpdate checks if a file would be updated without actually modifying it
// If forceCurrentYear is true, forces end year to current year regardless of git history
// Returns true if the file has copyrights matching targetHolder that need year updates
func NeedsUpdate(filePath string, targetHolder string, configYear int, forceCurrentYear bool) (bool, error) {
	// Skip .copywrite.hcl config file
	if filepath.Base(filePath) == ".copywrite.hcl" {
		return false, nil
	}

	// Extract all copyright statements in the file
	copyrights, err := ExtractAllCopyrightInfo(filePath)
	if err != nil {
		return false, err
	}

	if len(copyrights) == 0 {
		return false, nil
	}

	currentYear := time.Now().Year()

	// Get last commit year once for the file
	lastCommitYear, _ := GetFileLastCommitYear(filePath)

	// If configYear is 0, try to auto-detect from repo's first commit
	canonicalStartYear := configYear
	if canonicalStartYear == 0 {
		if repoFirstYear, err := GetRepoFirstCommitYear(filepath.Dir(filePath)); err == nil && repoFirstYear > 0 {
			canonicalStartYear = repoFirstYear
		}
	}

	// Process each copyright statement
	for _, info := range copyrights {
		// Check if holder matches target (case-insensitive partial match)
		if !strings.Contains(strings.ToLower(info.Holder), strings.ToLower(targetHolder)) {
			continue
		}

		needsUpdate := false

		// Condition 1: Update start year if canonical year differs from file's start year
		if canonicalStartYear > 0 && info.StartYear != canonicalStartYear {
			needsUpdate = true
		}

		// Condition 2: Check if file was modified after the copyright end year OR we're making any update
		if lastCommitYear > info.EndYear || needsUpdate {
			// File was modified after copyright end year (or will be modified by us), update end year
			if info.EndYear < currentYear {
				needsUpdate = true
			}
		}

		// Condition 3: Force current year if requested
		if forceCurrentYear && info.EndYear < currentYear {
			needsUpdate = true
		}

		if needsUpdate {
			return true, nil
		}
	}

	return false, nil
}
