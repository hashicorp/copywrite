// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This program ensures source code files have copyright license headers.
// See usage with "addlicense -h".

package addlicense

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"golang.org/x/sync/errgroup"
)

const helpText = `Usage: addlicense [flags] pattern [pattern ...]

The program ensures source code files have copyright license headers
by scanning directory patterns recursively.

It modifies all source files in place and avoids adding a license header
to any file that already has one.

The pattern argument can be provided multiple times, and may also refer
to single files.

Flags:

`

var (
	skipExtensionFlags stringSlice
	ignorePatterns     stringSlice
	spdx               spdxFlag

	holder    = flag.String("c", "Google LLC", "copyright holder")
	license   = flag.String("l", "apache", "license type: apache, bsd, mit, mpl")
	licensef  = flag.String("f", "", "license file")
	year      = flag.String("y", fmt.Sprint(time.Now().Year()), "copyright year(s)")
	verbose   = flag.Bool("v", false, "verbose mode: print the name of the files that are modified")
	checkonly = flag.Bool("check", false, "check only mode: verify presence of license headers and exit with non-zero code if missing")
)

func init() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, helpText)
		flag.PrintDefaults()
	}
	flag.Var(&skipExtensionFlags, "skip", "[deprecated: see -ignore] file extensions to skip, for example: -skip rb -skip go")
	flag.Var(&ignorePatterns, "ignore", "file patterns to ignore, for example: -ignore **/*.go -ignore vendor/**")
	flag.Var(&spdx, "s", "Include SPDX identifier in license header. Set -s=only to only include SPDX identifier.")
}

// stringSlice stores the results of a repeated command line flag as a string slice.
type stringSlice []string

func (i *stringSlice) String() string {
	return fmt.Sprint(*i)
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// spdxFlag defines the line flag behavior for specifying SPDX support.
type spdxFlag string

const (
	spdxOff  spdxFlag = ""
	spdxOn   spdxFlag = "true" // value set by flag package on bool flag
	spdxOnly spdxFlag = "only"
)

// IsBoolFlag causes a bare '-s' flag to be set as the string 'true'.  This
// allows the use of the bare '-s' or setting a string '-s=only'.
func (i *spdxFlag) IsBoolFlag() bool { return true }
func (i *spdxFlag) String() string   { return string(*i) }

func (i *spdxFlag) Set(value string) error {
	v := spdxFlag(value)
	if v != spdxOn && v != spdxOnly {
		return fmt.Errorf("error: flag 's' expects '%v' or '%v'", spdxOn, spdxOnly)
	}
	*i = v
	return nil
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Get non-flag command-line args
	patterns := flag.Args()

	// convert -skip flags to -ignore equivalents
	for _, s := range skipExtensionFlags {
		ignorePatterns = append(ignorePatterns, fmt.Sprintf("**/*.%s", s))
	}

	// map legacy license values
	if t, ok := legacyLicenseTypes[*license]; ok {
		*license = t
	}

	data := LicenseData{
		Year:   *year,
		Holder: *holder,
		SPDXID: *license,
	}

	// create logger to print updates to stdout
	logger := log.Default()

	// real main
	err := Run(
		ignorePatterns,
		spdx,
		data,
		*licensef,
		*verbose,
		*checkonly,
		patterns,
		logger,
	)

	if err != nil {
		if err.Error() == "missing license header" {
			// this retains the historical behavior of addLicense, which is to give a
			// non-zero exit code when the -check flag is used and headers are needed
			os.Exit(1)
		} else {
			log.Fatal(err)
		}
	}
}

func validatePatterns(patterns []string) error {
	invalidPatterns := []string{}
	for _, p := range patterns {
		if !doublestar.ValidatePattern(p) {
			invalidPatterns = append(invalidPatterns, p)
		}
	}

	if len(invalidPatterns) == 1 {
		return fmt.Errorf("headerignore pattern %q is not valid", invalidPatterns[0])
	} else if len(invalidPatterns) > 1 {
		return fmt.Errorf("headerignore patterns %q are not valid", strings.Join(invalidPatterns[:], `, `))
	}

	return nil
}

// Run executes addLicense with supplied variables
func Run(
	ignorePatternList []string,
	spdx spdxFlag,
	license LicenseData,
	licenseFileOverride string, // Provide a file to use as the license header
	verbose bool,
	checkonly bool,
	patterns []string,
	logger *log.Logger,
) error {
	// verify that all ignorePatterns are valid
	err := validatePatterns(ignorePatternList)
	if err != nil {
		return err
	}
	ignorePatterns = ignorePatternList

	tpl, err := fetchTemplate(license.SPDXID, licenseFileOverride, spdx)
	if err != nil {
		return err
	}
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}

	// process at most 1000 files in parallel
	ch := make(chan *file, 1000)
	done := make(chan struct{})
	var out error
	go func() {
		var wg errgroup.Group
		for f := range ch {
			f := f // https://golang.org/doc/faq#closures_and_goroutines
			wg.Go(func() error {
				err := processFile(f, t, license, checkonly, verbose, logger)
				return err
			})
		}
		out = wg.Wait()
		close(done)
	}()

	for _, d := range patterns {
		if err := walk(ch, d, logger); err != nil {
			return err
		}
	}
	close(ch)
	<-done

	return out
}

func processFile(f *file, t *template.Template, license LicenseData, checkonly bool, verbose bool, logger *log.Logger) error {
	if checkonly {
		// Check if file extension is known
		lic, err := licenseHeader(f.path, t, license)
		if err != nil {
			logger.Printf("%s: %v", f.path, err)
			return err
		}
		if lic == nil { // Unknown fileExtension
			return nil
		}
		// Check if file has a license
		hasLicense, err := fileHasLicense(f.path)
		if err != nil {
			logger.Printf("%s: %v", f.path, err)
			return err
		}
		if !hasLicense {
			logger.Printf("%s\n", f.path)
			return errors.New("missing license header")
		}
	} else {
		modified, err := addLicense(f.path, f.mode, t, license)
		if err != nil {
			logger.Printf("%s: %v", f.path, err)
			return err
		}
		if verbose && modified {
			logger.Printf("%s modified", f.path)
		}
	}
	return nil
}

type file struct {
	path string
	mode os.FileMode
}

func walk(ch chan<- *file, start string, logger *log.Logger) error {
	return filepath.Walk(start, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			logger.Printf("%s error: %v", path, err)
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		if fileMatches(path, ignorePatterns) {
			// The [DEBUG] level is inferred by go-hclog as a debug statement
			logger.Printf("[DEBUG] skipping: %s", path)
			return nil
		}
		ch <- &file{path, fi.Mode()}
		return nil
	})
}

// fileMatches determines if path matches one of the provided file patterns.
// Patterns are assumed to be valid.
func fileMatches(path string, patterns []string) bool {
	for _, p := range patterns {

		if runtime.GOOS == "windows" {
			// If on windows, change path seperators to /
			// in order for patterns to compare correctly
			path = filepath.ToSlash(path)
		}

		// ignore error, since we assume patterns are valid
		if match, _ := doublestar.Match(p, path); match {
			return true
		}
	}
	return false
}

// addLicense add a license to the file if missing.
//
// It returns true if the file was updated.
func addLicense(path string, fmode os.FileMode, tmpl *template.Template, data LicenseData) (bool, error) {
	var lic []byte
	var err error
	lic, err = licenseHeader(path, tmpl, data)
	if err != nil || lic == nil {
		return false, err
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if hasLicense(b) || isGenerated(b) {
		return false, err
	}

	line := hashBang(b, path)
	if len(line) > 0 {
		b = b[len(line):]
		if line[len(line)-1] != '\n' {
			line = append(line, '\n')
		}
		lic = append(line, lic...)
	}
	b = append(lic, b...)
	return true, os.WriteFile(path, b, fmode)
}

// fileHasLicense reports whether the file at path contains a license header.
func fileHasLicense(path string) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	// If generated, we count it as if it has a license.
	return hasLicense(b) || isGenerated(b), nil
}

// licenseHeader populates the provided license template with data, and returns
// it with the proper prefix for the file type specified by path. The file does
// not need to actually exist, only its name is used to determine the prefix.
func licenseHeader(path string, tmpl *template.Template, data LicenseData) ([]byte, error) {
	var lic []byte
	var err error
	base := strings.ToLower(filepath.Base(path))

	switch fileExtension(base) {
	case ".c", ".h", ".gv", ".java", ".scala", ".kt", ".kts":
		lic, err = executeTemplate(tmpl, data, "/*", " * ", " */")
	case ".js", ".mjs", ".cjs", ".jsx", ".tsx", ".css", ".scss", ".sass", ".ts", ".gjs", ".gts":
		lic, err = executeTemplate(tmpl, data, "/**", " * ", " */")
	case ".cc", ".cpp", ".cs", ".go", ".hh", ".hpp", ".m", ".mm", ".proto", ".rs", ".swift", ".dart", ".groovy", ".v", ".sv", ".lr":
		lic, err = executeTemplate(tmpl, data, "", "// ", "")
	case ".py", ".sh", ".bash", ".zsh", ".yaml", ".yml", ".dockerfile", "dockerfile", ".rb", "gemfile", ".ru", ".tcl", ".hcl", ".tf", ".tfvars", ".nomad", ".bzl", ".pl", ".pp", ".ps1", ".psd1", ".psm1", ".txtar", ".sentinel":
		lic, err = executeTemplate(tmpl, data, "", "# ", "")
	case ".el", ".lisp":
		lic, err = executeTemplate(tmpl, data, "", ";; ", "")
	case ".erl":
		lic, err = executeTemplate(tmpl, data, "", "% ", "")
	case ".hs", ".sql", ".sdl":
		lic, err = executeTemplate(tmpl, data, "", "-- ", "")
	case ".hbs":
		lic, err = executeTemplate(tmpl, data, "{{!", "  ", "}}")
	case ".html", ".htm", ".xml", ".vue", ".wxi", ".wxl", ".wxs":
		lic, err = executeTemplate(tmpl, data, "<!--", " ", "-->")
	case ".php":
		lic, err = executeTemplate(tmpl, data, "", "// ", "")
	case ".ml", ".mli", ".mll", ".mly":
		lic, err = executeTemplate(tmpl, data, "(**", "   ", "*)")
	case ".ejs":
		lic, err = executeTemplate(tmpl, data, "<%/*", "  ", "*/%>")
	default:
		// handle various cmake files
		if base == "cmakelists.txt" || strings.HasSuffix(base, ".cmake.in") || strings.HasSuffix(base, ".cmake") {
			lic, err = executeTemplate(tmpl, data, "", "# ", "")
		}
	}
	return lic, err
}

// fileExtension returns the file extension of name, or the full name if there
// is no extension.
func fileExtension(name string) string {
	if v := filepath.Ext(name); v != "" {
		return v
	}
	return name
}

var head = []string{
	"#!",                       // shell script
	"<?xml",                    // XML declaratioon
	"<!doctype",                // HTML doctype
	"# encoding:",              // Ruby encoding
	"# frozen_string_literal:", // Ruby interpreter instruction
	"#\\",                      // Ruby Rack directive https://github.com/rack/rack/wiki/(tutorial)-rackup-howto
	"<?php",                    // PHP opening tag
	"# escape",                 // Dockerfile directive https://docs.docker.com/engine/reference/builder/#parser-directives
	"# syntax",                 // Dockerfile directive https://docs.docker.com/engine/reference/builder/#parser-directives
	"/** @jest-environment",    // Jest Environment string https://jestjs.io/docs/configuration#testenvironment-string
}

// We need to skip the top file comments in sentinel files because they are are currently used to
// show policy text in UI in TFC. The patterns are created based on the comment format given
// in https://developer.hashicorp.com/sentinel/docs/language/spec#comments
var sentinelHeadPatterns = []string{
	`^#.*\n?(#.*\n?)*\n`,
	`^//.*\n?(//.*\n?)*\n`,
	`^/\*.*\n?(.*\n?)*\*/\n\n`,
}

// matches regex patterns to extract headings to skip
func matchPattern(b []byte, path string) []byte {
	base := strings.ToLower(filepath.Base(path))
	var headPatterns []string
	switch fileExtension(base) {
	case ".sentinel":
		headPatterns = sentinelHeadPatterns
	default:
		headPatterns = []string{}
	}

	for _, v := range headPatterns {
		re := regexp.MustCompile(v)
		match := re.Find(b)
		if len(match) > 0 {
			return match
		}
	}
	return []byte{}
}

func hashBang(b []byte, path string) []byte {
	var line []byte

	line = matchPattern(b, path)
	if len(line) > 0 {
		return line
	}

	for _, c := range b {
		line = append(line, c)
		if c == '\n' {
			break
		}
	}
	first := strings.ToLower(string(line))
	for _, h := range head {
		if strings.HasPrefix(first, h) {
			return line
		}
	}

	return nil
}

// go generate: ^// Code generated .* DO NOT EDIT\.$
var goGenerated = regexp.MustCompile(`(?m)^.{1,2} Code generated .* DO NOT EDIT\.$`)

// cargo raze: ^DO NOT EDIT! Replaced on runs of cargo-raze$
var cargoRazeGenerated = regexp.MustCompile(`(?m)^DO NOT EDIT! Replaced on runs of cargo-raze$`)

// terraform init: ^# This file is maintained automatically by "terraform init"\.$
var terraformGenerated = regexp.MustCompile(`(?m)^# This file is maintained automatically by "terraform init"\.$`)

// isGenerated returns true if it contains a string that implies the file was
// generated.
func isGenerated(b []byte) bool {
	return goGenerated.Match(b) || cargoRazeGenerated.Match(b) || terraformGenerated.Match(b)
}

func hasLicense(b []byte) bool {
	n := 1000
	if len(b) < 1000 {
		n = len(b)
	}
	return bytes.Contains(bytes.ToLower(b[:n]), []byte("copyright")) ||
		bytes.Contains(bytes.ToLower(b[:n]), []byte("mozilla public")) ||
		bytes.Contains(bytes.ToLower(b[:n]), []byte("spdx-license-identifier"))
}

// hasCopyrightButNotHashiCorpOrIBM checks if file has copyright from other companies
// that should NOT be modified (only HashiCorp and IBM copyrights should be processed)
func hasCopyrightButNotHashiCorpOrIBM(b []byte) bool {
	n := 1000
	if len(b) < 1000 {
		n = len(b)
	}

	content := string(bytes.ToLower(b[:n]))

	// First, check for actual copyright header patterns in the top few lines (first 300 chars)
	// This is where copyright headers typically appear
	topContent := content
	if len(content) > 300 {
		topContent = content[:300]
	}

	// Check for copyright regex patterns primarily in the top section
	copyrightRegex := regexp.MustCompile(`copyright\s+\d{4}`)
	if copyrightRegex.MatchString(topContent) {
		// Found copyright with year in top section - check if it's HashiCorp or IBM
		if strings.Contains(topContent, "hashicorp") || strings.Contains(topContent, "ibm corp") {
			return false // It's HashiCorp or IBM, we should process it
		}
		return true // It's another company's copyright, don't modify
	}

	// If no copyright regex pattern in top section, check for other copyright header patterns
	hasCopyrightHeader := strings.Contains(topContent, "copyright (c)") ||
		strings.Contains(topContent, "copyright ©") ||
		strings.Contains(topContent, "copyright:")

	// If found copyright header patterns in top section, check ownership
	if hasCopyrightHeader {
		if strings.Contains(topContent, "hashicorp") || strings.Contains(topContent, "ibm corp") {
			return false // It's HashiCorp or IBM, we should process it
		}
		return true // It's another company's copyright, don't modify
	}

	// No copyright header patterns found in top section, check entire content for any copyright mentions
	// but be more restrictive - only consider it a real copyright if it has specific patterns
	fullContentHasCopyright := strings.Contains(content, "copyright (c)") ||
		strings.Contains(content, "copyright ©") ||
		strings.Contains(content, "copyright:") ||
		copyrightRegex.MatchString(content)

	if !fullContentHasCopyright {
		return false // No actual copyright header found anywhere
	}

	// Found copyright patterns somewhere in file - check if it's HashiCorp or IBM
	if strings.Contains(content, "hashicorp") || strings.Contains(content, "ibm corp") {
		return false // It's HashiCorp or IBM, we should process it
	}

	// Has actual copyright header from another company - don't modify
	return true
}

// RunUpdate executes addLicense with supplied variables, but instead of only adding
// headers to files that don't have them, it also updates existing HashiCorp headers
// to IBM headers and updates existing IBM headers with new year/license information
func RunUpdate(
	ignorePatternList []string,
	spdx spdxFlag,
	license LicenseData,
	licenseFileOverride string, // Provide a file to use as the license header
	verbose bool,
	checkonly bool,
	patterns []string,
	logger *log.Logger,
) error {
	// Set the target license data for comparison
	setTargetLicenseData(license)

	// verify that all ignorePatterns are valid
	err := validatePatterns(ignorePatternList)
	if err != nil {
		return err
	}
	ignorePatterns = ignorePatternList

	tpl, err := fetchTemplate(license.SPDXID, licenseFileOverride, spdx)
	if err != nil {
		return err
	}
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return err
	}

	// process at most 1000 files in parallel
	ch := make(chan *file, 1000)
	done := make(chan struct{})
	var out error
	go func() {
		var wg errgroup.Group
		for f := range ch {
			f := f // https://golang.org/doc/faq#closures_and_goroutines
			wg.Go(func() error {
				err := processFileUpdate(f, t, license, checkonly, verbose, logger)
				return err
			})
		}
		out = wg.Wait()
		close(done)
	}()

	for _, d := range patterns {
		if err := walk(ch, d, logger); err != nil {
			return err
		}
	}
	close(ch)
	<-done

	return out
}

// processFileUpdate processes a file for the update command, which handles both
// adding headers to files without them and replacing HashiCorp headers with IBM headers
func processFileUpdate(f *file, t *template.Template, license LicenseData, checkonly bool, verbose bool, logger *log.Logger) error {
	if checkonly {
		// Check if file extension is known
		lic, err := licenseHeader(f.path, t, license)
		if err != nil {
			logger.Printf("%s: %v", f.path, err)
			return err
		}
		if lic == nil { // Unknown fileExtension
			return nil
		}

		// Check if file needs updating (either no license or has HashiCorp header)
		needsUpdate, err := fileNeedsUpdate(f.path)
		if err != nil {
			logger.Printf("%s: %v", f.path, err)
			return err
		}
		if needsUpdate {
			logger.Printf("%s\n", f.path)
			return errors.New("file needs header update")
		}
	} else {
		modified, err := updateLicense(f.path, f.mode, t, license)
		if err != nil {
			logger.Printf("%s: %v", f.path, err)
			return err
		}
		if verbose && modified {
			logger.Printf("%s modified", f.path)
		}
	}
	return nil
}

// fileNeedsUpdate reports whether the file at path needs a header update
// (only HashiCorp headers or IBM headers with different year/license info)
func fileNeedsUpdate(path string) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	// If generated, we don't update it
	if isGenerated(b) {
		return false, nil
	}

	// If no license header at all, do NOT update (removed this feature)
	if !hasLicense(b) {
		return false, nil
	}

	// If it has a HashiCorp header, it needs to be replaced
	if hasHashiCorpHeader(b) {
		return true, nil
	}

	// If it has an IBM header, check if it needs to be updated
	if hasIBMHeader(b) {
		return hasIBMHeaderNeedingUpdate(b), nil
	}

	// If it has SPDX but no copyright header (license-only files), it needs copyright added
	n := len(b)
	if n > 1000 {
		n = 1000
	}

	content := strings.ToLower(string(b[:n]))

	// First, check for actual copyright header patterns in the top few lines (first 300 chars)
	// This is where copyright headers typically appear
	topContent := content
	if len(content) > 300 {
		topContent = content[:300]
	}

	// Check for copyright regex patterns primarily in the top section
	copyrightRegex := regexp.MustCompile(`copyright\s+\d{4}`)
	if copyrightRegex.MatchString(topContent) {
		return false, nil // Found copyright with year in top section, don't modify
	}

	// If no copyright regex pattern in top section, check for other copyright header patterns
	hasCopyrightHeader := strings.Contains(topContent, "copyright (c)") ||
		strings.Contains(topContent, "copyright ©") ||
		strings.Contains(topContent, "copyright:")

	if hasCopyrightHeader {
		return false, nil // Found copyright header patterns in top section, don't modify
	}

	// No copyright header patterns found in top section, check entire content for any copyright mentions
	// but be more restrictive - only consider it a real copyright if it has specific patterns
	fullContentHasCopyright := strings.Contains(content, "copyright (c)") ||
		strings.Contains(content, "copyright ©") ||
		strings.Contains(content, "copyright:") ||
		copyrightRegex.MatchString(content)

	if !fullContentHasCopyright {
		return true, nil // No actual copyright header found anywhere, needs update
	}

	// File has copyright from other companies, don't modify
	return false, nil
}

// hasHashiCorpHeader checks if the file contains a HashiCorp copyright header
// This function is comprehensive and detects various forms of HashiCorp headers,
// including those with additional text or formatting variations
func hasHashiCorpHeader(b []byte) bool {
	n := 1000
	if len(b) < 1000 {
		n = len(b)
	}
	content := string(bytes.ToLower(b[:n]))

	// Split content into lines for line-by-line analysis
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		// Clean the line by removing comment markers and extra whitespace
		cleanLine := strings.TrimSpace(line)
		cleanLine = strings.TrimPrefix(cleanLine, "//")
		cleanLine = strings.TrimPrefix(cleanLine, "/*")
		cleanLine = strings.TrimPrefix(cleanLine, "*")
		cleanLine = strings.TrimPrefix(cleanLine, "#")
		cleanLine = strings.TrimPrefix(cleanLine, "<!--")
		cleanLine = strings.TrimSpace(cleanLine)

		// Check for various HashiCorp copyright patterns
		if strings.Contains(cleanLine, "copyright") {
			// Pattern 1: "Copyright (c) HashiCorp" (with any additional text)
			if strings.Contains(cleanLine, "hashicorp") {
				// Further validate it's actually a copyright line and not just mentioning HashiCorp
				if strings.Contains(cleanLine, "(c)") ||
					strings.Contains(cleanLine, "©") ||
					(strings.Contains(cleanLine, "copyright") && strings.Index(cleanLine, "copyright") < strings.Index(cleanLine, "hashicorp")) {
					return true
				}
			}

			// Pattern 2: "Copyright HashiCorp, Inc." (with any additional text)
			if strings.Contains(cleanLine, "hashicorp, inc") {
				return true
			}

			// Pattern 3: "Copyright HashiCorp Inc." (without comma, with any additional text)
			if strings.Contains(cleanLine, "hashicorp inc") {
				return true
			}
		}
	}

	// Fallback: check for the old simple patterns in the entire content
	return strings.Contains(content, "copyright (c) hashicorp") ||
		strings.Contains(content, "hashicorp, inc.") ||
		strings.Contains(content, "hashicorp inc.")
}

// hasIBMHeader checks if the file contains an IBM copyright header (both old and new formats)
func hasIBMHeader(b []byte) bool {
	n := 1000
	if len(b) < 1000 {
		n = len(b)
	}
	content := bytes.ToLower(b[:n])
	// Check for both modern format "Copyright IBM Corp" and old format "Copyright (c) IBM Corp"
	return bytes.Contains(content, []byte("copyright ibm corp")) ||
		bytes.Contains(content, []byte("copyright (c) ibm corp"))
}

// Global variables to store the target license data for comparison
var targetLicenseData *LicenseData

// setTargetLicenseData sets the target license data for comparison
func setTargetLicenseData(data LicenseData) {
	targetLicenseData = &data
}

// hasIBMHeaderNeedingUpdate checks if the file has an IBM header that needs updating
// (different year range or SPDX license identifier)
func hasIBMHeaderNeedingUpdate(b []byte) bool {
	if !hasIBMHeader(b) {
		return false
	}

	// If we don't have target license data, assume no update needed
	if targetLicenseData == nil {
		return false
	}

	n := 1000
	if len(b) < 1000 {
		n = len(b)
	}
	content := string(b[:n])
	lines := strings.Split(content, "\n")

	var currentYear, currentSPDX string

	// Extract current year and SPDX information
	for _, line := range lines {
		lowerLine := strings.ToLower(strings.TrimSpace(line))

		// Look for copyright line with IBM Corp (both old and new formats)
		if strings.Contains(lowerLine, "copyright ibm corp") || strings.Contains(lowerLine, "copyright (c) ibm corp") {
			// Extract year information from the line
			// Pattern: "Copyright IBM Corp. YEAR" or "Copyright IBM Corp. YEAR1, YEAR2"
			// Pattern: "Copyright (c) IBM Corp. YEAR" or "Copyright (c) IBM Corp. YEAR1, YEAR2"
			parts := strings.Fields(line)
			for i, part := range parts {
				if strings.ToLower(part) == "corp." && i+1 < len(parts) {
					currentYear = strings.TrimSuffix(parts[i+1], ",")
					if i+2 < len(parts) && strings.Contains(parts[i+2], "20") {
						currentYear += ", " + parts[i+2]
					}
					break
				}
			}
		}

		// Look for SPDX license identifier
		if strings.Contains(lowerLine, "spdx-license-identifier") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				currentSPDX = strings.TrimSpace(parts[1])
			}
		}
	}

	// Compare with target data
	targetYear := targetLicenseData.Year
	targetSPDX := targetLicenseData.SPDXID

	// Check if year needs updating
	if targetYear != "" && currentYear != targetYear {
		return true
	}

	// Check if SPDX needs updating
	if targetSPDX != "" && currentSPDX != targetSPDX {
		return true
	}

	return false
}

// buildSmartYearRange builds a year range from explicit year flags while preserving existing years
// Supports selective year updates: if only year1 or year2 is provided, it merges with existing years
func buildSmartYearRange(targetData LicenseData, existingContent []byte) string {
	targetYear := targetData.Year
	if targetYear == "" {
		return ""
	}

	// Check if both years were explicitly provided (even if same)
	if strings.HasPrefix(targetYear, "EXPLICIT_BOTH:") {
		// Extract the year and return it as single year (overriding any existing years)
		year := strings.TrimPrefix(targetYear, "EXPLICIT_BOTH:")
		return year
	}

	// Extract existing years from the file content
	existingYear1, existingYear2 := extractExistingYears(existingContent)

	// Handle explicit year1 only update
	if strings.HasPrefix(targetYear, "YEAR1_ONLY:") {
		newYear1 := strings.TrimPrefix(targetYear, "YEAR1_ONLY:")
		if existingYear2 != "" {
			// File has existing year2, create range
			if newYear1 == existingYear2 {
				return newYear1 // Same year, return single year
			}
			// Ensure year1 <= year2
			if newYear1 > existingYear2 {
				return existingYear2 + ", " + newYear1
			}
			return newYear1 + ", " + existingYear2
		} else if existingYear1 != "" {
			// File has single existing year, replace it
			return newYear1
		} else {
			// No existing years, use provided year
			return newYear1
		}
	}

	// Handle explicit year2 only update
	if strings.HasPrefix(targetYear, "YEAR2_ONLY:") {
		newYear2 := strings.TrimPrefix(targetYear, "YEAR2_ONLY:")
		if existingYear1 != "" {
			// File has existing year1, create range
			if existingYear1 == newYear2 {
				return newYear2 // Same year, return single year
			}
			// Ensure year1 <= year2
			if existingYear1 > newYear2 {
				return newYear2 + ", " + existingYear1
			}
			return existingYear1 + ", " + newYear2
		} else {
			// No existing year1, use provided year as single year
			return newYear2
		}
	}

	// Handle regular year range (fallback for backward compatibility)
	targetParts := strings.Split(targetYear, ", ")
	var newYear1, newYear2 string

	if len(targetParts) == 1 {
		// Only one year provided - could be year1 only, year2 only, or both same
		singleYear := strings.TrimSpace(targetParts[0])

		// Check if we have existing years to determine if this is selective update
		if existingYear1 != "" && existingYear2 != "" {
			// File has existing year range - determine if this is year1 or year2 update
			// Heuristic: if provided year is <= existing year1, it's updating year1
			// if provided year > existing year1, it's updating year2
			if singleYear <= existingYear1 {
				// Updating year1
				newYear1 = singleYear
				newYear2 = existingYear2
			} else {
				// Updating year2
				newYear1 = existingYear1
				newYear2 = singleYear
			}
		} else if existingYear1 != "" {
			// File has single existing year - this could be extending to a range
			if singleYear == existingYear1 {
				// Same year, no change needed
				return singleYear
			} else if singleYear < existingYear1 {
				// New start year
				newYear1 = singleYear
				newYear2 = existingYear1
			} else {
				// New end year
				newYear1 = existingYear1
				newYear2 = singleYear
			}
		} else {
			// No existing years, use provided year
			return singleYear
		}
	} else if len(targetParts) == 2 {
		// Two years provided - use them directly
		newYear1 = strings.TrimSpace(targetParts[0])
		newYear2 = strings.TrimSpace(targetParts[1])
	} else {
		// Invalid format, return as-is
		return targetYear
	}

	// If year1 == year2, return only one year
	if newYear1 == newYear2 {
		return newYear1
	}

	// Ensure year1 <= year2
	if newYear1 > newYear2 {
		newYear1, newYear2 = newYear2, newYear1
	}

	// Return year range
	return newYear1 + ", " + newYear2
}

// extractExistingYears extracts existing copyright years from file content
func extractExistingYears(content []byte) (year1, year2 string) {
	// Look for copyright patterns in the first 500 characters (header area)
	n := 500
	if len(content) < 500 {
		n = len(content)
	}

	headerContent := string(content[:n])

	// Patterns to match copyright years
	patterns := []string{
		`Copyright\s+(?:IBM Corp\.|HashiCorp,?\s+Inc\.?)\s+(\d{4}),?\s*(\d{4})`,         // Range: "Copyright IBM Corp. 2020, 2025"
		`Copyright\s+(?:IBM Corp\.|HashiCorp,?\s+Inc\.?)\s+(\d{4})`,                     // Single: "Copyright IBM Corp. 2020"
		`Copyright\s+\(c\)\s+(?:IBM Corp\.|HashiCorp,?\s+Inc\.?)\s+(\d{4}),?\s*(\d{4})`, // Range with (c)
		`Copyright\s+\(c\)\s+(?:IBM Corp\.|HashiCorp,?\s+Inc\.?)\s+(\d{4})`,             // Single with (c)
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(headerContent)

		if len(matches) >= 2 {
			year1 = matches[1]
			if len(matches) >= 3 && matches[2] != "" {
				year2 = matches[2]
			} else {
				year2 = year1 // Single year case
			}
			return
		}
	}

	return "", ""
}

// updateLicense adds a license header to a file or replaces an existing HashiCorp/IBM header
func updateLicense(path string, fmode os.FileMode, tmpl *template.Template, data LicenseData) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	// Skip generated files
	if isGenerated(b) {
		return false, nil
	}

	// For any IBM or HashiCorp header, always use the year flags provided
	var finalData = data
	if hasHashiCorpHeader(b) || hasIBMHeader(b) {
		smartYear := buildSmartYearRange(data, b)
		finalData.Year = smartYear
	}

	// If file has a HashiCorp header or IBM header, do targeted replacement
	if hasHashiCorpHeader(b) || hasIBMHeader(b) {
		updatedContent, modified, err := replaceHeaderLines(b, path, finalData)
		if err != nil {
			return false, err
		}
		if modified {
			return true, os.WriteFile(path, updatedContent, fmode)
		}
		return false, nil
	} else if hasCopyrightButNotHashiCorpOrIBM(b) {
		// If it has copyright from other companies that we shouldn't touch, don't modify
		return false, nil
	}

	// File has no copyright header - do NOT add one (removed this feature)
	// Update command should only modify existing HashiCorp or IBM headers
	return false, nil
}

// replaceHeaderLines does targeted replacement of copyright lines only, preserving SPDX and other content
func replaceHeaderLines(content []byte, path string, data LicenseData) ([]byte, bool, error) {
	lines := bytes.Split(content, []byte("\n"))
	if len(lines) == 0 {
		return content, false, nil
	}

	// Determine comment style based on file extension
	base := strings.ToLower(filepath.Base(path))
	ext := fileExtension(base)

	modified := false
	var processedLines [][]byte
	expectedCopyright := fmt.Sprintf("Copyright %s %s", data.Holder, data.Year)
	addedCopyright := false

	for i, line := range lines {
		lineStr := string(line)
		lowerLineStr := strings.ToLower(lineStr)

		// Skip hashbang line
		if i == 0 && strings.HasPrefix(lineStr, "#!") {
			processedLines = append(processedLines, line)
			continue
		}

		// Check if this line contains a copyright statement
		if strings.Contains(lowerLineStr, "copyright") {
			// Only process HashiCorp and IBM Corp headers
			if strings.Contains(lowerLineStr, "hashicorp") || strings.Contains(lowerLineStr, "ibm corp") {
				newLine := surgicallyReplaceCopyright(lineStr, ext, data)
				if newLine != lineStr {
					// Check if we already added this exact copyright line
					if !addedCopyright && strings.Contains(newLine, expectedCopyright) {
						processedLines = append(processedLines, []byte(newLine))
						addedCopyright = true
						modified = true
					} else if !strings.Contains(newLine, expectedCopyright) {
						// If the surgical replacement didn't produce expected result, still add it
						processedLines = append(processedLines, []byte(newLine))
						modified = true
					}
					// Skip duplicate IBM copyright lines
					continue
				}
			} else {
				// Preserve copyright lines from other companies (non-HashiCorp, non-IBM)
				processedLines = append(processedLines, line)
				continue
			}
		}

		// DO NOT MODIFY SPDX LICENSE IDENTIFIERS - preserve them as-is
		// Keep all SPDX lines unchanged regardless of what they contain

		// Keep all other lines unchanged
		processedLines = append(processedLines, line)
	}

	if modified {
		return bytes.Join(processedLines, []byte("\n")), true, nil
	}
	return content, false, nil
}

// surgicallyReplaceCopyright replaces copyright statement while preserving any additional text on the same line
func surgicallyReplaceCopyright(line, ext string, data LicenseData) string {
	newCopyright := fmt.Sprintf("Copyright %s %s", data.Holder, data.Year)

	// Create regex patterns for different copyright formats - updated to match actual HashiCorp formats
	patterns := []string{
		// Pattern 1: "Copyright (c) 2020 HashiCorp, Inc."
		`(?i)(.*?)(copyright\s*\(c\)\s*\d{4}(?:[-,]\s*\d{4})*\s+hashicorp,?\s+inc\.?)(.*?)`,
		// Pattern 2: "Copyright (c) Hashicorp Inc. 2020"
		`(?i)(.*?)(copyright\s*\(c\)\s*(?:hashicorp,?\s+inc\.?)(?:\s+\d{4}(?:[-,]\s*\d{4})*)?)(.*?)`,
		// Pattern 3: "Copyright Hashicorp Inc. 2020"
		`(?i)(.*?)(copyright\s+(?:hashicorp,?\s+inc\.?)(?:\s+\d{4}(?:[-,]\s*\d{4})*)?)(.*?)`,
		// Pattern 4: Handle format like "Copyright (c) 2019 Hashicorp Inc."
		`(?i)(.*?)(copyright\s*\(c\)\s*\d{4}(?:[-,]\s*\d{4})*\s+hashicorp,?\s+inc\.?)(.*?)`,
		// Pattern 5: Handle IBM headers with years (modern format)
		`(?i)(.*?)(copyright\s+ibm\s+corp\.?\s+\d{4}(?:[-,]\s*\d{4})*)(.*?)`,
		// Pattern 6: Handle old IBM headers with (c) format - "Copyright (c) IBM Corp. 2020, 2025"
		`(?i)(.*?)(copyright\s*\(c\)\s*ibm\s+corp\.?(?:\s+\d{4}(?:[-,]\s*\d{4})*)?)(.*?)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(line) {
			return re.ReplaceAllString(line, "${1}"+newCopyright+"${3}")
		}
	}

	return line
}
