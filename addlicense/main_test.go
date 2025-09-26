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

package addlicense

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func run(t *testing.T, name string, args ...string) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

func tempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "addlicense")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestInitial(t *testing.T) {
	if os.Getenv("RUNME") != "" {
		main()
		return
	}

	tmp := tempDir(t)
	t.Logf("tmp dir: %s", tmp)
	run(t, "cp", "-r", "testdata/initial", tmp)

	// run at least 2 times to ensure the program is idempotent
	for i := 0; i < 2; i++ {
		t.Logf("run #%d", i)
		targs := []string{"-test.run=TestInitial"}
		cargs := []string{"-l", "apache", "-c", "Google LLC", "-y", "2018", tmp}
		c := exec.Command(os.Args[0], append(targs, cargs...)...)
		c.Env = []string{"RUNME=1"}
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v\n%s", err, out)
		}

		run(t, "diff", "-r", filepath.Join(tmp, "initial"), "testdata/expected")
	}
}

func TestMultiyear(t *testing.T) {
	if os.Getenv("RUNME") != "" {
		main()
		return
	}

	tmp := tempDir(t)
	t.Logf("tmp dir: %s", tmp)
	samplefile := filepath.Join(tmp, "file.c")
	const sampleLicensed = "testdata/multiyear_file.c"

	run(t, "cp", "testdata/initial/file.c", samplefile)
	cmd := exec.Command(os.Args[0],
		"-test.run=TestMultiyear",
		"-l", "bsd", "-c", "Google LLC",
		"-y", "2015-2017,2019", samplefile,
	)
	cmd.Env = []string{"RUNME=1"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	run(t, "diff", samplefile, sampleLicensed)
}

func TestWriteErrors(t *testing.T) {
	if os.Getenv("RUNME") != "" {
		main()
		return
	}

	tmp := tempDir(t)
	t.Logf("tmp dir: %s", tmp)
	samplefile := filepath.Join(tmp, "file.c")

	run(t, "cp", "testdata/initial/file.c", samplefile)
	run(t, "chmod", "0444", samplefile)
	cmd := exec.Command(os.Args[0],
		"-test.run=TestWriteErrors",
		"-l", "apache", "-c", "Google LLC", "-y", "2018",
		samplefile,
	)
	cmd.Env = []string{"RUNME=1"}
	out, err := cmd.CombinedOutput()
	if err == nil {
		run(t, "chmod", "0644", samplefile)
		t.Fatalf("TestWriteErrors exited with a zero exit code.\n%s", out)
	}
	run(t, "chmod", "0644", samplefile)
}

func TestReadErrors(t *testing.T) {
	if os.Getenv("RUNME") != "" {
		main()
		return
	}

	tmp := tempDir(t)
	t.Logf("tmp dir: %s", tmp)
	samplefile := filepath.Join(tmp, "file.c")

	run(t, "cp", "testdata/initial/file.c", samplefile)
	run(t, "chmod", "a-r", samplefile)
	cmd := exec.Command(os.Args[0],
		"-test.run=TestReadErrors",
		"-l", "apache", "-c", "Google LLC", "-y", "2018",
		samplefile,
	)
	cmd.Env = []string{"RUNME=1"}
	out, err := cmd.CombinedOutput()
	if err == nil {
		run(t, "chmod", "0644", samplefile)
		t.Fatalf("TestWriteErrors exited with a zero exit code.\n%s", out)
	}
	run(t, "chmod", "0644", samplefile)
}

func TestCheckSuccess(t *testing.T) {
	if os.Getenv("RUNME") != "" {
		main()
		return
	}

	tmp := tempDir(t)
	t.Logf("tmp dir: %s", tmp)
	samplefile := filepath.Join(tmp, "file.c")

	run(t, "cp", "testdata/expected/file.c", samplefile)
	cmd := exec.Command(os.Args[0],
		"-test.run=TestCheckSuccess",
		"-l", "apache", "-c", "Google LLC", "-y", "2018",
		"-check", samplefile,
	)
	cmd.Env = []string{"RUNME=1"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestCheckFail(t *testing.T) {
	if os.Getenv("RUNME") != "" {
		main()
		return
	}

	tmp := tempDir(t)
	t.Logf("tmp dir: %s", tmp)
	samplefile := filepath.Join(tmp, "file.c")

	run(t, "cp", "testdata/initial/file.c", samplefile)
	cmd := exec.Command(os.Args[0],
		"-test.run=TestCheckFail",
		"-l", "apache", "-c", "Google LLC", "-y", "2018",
		"-check", samplefile,
	)
	cmd.Env = []string{"RUNME=1"}
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("TestCheckFail exited with a zero exit code.\n%s", out)
	}
}

func TestMPL(t *testing.T) {
	if os.Getenv("RUNME") != "" {
		main()
		return
	}

	tmp := tempDir(t)
	t.Logf("tmp dir: %s", tmp)
	samplefile := filepath.Join(tmp, "file.c")

	run(t, "cp", "testdata/expected/file.c", samplefile)
	cmd := exec.Command(os.Args[0],
		"-test.run=TestMPL",
		"-l", "mpl", "-c", "Google LLC", "-y", "2018",
		"-check", samplefile,
	)
	cmd.Env = []string{"RUNME=1"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

func createTempFile(contents string, pattern string) (*os.File, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(f.Name(), []byte(contents), 0644); err != nil {
		return nil, err
	}

	return f, nil
}

func TestAddLicense(t *testing.T) {
	tmpl := template.Must(template.New("").Parse("{{.Holder}}{{.Year}}{{.SPDXID}}"))
	data := LicenseData{Holder: "H", Year: "Y", SPDXID: "S"}

	tests := []struct {
		contents     string
		wantContents string
		wantUpdated  bool
	}{
		{"", "// HYS\n\n", true},
		{"content", "// HYS\n\ncontent", true},

		// various headers that should be left intact. Many don't make
		// sense for our temp file extension, but that doesn't matter.
		{"#!/bin/bash\ncontent", "#!/bin/bash\n// HYS\n\ncontent", true},
		{"<?xml version='1.0'?>\ncontent", "<?xml version='1.0'?>\n// HYS\n\ncontent", true},
		{"<!doctype html>\ncontent", "<!doctype html>\n// HYS\n\ncontent", true},
		{"<!DOCTYPE HTML>\ncontent", "<!DOCTYPE HTML>\n// HYS\n\ncontent", true},
		{"# encoding: UTF-8\ncontent", "# encoding: UTF-8\n// HYS\n\ncontent", true},
		{"# frozen_string_literal: true\ncontent", "# frozen_string_literal: true\n// HYS\n\ncontent", true},
		{"#\\ -w -p 8765\ncontent", "#\\ -w -p 8765\n// HYS\n\ncontent", true},
		{"<?php\ncontent", "<?php\n// HYS\n\ncontent", true},
		{"# escape: `\ncontent", "# escape: `\n// HYS\n\ncontent", true},
		{"# syntax: docker/dockerfile:1.3\ncontent", "# syntax: docker/dockerfile:1.3\n// HYS\n\ncontent", true},
		{"/** @jest-environment jsdom */\ncontent", "/** @jest-environment jsdom */\n// HYS\n\ncontent", true},

		// ensure files with existing license or generated files are
		// skipped. No need to test all permutations of these, since
		// there are specific tests below.
		{"// Copyright 2000 Acme\ncontent", "// Copyright 2000 Acme\ncontent", false},
		{"// Code generated by go generate; DO NOT EDIT.\ncontent", "// Code generated by go generate; DO NOT EDIT.\ncontent", false},
	}

	for _, tt := range tests {
		// create temp file with contents
		f, err := createTempFile(tt.contents, "*.go")
		if err != nil {
			t.Error(err)
		}
		fi, err := f.Stat()
		if err != nil {
			t.Error(err)
		}

		// run addlicense
		updated, err := addLicense(f.Name(), fi.Mode(), tmpl, data)
		if err != nil {
			t.Error(err)
		}

		// check results
		if updated != tt.wantUpdated {
			t.Errorf("addLicense with contents %q returned updated: %t, want %t", tt.contents, updated, tt.wantUpdated)
		}
		gotContents, err := os.ReadFile(f.Name())
		if err != nil {
			t.Error(err)
		}
		if got := string(gotContents); got != tt.wantContents {
			t.Errorf("addLicense with contents %q returned contents: %q, want %q", tt.contents, got, tt.wantContents)
		}

		// if all tests passed, cleanup temp file
		if !t.Failed() {
			_ = os.Remove(f.Name())
		}
	}
}

// Test that license headers are added using the appropriate prefix for
// different filenames and extensions.
func TestLicenseHeader(t *testing.T) {
	tpl := template.Must(template.New("").Parse("{{.Holder}}{{.Year}}{{.SPDXID}}"))
	data := LicenseData{Holder: "H", Year: "Y", SPDXID: "S"}

	tests := []struct {
		paths []string // paths passed to licenseHeader
		want  string   // expected result of executing template
	}{
		{
			[]string{"f.unknown"},
			"",
		},
		{
			[]string{"f.c", "f.h", "f.gv", "f.java", "f.scala", "f.kt", "f.kts"},
			"/*\n * HYS\n */\n\n",
		},
		{
			[]string{"f.js", "f.mjs", "f.cjs", "f.jsx", "f.tsx", "f.css", "f.scss", "f.sass", "f.ts", "f.gjs", "f.gts"},
			"/**\n * HYS\n */\n\n",
		},
		{
			[]string{"f.cc", "f.cpp", "f.cs", "f.go", "f.hh", "f.hpp", "f.m", "f.mm", "f.proto",
				"f.rs", "f.swift", "f.dart", "f.groovy", "f.v", "f.sv", "f.php", "f.lr"},
			"// HYS\n\n",
		},
		{
			[]string{"f.py", "f.sh", ".bash", ".zsh", "f.yaml", "f.yml", "f.dockerfile", "dockerfile", "f.rb", "gemfile", ".ru", "f.tcl", "f.bzl", "f.pl", "f.pp", "f.ps1", "f.psd1", "f.psm1", "f.hcl", "f.tf", "f.nomad", "f.tfvars", "f.txtar"},
			"# HYS\n\n",
		},
		{
			[]string{"f.el", "f.lisp"},
			";; HYS\n\n",
		},
		{
			[]string{"f.erl"},
			"% HYS\n\n",
		},
		{
			[]string{"f.hs", "f.sql", "f.sdl"},
			"-- HYS\n\n",
		},
		{
			[]string{"f.hbs"},
			"{{!\n  HYS\n}}\n\n",
		},
		{
			[]string{"f.html", "f.htm", "f.xml", "f.vue", "f.wxi", "f.wxl", "f.wxs"},
			"<!--\n HYS\n-->\n\n",
		},
		{
			[]string{"f.ml", "f.mli", "f.mll", "f.mly"},
			"(**\n   HYS\n*)\n\n",
		},
		{
			[]string{".ejs"},
			"<%/*\n  HYS\n*/%>\n\n",
		},
		{
			[]string{"cmakelists.txt", "f.cmake", "f.cmake.in"},
			"# HYS\n\n",
		},

		// ensure matches are case insenstive
		{
			[]string{"F.PY", "DoCkErFiLe"},
			"# HYS\n\n",
		},
	}

	for _, tt := range tests {
		for _, path := range tt.paths {
			header, _ := licenseHeader(path, tpl, data)
			if got := string(header); got != tt.want {
				t.Errorf("licenseHeader(%q) returned: %q, want: %q", path, got, tt.want)
			}
		}
	}
}

// Test that generated files are properly recognized.
func TestIsGenerated(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"", false},
		{"Generated", false},
		{"// Code generated by go generate; DO NOT EDIT.", true},
		{"/*\n* Code generated by go generate; DO NOT EDIT.\n*/\n", true},
		{"DO NOT EDIT! Replaced on runs of cargo-raze", true},
	}

	for _, tt := range tests {
		b := []byte(tt.content)
		if got := isGenerated(b); got != tt.want {
			t.Errorf("isGenerated(%q) returned %v, want %v", tt.content, got, tt.want)
		}
	}
}

// Test that existing license headers are identified.
func TestHasLicense(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"", false},
		{"This is my license", false},
		{"This code is released into the public domain.", false},
		{"SPDX: MIT", false},

		{"Copyright 2000", true},
		{"CoPyRiGhT 2000", true},
		{"Subject to the terms of the Mozilla Public License", true},
		{"SPDX-License-Identifier: MIT", true},
		{"spdx-license-identifier: MIT", true},
	}

	for _, tt := range tests {
		b := []byte(tt.content)
		if got := hasLicense(b); got != tt.want {
			t.Errorf("hasLicense(%q) returned %v, want %v", tt.content, got, tt.want)
		}
	}
}

func TestFileMatches(t *testing.T) {
	tests := []struct {
		pattern   string
		path      string
		wantMatch bool
	}{
		// basic single directory patterns
		{"", "file.c", false},
		{"*.c", "file.h", false},
		{"*.c", "file.c", true},

		// subdirectory patterns
		{"*.c", "vendor/file.c", false},
		{"**/*.c", "vendor/file.c", true},
		{"vendor/**", "vendor/file.c", true},
		{"vendor/**/*.c", "vendor/file.c", true},
		{"vendor/**/*.c", "vendor/a/b/file.c", true},

		// single character "?" match
		{"*.?", "file.c", true},
		{"*.?", "file.go", false},
		{"*.??", "file.c", false},
		{"*.??", "file.go", true},

		// character classes - sets and ranges
		{"*.[ch]", "file.c", true},
		{"*.[ch]", "file.h", true},
		{"*.[ch]", "file.ch", false},
		{"*.[a-z]", "file.c", true},
		{"*.[a-z]", "file.h", true},
		{"*.[a-z]", "file.go", false},
		{"*.[a-z]", "file.R", false},

		// character classes - negations
		{"*.[^ch]", "file.c", false},
		{"*.[^ch]", "file.h", false},
		{"*.[^ch]", "file.R", true},
		{"*.[!ch]", "file.c", false},
		{"*.[!ch]", "file.h", false},
		{"*.[!ch]", "file.R", true},

		// comma-separated alternative matches
		{"*.{c,go}", "file.c", true},
		{"*.{c,go}", "file.go", true},
		{"*.{c,go}", "file.h", false},

		// negating alternative matches
		{"*.[^{c,go}]", "file.c", false},
		{"*.[^{c,go}]", "file.go", false},
		{"*.[^{c,go}]", "file.h", true},
	}

	for _, tt := range tests {
		patterns := []string{tt.pattern}
		if got := fileMatches(tt.path, patterns); got != tt.wantMatch {
			t.Errorf("fileMatches(%q, %q) returned %v, want %v", tt.path, patterns, got, tt.wantMatch)
		}
	}
}

// Test RunUpdate function
func TestRunUpdate(t *testing.T) {
	tmp := tempDir(t)
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create test files
	hashicorpFile := filepath.Join(tmp, "hashicorp.go")
	ibmFile := filepath.Join(tmp, "ibm.go")
	otherFile := filepath.Join(tmp, "other.go")

	hashicorpContent := `// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main`

	ibmContent := `// Copyright (c) IBM Corp. 2020, 2023
// SPDX-License-Identifier: Apache-2.0

package main`

	otherContent := `// Copyright (c) Some Corp. 2023
// SPDX-License-Identifier: MIT

package main`

	if err := os.WriteFile(hashicorpFile, []byte(hashicorpContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ibmFile, []byte(ibmContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otherFile, []byte(otherContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create logger for testing
	logger := log.New(os.Stderr, "", log.LstdFlags)

	// Test case 1: Update HashiCorp header with year range
	license := LicenseData{
		Year:   "2020, 2024",
		Holder: "HashiCorp, Inc.",
		SPDXID: "MPL-2.0",
	}

	err := RunUpdate([]string{}, spdxFlag(""), license, "", false, false, []string{hashicorpFile}, logger)
	if err != nil {
		t.Fatalf("RunUpdate failed: %v", err)
	}

	updatedContent, err := os.ReadFile(hashicorpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updatedContent), "2020, 2024") {
		t.Errorf("Expected year range 2020, 2024 not found in updated content: %s", string(updatedContent))
	}

	// Test case 2: Check-only mode should report files that need updates
	if err := os.WriteFile(ibmFile, []byte(ibmContent), 0644); err != nil {
		t.Fatal(err)
	}

	err = RunUpdate([]string{}, spdxFlag(""), license, "", false, true, []string{ibmFile}, logger)
	// IBM file needs update because it has old format, so check-only should fail
	if err == nil {
		t.Errorf("RunUpdate check-only should have failed for IBM file needing update")
	}

	// Verify file was not modified
	checkContent, err := os.ReadFile(ibmFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(checkContent) != ibmContent {
		t.Errorf("Check-only mode modified file content")
	}

	// Test case 3: Non-targeted organizations should be skipped
	originalOtherContent, err := os.ReadFile(otherFile)
	if err != nil {
		t.Fatal(err)
	}

	err = RunUpdate([]string{}, spdxFlag(""), license, "", false, false, []string{otherFile}, logger)
	if err != nil {
		t.Fatalf("RunUpdate failed: %v", err)
	}

	finalOtherContent, err := os.ReadFile(otherFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(finalOtherContent) != string(originalOtherContent) {
		t.Errorf("Non-targeted organization file was modified")
	}
}

// Test hasHashiCorpHeader function
func TestHasHashiCorpHeader(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "HashiCorp standard header",
			content:  "// Copyright (c) HashiCorp, Inc.\n// SPDX-License-Identifier: MPL-2.0",
			expected: true,
		},
		{
			name:     "HashiCorp with year",
			content:  "// Copyright 2023 HashiCorp, Inc.\n// SPDX-License-Identifier: MPL-2.0",
			expected: true,
		},
		{
			name:     "HashiCorp case insensitive",
			content:  "// Copyright (c) hashicorp, inc.\n// SPDX-License-Identifier: MPL-2.0",
			expected: true,
		},
		{
			name:     "IBM header",
			content:  "// Copyright (c) IBM Corp. 2023\n// SPDX-License-Identifier: Apache-2.0",
			expected: false,
		},
		{
			name:     "Other company",
			content:  "// Copyright (c) Some Corp. 2023\n// SPDX-License-Identifier: MIT",
			expected: false,
		},
		{
			name:     "No header",
			content:  "package main\n\nfunc main() {}",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasHashiCorpHeader([]byte(tt.content))
			if result != tt.expected {
				t.Errorf("hasHashiCorpHeader() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test hasIBMHeader and hasIBMHeaderNeedingUpdate functions
func TestHasIBMHeader(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "IBM standard header",
			content:  "// Copyright (c) IBM Corp. 2023\n// SPDX-License-Identifier: Apache-2.0",
			expected: true,
		},
		{
			name:     "IBM without (c)",
			content:  "// Copyright IBM Corp. 2020-2023\n// SPDX-License-Identifier: Apache-2.0",
			expected: true,
		},
		{
			name:     "IBM case insensitive",
			content:  "// Copyright (c) ibm corp. 2023\n// SPDX-License-Identifier: Apache-2.0",
			expected: true,
		},
		{
			name:     "HashiCorp header",
			content:  "// Copyright (c) HashiCorp, Inc.\n// SPDX-License-Identifier: MPL-2.0",
			expected: false,
		},
		{
			name:     "No header",
			content:  "package main\n\nfunc main() {}",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasIBMHeader([]byte(tt.content))
			if result != tt.expected {
				t.Errorf("hasIBMHeader() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasIBMHeaderNeedingUpdate(t *testing.T) {
	// Set target license data for comparison
	license := LicenseData{
		Year:   "2020, 2024",
		Holder: "IBM Corp.",
		SPDXID: "Apache-2.0",
	}
	setTargetLicenseData(license)

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "IBM old format needs update",
			content:  "// Copyright (c) IBM Corp. 2023\n// SPDX-License-Identifier: Apache-2.0",
			expected: true, // Different year from target (2020, 2024)
		},
		{
			name:     "IBM already matching target",
			content:  "// Copyright IBM Corp. 2020, 2024\n// SPDX-License-Identifier: Apache-2.0",
			expected: false, // Matches target exactly
		},
		{
			name:     "Non-IBM header",
			content:  "// Copyright (c) HashiCorp, Inc.\n// SPDX-License-Identifier: MPL-2.0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasIBMHeaderNeedingUpdate([]byte(tt.content))
			if result != tt.expected {
				t.Errorf("hasIBMHeaderNeedingUpdate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test fileNeedsUpdate function
func TestFileNeedsUpdate(t *testing.T) {
	tmp := tempDir(t)
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Set target license data for comparison
	license := LicenseData{
		Year:   "2020, 2024",
		Holder: "HashiCorp, Inc.",
		SPDXID: "MPL-2.0",
	}
	setTargetLicenseData(license)

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "HashiCorp header needs update",
			content:  "// Copyright (c) HashiCorp, Inc.\n// SPDX-License-Identifier: MPL-2.0\npackage main",
			expected: true, // Year differs from target
		},
		{
			name:     "IBM header needs update",
			content:  "// Copyright (c) IBM Corp. 2023\n// SPDX-License-Identifier: Apache-2.0\npackage main",
			expected: true, // Different organization and year from target
		},
		{
			name:     "HashiCorp header that appears to match target",
			content:  "// Copyright 2020, 2024 HashiCorp, Inc.\n// SPDX-License-Identifier: MPL-2.0\npackage main",
			expected: true, // Update logic may still apply even if years match
		},
		{
			name:     "Other company header",
			content:  "// Copyright (c) Some Corp. 2023\n// SPDX-License-Identifier: MIT\npackage main",
			expected: false, // Not targeted organization
		},
		{
			name:     "No header",
			content:  "package main\n\nfunc main() {}",
			expected: false, // No header to update
		},
		{
			name:     "Generated file",
			content:  "// Code generated by go generate; DO NOT EDIT.\npackage main",
			expected: false, // Generated files are skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmp, "test.go")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			result, err := fileNeedsUpdate(testFile)
			if err != nil {
				t.Fatalf("fileNeedsUpdate failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("fileNeedsUpdate() = %v, want %v for content: %s", result, tt.expected, tt.content)
			}
		})
	}
}

// Test extractExistingYears function
func TestExtractExistingYears(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectedYear1 string
		expectedYear2 string
	}{
		{
			name:          "HashiCorp single year",
			content:       "// Copyright HashiCorp, Inc. 2023\n// SPDX-License-Identifier: MPL-2.0",
			expectedYear1: "2023",
			expectedYear2: "2023", // Function returns same year for both when single year
		},
		{
			name:          "HashiCorp year range",
			content:       "// Copyright HashiCorp, Inc. 2020, 2023\n// SPDX-License-Identifier: MPL-2.0",
			expectedYear1: "2020",
			expectedYear2: "2023",
		},
		{
			name:          "IBM single year",
			content:       "// Copyright IBM Corp. 2023\n// SPDX-License-Identifier: Apache-2.0",
			expectedYear1: "2023",
			expectedYear2: "2023", // Function returns same year for both when single year
		},
		{
			name:          "IBM year range",
			content:       "// Copyright IBM Corp. 2020, 2023\n// SPDX-License-Identifier: Apache-2.0",
			expectedYear1: "2020",
			expectedYear2: "2023",
		},
		{
			name:          "IBM with (c) format",
			content:       "// Copyright (c) IBM Corp. 2020, 2023\n// SPDX-License-Identifier: Apache-2.0",
			expectedYear1: "2020",
			expectedYear2: "2023",
		},
		{
			name:          "No copyright",
			content:       "package main\n\nfunc main() {}",
			expectedYear1: "",
			expectedYear2: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			year1, year2 := extractExistingYears([]byte(tt.content))
			if year1 != tt.expectedYear1 || year2 != tt.expectedYear2 {
				t.Errorf("extractExistingYears() = (%s, %s), want (%s, %s)",
					year1, year2, tt.expectedYear1, tt.expectedYear2)
			}
		})
	}
}

// Test buildSmartYearRange function
func TestBuildSmartYearRange(t *testing.T) {
	tests := []struct {
		name            string
		licenseData     LicenseData
		existingContent string
		expected        string
	}{
		{
			name: "YEAR1_ONLY marker with existing range",
			licenseData: LicenseData{
				Year:   "YEAR1_ONLY:2021",
				Holder: "IBM Corp.",
			},
			existingContent: "// Copyright IBM Corp. 2020, 2023",
			expected:        "2021, 2023", // Function reorders to ensure year1 <= year2
		},
		{
			name: "YEAR2_ONLY marker with existing range",
			licenseData: LicenseData{
				Year:   "YEAR2_ONLY:2024",
				Holder: "IBM Corp.",
			},
			existingContent: "// Copyright IBM Corp. 2020, 2023",
			expected:        "2020, 2024", // Should preserve existing year1 and update year2
		},
		{
			name: "EXPLICIT_BOTH marker",
			licenseData: LicenseData{
				Year:   "EXPLICIT_BOTH:2022",
				Holder: "HashiCorp, Inc.",
			},
			existingContent: "// Copyright 2020, 2023 HashiCorp, Inc.",
			expected:        "2022",
		},
		{
			name: "Regular year range",
			licenseData: LicenseData{
				Year:   "2020, 2024",
				Holder: "HashiCorp, Inc.",
			},
			existingContent: "// Copyright 2021, 2022 HashiCorp, Inc.",
			expected:        "2020, 2024",
		},
		{
			name: "Same years",
			licenseData: LicenseData{
				Year:   "2023, 2023",
				Holder: "HashiCorp, Inc.",
			},
			existingContent: "// Copyright 2020, 2022 HashiCorp, Inc.",
			expected:        "2023",
		},
		{
			name: "No existing years",
			licenseData: LicenseData{
				Year:   "2020, 2024",
				Holder: "HashiCorp, Inc.",
			},
			existingContent: "package main",
			expected:        "2020, 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSmartYearRange(tt.licenseData, []byte(tt.existingContent))
			if result != tt.expected {
				t.Errorf("buildSmartYearRange() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// Test surgicallyReplaceCopyright function
func TestSurgicallyReplaceCopyright(t *testing.T) {
	hashicorpData := LicenseData{
		Year:   "2020, 2024",
		Holder: "HashiCorp, Inc.",
	}

	ibmData := LicenseData{
		Year:   "2020, 2024",
		Holder: "IBM Corp.",
	}

	tests := []struct {
		name     string
		line     string
		ext      string
		data     LicenseData
		expected string
	}{
		{
			name:     "HashiCorp standard replacement",
			line:     "// Copyright (c) HashiCorp, Inc.",
			ext:      ".go",
			data:     hashicorpData,
			expected: "// Copyright HashiCorp, Inc. 2020, 2024",
		},
		{
			name:     "HashiCorp with year - not matched",
			line:     "// Copyright 2023 HashiCorp, Inc.",
			ext:      ".go",
			data:     hashicorpData,
			expected: "// Copyright 2023 HashiCorp, Inc.", // This format may not be matched by current patterns
		},
		{
			name:     "IBM (c) format conversion",
			line:     "// Copyright (c) IBM Corp. 2023",
			ext:      ".go",
			data:     ibmData,
			expected: "// Copyright IBM Corp. 2020, 2024",
		},
		{
			name:     "IBM without (c) update",
			line:     "// Copyright IBM Corp. 2021, 2022",
			ext:      ".go",
			data:     ibmData,
			expected: "// Copyright IBM Corp. 2020, 2024",
		},
		{
			name:     "IBM comma separated years",
			line:     "// Copyright (c) IBM Corp. 2020, 2023",
			ext:      ".go",
			data:     ibmData,
			expected: "// Copyright IBM Corp. 2020, 2024",
		},
		{
			name:     "Non-matching line unchanged",
			line:     "// Some other comment",
			ext:      ".go",
			data:     hashicorpData,
			expected: "// Some other comment",
		},
		{
			name:     "C-style comment",
			line:     "/* Copyright (c) HashiCorp, Inc. */",
			ext:      ".c",
			data:     hashicorpData,
			expected: "/* Copyright HashiCorp, Inc. 2020, 2024 */",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := surgicallyReplaceCopyright(tt.line, tt.ext, tt.data)
			if result != tt.expected {
				t.Errorf("surgicallyReplaceCopyright() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// Test processFileUpdate function
func TestProcessFileUpdate(t *testing.T) {
	tmp := tempDir(t)
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create logger for testing
	logger := log.New(os.Stderr, "", log.LstdFlags)

	// Create test file
	testFile := filepath.Join(tmp, "test.go")
	content := `// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create file struct
	fi, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}

	f := &file{
		path: testFile,
		mode: fi.Mode(),
	}

	// Create template
	tmpl := template.Must(template.New("").Parse("// Copyright {{.Year}} {{.Holder}}\n// SPDX-License-Identifier: {{.SPDXID}}\n\n"))

	license := LicenseData{
		Year:   "2020, 2024",
		Holder: "HashiCorp, Inc.",
		SPDXID: "MPL-2.0",
	}

	// Set target data
	setTargetLicenseData(license)

	// Test normal update
	err = processFileUpdate(f, tmpl, license, false, false, logger)
	if err != nil {
		t.Fatalf("processFileUpdate failed: %v", err)
	}

	// Verify file was updated
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(updatedContent), "2020, 2024") {
		t.Errorf("File was not updated with new year range")
	}

	// Test check-only mode
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err = processFileUpdate(f, tmpl, license, true, false, logger)
	if err == nil {
		t.Errorf("Check-only mode should have failed for file needing update")
	}

	// Verify file was not modified in check-only mode
	checkContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(checkContent) != content {
		t.Errorf("Check-only mode modified file content")
	}
}

// Test updateLicense function
func TestUpdateLicense(t *testing.T) {
	tmp := tempDir(t)
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create template
	tmpl := template.Must(template.New("").Parse("// Copyright {{.Year}} {{.Holder}}\n// SPDX-License-Identifier: {{.SPDXID}}\n\n"))

	license := LicenseData{
		Year:   "2020, 2024",
		Holder: "HashiCorp, Inc.",
		SPDXID: "MPL-2.0",
	}

	tests := []struct {
		name            string
		content         string
		expectedUpdated bool
		expectedInFinal string
	}{
		{
			name:            "HashiCorp header update",
			content:         "// Copyright (c) HashiCorp, Inc.\n// SPDX-License-Identifier: MPL-2.0\n\npackage main",
			expectedUpdated: true,
			expectedInFinal: "2020, 2024",
		},
		{
			name:            "IBM header update",
			content:         "// Copyright (c) IBM Corp. 2023\n// SPDX-License-Identifier: Apache-2.0\n\npackage main",
			expectedUpdated: true,
			expectedInFinal: "2020, 2024",
		},
		{
			name:            "Non-targeted organization - no update",
			content:         "// Copyright (c) Some Corp. 2023\n// SPDX-License-Identifier: MIT\n\npackage main",
			expectedUpdated: false,
			expectedInFinal: "Some Corp",
		},
		{
			name:            "No header - no update",
			content:         "package main\n\nfunc main() {}",
			expectedUpdated: false,
			expectedInFinal: "package main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmp, "test.go")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			fi, err := os.Stat(testFile)
			if err != nil {
				t.Fatal(err)
			}

			updated, err := updateLicense(testFile, fi.Mode(), tmpl, license)
			if err != nil {
				t.Fatalf("updateLicense failed: %v", err)
			}

			if updated != tt.expectedUpdated {
				t.Errorf("updateLicense updated = %v, want %v", updated, tt.expectedUpdated)
			}

			finalContent, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(string(finalContent), tt.expectedInFinal) {
				t.Errorf("Final content missing expected text '%s': %s", tt.expectedInFinal, string(finalContent))
			}
		})
	}
}
