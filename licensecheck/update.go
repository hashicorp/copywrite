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
	PrefixIndex  int    // Byte index in the line where the comment prefix starts (for inline comments)
}

// extractAllCopyrightInfo extracts all copyright information from a file
func extractAllCopyrightInfo(filePath string) ([]*CopyrightInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	var copyrights []*CopyrightInfo

	// Scan entire file for all copyright statements
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check if line contains "copyright"
		if strings.Contains(strings.ToLower(line), "copyright") {
			info := parseCopyrightLine(line, lineNum, filePath)
			if info != nil {
				copyrights = append(copyrights, info)
			}
		}
	}

	return copyrights, scanner.Err()
}

// parseCopyrightLine extracts copyright details from a line
func parseCopyrightLine(line string, lineNum int, filePath string) *CopyrightInfo {
	// 1. Determine the prefix and content source
	bestIdx := -1
	bestPrefix := ""
	for _, p := range commentPrefixes {
		if idx := strings.Index(line, p); idx >= 0 {
			if bestIdx == -1 || idx < bestIdx {
				bestIdx = idx
				bestPrefix = p
			}
		}
	}

	var prefix string
	var prefixStart int
	var content string

	if bestIdx != -1 {
		// Case A: Comment prefix found
		// Include contiguous spaces/tabs immediately before the prefix as part of the prefix index
		prefixStart = bestIdx
		for i := bestIdx - 1; i >= 0; i-- {
			if line[i] == ' ' || line[i] == '\t' {
				prefixStart = i
			} else {
				break
			}
		}

		// Calculate bounds
		prefixEnd := bestIdx + len(bestPrefix)
		prefix = line[prefixStart:prefixEnd]

		if prefixEnd < len(line) {
			content = line[prefixEnd:]
		}
	} else {
		// Case B: No prefix found (check if it is a LICENSE file)
		base := strings.ToLower(filepath.Base(filePath))
		if strings.HasPrefix(base, "license") {
			// Treat entire line as content with empty prefix
			prefix = ""
			prefixStart = 0
			content = line
		} else {
			// No prefix and not a license file -> Ignore
			return nil
		}
	}

	// Validate content starts with "Copyright"
	// Normalize content for the check
	content = strings.TrimSpace(content)
	if !regexp.MustCompile(`(?i)^copyright\b`).MatchString(content) {
		return nil
	}

	// Initialize the Info struct
	info := &CopyrightInfo{
		LineNumber:   lineNum,
		OriginalLine: line,
		Prefix:       prefix,
		PrefixIndex:  prefixStart,
	}

	// Parse the Copyright String (Unified Logic)
	// Remove "Copyright" and optional (c) from the beginning
	re := regexp.MustCompile(`(?i)^copyright\s*(?:\(c\))?\s*`)
	afterCopyright := re.ReplaceAllString(content, "")
	afterCopyright = strings.TrimSpace(afterCopyright)

	// Strategy: Find all 4-digit years in the line
	yearPattern := regexp.MustCompile(`\b(\d{4})\b`)
	yearMatches := yearPattern.FindAllStringIndex(afterCopyright, -1)

	// If no years found, the whole string is the holder
	if len(yearMatches) == 0 {
		info.Holder = strings.TrimSpace(afterCopyright)
		return info
	}

	// Find the last occurrence of years (which should be the copyright years)
	lastYearIdx := yearMatches[len(yearMatches)-1]

	// Extract years - check if there's a year before the last one (start year)
	if len(yearMatches) >= 2 {
		prevYearIdx := yearMatches[len(yearMatches)-2]
		between := afterCopyright[prevYearIdx[1]:lastYearIdx[0]]

		// If only separators (-, ) between them, treat as start and end year
		if strings.TrimSpace(strings.Trim(between, "-, ")) == "" {
			startYearStr := afterCopyright[prevYearIdx[0]:prevYearIdx[1]]
			if year, err := strconv.Atoi(startYearStr); err == nil {
				info.StartYear = year
			}
		}
	}

	endYearStr := afterCopyright[lastYearIdx[0]:lastYearIdx[1]]
	if year, err := strconv.Atoi(endYearStr); err == nil {
		info.EndYear = year
		if info.StartYear == 0 {
			info.StartYear = year
		}
	}

	// Determine where the Holder name ends
	// Usually, everything before the first recognized year (or year pair) is the holder
	holderEndIdx := yearMatches[0][0]
	if len(yearMatches) >= 2 && info.StartYear != 0 {
		holderEndIdx = yearMatches[len(yearMatches)-2][0]
	}

	holder := strings.TrimSpace(afterCopyright[:holderEndIdx])
	info.Holder = holder

	// Everything after the last year is trailing text
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
	for _, prefix := range commentPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return leadingSpace + prefix
		}
	}

	return leadingSpace
}

// Generated file detection patterns (from addlicense)
var (
	// go generate: ^// Code generated .* DO NOT EDIT\.$
	goGenerated = regexp.MustCompile(`(?m)^.{1,2} Code generated .* DO NOT EDIT\.$`)
	// cargo raze: ^DO NOT EDIT! Replaced on runs of cargo-raze$
	cargoRazeGenerated = regexp.MustCompile(`(?m)^DO NOT EDIT! Replaced on runs of cargo-raze$`)
	// terraform init: ^# This file is maintained automatically by "terraform init"\.$
	terraformGenerated = regexp.MustCompile(`(?m)^# This file is maintained automatically by "terraform init"\.$`)
)

// isGenerated returns true if the file content contains a string that implies
// the file was auto-generated and should not be modified.
// This prevents updating copyright headers in generated files.
func isGenerated(content []byte) bool {
	// Scan entire file for generated markers
	return goGenerated.Match(content) ||
		cargoRazeGenerated.Match(content) ||
		terraformGenerated.Match(content)
}

// Special line prefixes that should be preserved at the start of files (from addlicense)
var specialLineHeads = []string{
	"#!",                       // shell script shebang
	"<?xml",                    // XML declaration
	"<!doctype",                // HTML doctype
	"# encoding:",              // Ruby encoding
	"# frozen_string_literal:", // Ruby interpreter instruction
	"#\\",                      // Ruby Rack directive
	"<?php",                    // PHP opening tag
	"# escape",                 // Dockerfile directive
	"# syntax",                 // Dockerfile directive
	"/** @jest-environment",    // Jest Environment string
}

// Sentinel file special patterns (from addlicense)
var sentinelHeadPatterns = []string{
	`^#.*\n?(#.*\n?)*\n`,
	`^//.*\n?(//.*\n?)*\n`,
	`^/\*.*\n?(.*\n?)*\*/\n\n`,
}

// Common comment prefixes used across parsing functions. Ordered by specificity.
var commentPrefixes = []string{
	"<%/* ", "<%/*",
	"(** ", "(**",
	"/** ", "/**",
	"<!-- ", "<!--",
	"{{! ", "{{!",
	"/* ", "/*",
	";; ", ";;",
	"-- ", "--",
	"// ", "//",
	"# ", "#",
	"% ", "%",
	"* ", "*",
}

// hasSpecialFirstLine checks if the file content starts with a special line
// that should be preserved (like shebangs, XML declarations, etc.)
func hasSpecialFirstLine(content []byte, filePath string) bool {
	if len(content) == 0 {
		return false
	}

	// Check for Sentinel file patterns
	if strings.HasSuffix(strings.ToLower(filepath.Base(filePath)), ".sentinel") {
		for _, pattern := range sentinelHeadPatterns {
			if matched, _ := regexp.Match(pattern, content); matched {
				return true
			}
		}
	}

	// Get first line
	firstLine := content
	for i, c := range content {
		if c == '\n' {
			firstLine = content[:i+1]
			break
		}
	}

	// Check against special prefixes
	lowerFirst := strings.ToLower(string(firstLine))
	for _, prefix := range specialLineHeads {
		if strings.HasPrefix(lowerFirst, prefix) {
			return true
		}
	}

	return false
}

// executeGitCommand executes a git command and returns the output
func executeGitCommand(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Output()
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

// calculateYearUpdates determines if a copyright needs updating and calculates new years
// Returns: (shouldUpdate bool, newStartYear int, newEndYear int)
func calculateYearUpdates(
	info *CopyrightInfo,
	canonicalStartYear int,
	lastCommitYear int,
	currentYear int,
	forceCurrentYear bool,
) (bool, int, int) {
	shouldUpdate := false
	newStartYear := info.StartYear
	newEndYear := info.EndYear

	// Condition 1: Update start year if canonical year differs from file's start year
	if canonicalStartYear > 0 && info.StartYear != canonicalStartYear {
		newStartYear = canonicalStartYear
		shouldUpdate = true
	}

	// Condition 2: Only update end year if file was modified after the copyright end year, or forceCurrentYear is true
	if lastCommitYear > info.EndYear {
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

	return shouldUpdate, newStartYear, newEndYear
}

// getRepoRoot finds the git repository root from a given directory
func getRepoRoot(workingDir string) (string, error) {
	repoRootOutput, err := executeGitCommand(
		workingDir,
		"rev-parse", "--show-toplevel",
	)
	if err != nil {
		return "", fmt.Errorf("failed to find git repo root: %w", err)
	}
	return strings.TrimSpace(string(repoRootOutput)), nil
}

// getFileLastCommitYear returns the year of the last commit that modified a file
func getFileLastCommitYear(filePath string) (int, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return 0, err
	}

	// Find repository root
	repoRoot, err := getRepoRoot(filepath.Dir(absPath))
	if err != nil {
		return 0, err
	}

	// Calculate relative path from repo root to file
	relPath, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Run git log from repo root with relative path
	output, err := executeGitCommand(
		repoRoot,
		"log", "-1", "--format=%ad", "--date=format:%Y", "--", relPath,
	)
	if err != nil {
		return 0, err
	}

	return parseYearFromGitOutput(output, false)
}

// GetRepoFirstCommitYear returns the year of the first commit in the repository
func GetRepoFirstCommitYear(workingDir string) (int, error) {
	// Find repository root for consistency
	repoRoot, err := getRepoRoot(workingDir)
	if err != nil {
		return 0, err
	}

	output, err := executeGitCommand(repoRoot, "log", "--reverse", "--format=%ad", "--date=format:%Y")
	if err != nil {
		return 0, err
	}

	return parseYearFromGitOutput(output, true)
}

// GetRepoLastCommitYear returns the year of the last commit in the repository
func GetRepoLastCommitYear(workingDir string) (int, error) {
	// Find repository root for consistency
	repoRoot, err := getRepoRoot(workingDir)
	if err != nil {
		return 0, err
	}

	output, err := executeGitCommand(repoRoot, "log", "-1", "--format=%ad", "--date=format:%Y")
	if err != nil {
		return 0, err
	}

	return parseYearFromGitOutput(output, false)
}

// evaluateCopyrightUpdates evaluates all copyrights in a file and returns which ones need updating
// This is shared logic between UpdateCopyrightHeader and NeedsUpdate
func evaluateCopyrightUpdates(
	copyrights []*CopyrightInfo,
	targetHolder string,
	configYear int,
	lastCommitYear int,
	currentYear int,
	forceCurrentYear bool,
	repoFirstYear int,
) []*struct {
	info         *CopyrightInfo
	newStartYear int
	newEndYear   int
} {
	// If configYear is 0, use repo's first commit year
	canonicalStartYear := configYear
	if canonicalStartYear == 0 && repoFirstYear > 0 {
		canonicalStartYear = repoFirstYear
	}

	var updates []*struct {
		info         *CopyrightInfo
		newStartYear int
		newEndYear   int
	}

	// Process each copyright statement
	for _, info := range copyrights {
		// Check if holder matches target (case-insensitive partial match)
		if !strings.Contains(strings.ToLower(info.Holder), strings.ToLower(targetHolder)) {
			continue
		}

		shouldUpdate, newStartYear, newEndYear := calculateYearUpdates(
			info, canonicalStartYear, lastCommitYear, currentYear, forceCurrentYear,
		)

		if shouldUpdate {
			updates = append(updates, &struct {
				info         *CopyrightInfo
				newStartYear int
				newEndYear   int
			}{
				info:         info,
				newStartYear: newStartYear,
				newEndYear:   newEndYear,
			})
		}
	}

	return updates
}

// UpdateCopyrightHeader updates all copyright headers in a file if needed
// If forceCurrentYear is true, forces end year to current year regardless of git history
// Returns true if the file was modified
func UpdateCopyrightHeader(filePath string, targetHolder string, configYear int, forceCurrentYear bool) (bool, error) {
	// Skip .copywrite.hcl config file
	if filepath.Base(filePath) == ".copywrite.hcl" {
		return false, nil
	}

	// Read file content once for all checks
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	// Skip generated files (DO NOT EDIT markers, etc.)
	if isGenerated(content) {
		return false, nil
	}

	// Extract all copyright statements in the file
	copyrights, err := extractAllCopyrightInfo(filePath)
	if err != nil {
		return false, err
	}

	if len(copyrights) == 0 {
		// No copyright headers found
		return false, nil
	}

	currentYear := time.Now().Year()
	lastCommitYear, _ := getFileLastCommitYear(filePath)
	repoFirstYear, _ := GetRepoFirstCommitYear(filepath.Dir(filePath))

	// Evaluate which copyrights need updating
	updates := evaluateCopyrightUpdates(
		copyrights, targetHolder, configYear, lastCommitYear, currentYear, forceCurrentYear, repoFirstYear,
	)

	if len(updates) == 0 {
		return false, nil
	}

	// Apply updates
	lines := strings.Split(string(content), "\n")
	for _, update := range updates {
		info := update.info
		if info.LineNumber < 1 || info.LineNumber > len(lines) {
			continue
		}

		// Reconstruct the copyright fragment preserving format and trailing text.
		var yearStr string
		if update.newStartYear == update.newEndYear {
			yearStr = fmt.Sprintf("%d", update.newEndYear)
		} else {
			yearStr = fmt.Sprintf("%d, %d", update.newStartYear, update.newEndYear)
		}

		// Build the new copyright text (prefix included)
		newCopyright := fmt.Sprintf("%sCopyright %s %s", info.Prefix, info.Holder, yearStr)
		if info.TrailingText != "" {
			newCopyright += info.TrailingText
		}

		// If PrefixIndex is set, replace only the comment suffix starting at PrefixIndex,
		// preserving any code before the comment (inline comment case).
		idx := info.LineNumber - 1
		origLine := lines[idx]
		if info.PrefixIndex > 0 && info.PrefixIndex < len(origLine) {
			lines[idx] = origLine[:info.PrefixIndex] + newCopyright
		} else {
			// PrefixIndex == 0 or out-of-range: replace the whole line
			lines[idx] = newCopyright
		}
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

	// Read file content for generated file check
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	// Skip generated files (DO NOT EDIT markers, etc.)
	if isGenerated(content) {
		return false, nil
	}

	// Extract all copyright statements in the file
	copyrights, err := extractAllCopyrightInfo(filePath)
	if err != nil {
		return false, err
	}

	if len(copyrights) == 0 {
		return false, nil
	}

	currentYear := time.Now().Year()
	lastCommitYear, _ := getFileLastCommitYear(filePath)
	repoFirstYear, _ := GetRepoFirstCommitYear(filepath.Dir(filePath))

	// Evaluate which copyrights need updating
	updates := evaluateCopyrightUpdates(
		copyrights, targetHolder, configYear, lastCommitYear, currentYear, forceCurrentYear, repoFirstYear,
	)

	return len(updates) > 0, nil
}
