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

func TestUpdateLicenseHolder(t *testing.T) {
	data := LicenseData{Holder: "IBM Corp.", Year: "2023, 2026", SPDXID: "MPL-2.0"}

	tests := []struct {
		name        string
		content     string
		wantContent string
		wantUpdated bool
	}{
		{
			name:        "Update HashiCorp, Inc. with year after",
			content:     "// Copyright (c) HashiCorp, Inc. 2023\n\npackage main",
			wantContent: "// Copyright IBM Corp. 2023, 2026\n\npackage main",
			wantUpdated: true,
		},
		{
			name:        "Update HashiCorp, Inc. with year before",
			content:     "// Copyright 2023 HashiCorp, Inc.\n\npackage main",
			wantContent: "// Copyright IBM Corp. 2023, 2026\n\npackage main",
			wantUpdated: true,
		},
		{
			name:        "Update HashiCorp without Inc",
			content:     "// Copyright HashiCorp 2023\n\npackage main",
			wantContent: "// Copyright IBM Corp. 2023, 2026\n\npackage main",
			wantUpdated: true,
		},
		{
			name:        "Update HashiCorp without (c) symbol",
			content:     "// Copyright HashiCorp, Inc. 2023\n\npackage main",
			wantContent: "// Copyright IBM Corp. 2023, 2026\n\npackage main",
			wantUpdated: true,
		},
		{
			name:        "Update with Python comment style",
			content:     "# Copyright (c) HashiCorp, Inc. 2023\n\nprint('hello')",
			wantContent: "# Copyright IBM Corp. 2023, 2026\n\nprint('hello')",
			wantUpdated: true,
		},
		{
			name:        "Update with block comment style",
			content:     "/*\n * Copyright (c) HashiCorp, Inc. 2023\n */\n\nint main() {}",
			wantContent: "/*\n * Copyright IBM Corp. 2023, 2026\n */\n\nint main() {}",
			wantUpdated: true,
		},
		{
			name:        "Update with HTML comment style",
			content:     "<!-- Copyright (c) HashiCorp, Inc. 2023 -->\n\n<html></html>",
			wantContent: "<!-- Copyright IBM Corp. 2023, 2026 -->\n\n<html></html>",
			wantUpdated: true,
		},
		{
			name:        "Update HashiCorp Inc without comma",
			content:     "// Copyright HashiCorp Inc 2023\n\npackage main",
			wantContent: "// Copyright IBM Corp. 2023, 2026\n\npackage main",
			wantUpdated: true,
		},
		{
			name:        "Update with year range",
			content:     "// Copyright (c) HashiCorp, Inc. 2020, 2023\n\npackage main",
			wantContent: "// Copyright IBM Corp. 2023, 2026\n\npackage main",
			wantUpdated: true,
		},
		{
			name:        "No update when different holder",
			content:     "// Copyright (c) Google LLC 2023\n\npackage main",
			wantContent: "// Copyright (c) Google LLC 2023\n\npackage main",
			wantUpdated: false,
		},
		{
			name:        "No update when no copyright",
			content:     "package main\n\nfunc main() {}",
			wantContent: "package main\n\nfunc main() {}",
			wantUpdated: false,
		},
		{
			name:        "No update for HashiCorp in code body",
			content:     "package main\n\n// This mentions HashiCorp, Inc.\nfunc main() {}",
			wantContent: "package main\n\n// This mentions HashiCorp, Inc.\nfunc main() {}",
			wantUpdated: false,
		},
		{
			name:        "Update case insensitive",
			content:     "// Copyright (c) HASHICORP, INC. 2023\n\npackage main",
			wantContent: "// Copyright IBM Corp. 2023, 2026\n\npackage main",
			wantUpdated: true,
		},
		{
			name:        "Update with extra whitespace",
			content:     "//   Copyright  (c)  HashiCorp, Inc.  2023\n\npackage main",
			wantContent: "//   Copyright IBM Corp. 2023, 2026\n\npackage main",
			wantUpdated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpfile, err := os.CreateTemp("", "test-*.go")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.Remove(tmpfile.Name()); err != nil {
					t.Logf("Failed to remove temp file: %v", err)
				}
			}()

			// Write test content
			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			// Run update
			updated, err := updateLicenseHolder(tmpfile.Name(), 0644, data)
			if err != nil {
				t.Fatalf("updateLicenseHolder() error = %v", err)
			}

			if updated != tt.wantUpdated {
				t.Errorf("updateLicenseHolder() updated = %v, want %v", updated, tt.wantUpdated)
			}

			// Read result
			result, err := os.ReadFile(tmpfile.Name())
			if err != nil {
				t.Fatal(err)
			}

			if string(result) != tt.wantContent {
				t.Errorf("updateLicenseHolder() result:\ngot:\n%s\n\nwant:\n%s", result, tt.wantContent)
			}
		})
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
		if got := FileMatches(tt.path, patterns); got != tt.wantMatch {
			t.Errorf("fileMatches(%q, %q) returned %v, want %v", tt.path, patterns, got, tt.wantMatch)
		}
	}
}

func TestWouldUpdateLicenseHolder(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		license  LicenseData
		expected bool
	}{
		{
			name:     "Would update HashiCorp, Inc.",
			content:  "// Copyright (c) 2023 HashiCorp, Inc.\n",
			license:  LicenseData{Holder: "IBM Corp.", Year: "2026"},
			expected: true,
		},
		{
			name:     "Would update HashiCorp Inc without comma",
			content:  "// Copyright 2023 HashiCorp Inc\n",
			license:  LicenseData{Holder: "IBM Corp.", Year: "2026"},
			expected: true,
		},
		{
			name:     "Would not update different holder",
			content:  "// Copyright 2023 Google LLC\n",
			license:  LicenseData{Holder: "IBM Corp.", Year: "2026"},
			expected: false,
		},
		{
			name:     "Would not update no copyright",
			content:  "// This is just a comment\n",
			license:  LicenseData{Holder: "IBM Corp.", Year: "2026"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "test")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.Remove(tmpfile.Name()); err != nil {
					t.Logf("Failed to remove temp file: %v", err)
				}
			}()

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			got, err := wouldUpdateLicenseHolder(tmpfile.Name(), tt.license)
			if err != nil {
				t.Fatalf("wouldUpdateLicenseHolder returned error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("wouldUpdateLicenseHolder() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestIsDirectory(t *testing.T) {
	// Create temporary directory for test
	tmpDir := tempDir(t)
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Test regular file
	tmpfile, err := os.CreateTemp(tmpDir, "test-file-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Test regular directory
	testDir := filepath.Join(tmpDir, "test-directory")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test symlink to file
	symlinkToFile := filepath.Join(tmpDir, "symlink-to-file")
	if err := os.Symlink(tmpfile.Name(), symlinkToFile); err != nil {
		t.Fatal(err)
	}

	// Test symlink to directory
	symlinkToDir := filepath.Join(tmpDir, "symlink-to-dir")
	if err := os.Symlink(testDir, symlinkToDir); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
		wantErr  bool
	}{
		{
			name:     "Regular file",
			path:     tmpfile.Name(),
			expected: false,
			wantErr:  false,
		},
		{
			name:     "Regular directory",
			path:     testDir,
			expected: true,
			wantErr:  false,
		},
		{
			name:     "Symlink to file",
			path:     symlinkToFile,
			expected: false,
			wantErr:  false,
		},
		{
			name:     "Symlink to directory",
			path:     symlinkToDir,
			expected: true,
			wantErr:  false,
		},
		{
			name:     "Non-existent path",
			path:     filepath.Join(tmpDir, "does-not-exist"),
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isDirectory(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("isDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("isDirectory() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestUpdateLicenseHolderSkipsDirectories(t *testing.T) {
	// Create temporary directory for test
	tmpDir := tempDir(t)
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	data := LicenseData{Holder: "IBM Corp.", Year: "2026", SPDXID: "MPL-2.0"}

	// Test regular directory
	testDir := filepath.Join(tmpDir, "test-directory")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test symlink to directory
	symlinkToDir := filepath.Join(tmpDir, "symlink-to-dir")
	if err := os.Symlink(testDir, symlinkToDir); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
	}{
		{
			name: "Regular directory",
			path: testDir,
		},
		{
			name: "Symlink to directory",
			path: symlinkToDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, err := updateLicenseHolder(tt.path, 0755, data)
			if err != nil {
				t.Fatalf("updateLicenseHolder() should not error on directories: %v", err)
			}
			if updated {
				t.Errorf("updateLicenseHolder() should not update directories, got updated=true")
			}
		})
	}
}

func TestWouldUpdateLicenseHolderSkipsDirectories(t *testing.T) {
	// Create temporary directory for test
	tmpDir := tempDir(t)
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	data := LicenseData{Holder: "IBM Corp.", Year: "2026", SPDXID: "MPL-2.0"}

	// Test regular directory
	testDir := filepath.Join(tmpDir, "test-directory")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test symlink to directory
	symlinkToDir := filepath.Join(tmpDir, "symlink-to-dir")
	if err := os.Symlink(testDir, symlinkToDir); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
	}{
		{
			name: "Regular directory",
			path: testDir,
		},
		{
			name: "Symlink to directory",
			path: symlinkToDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wouldUpdate, err := wouldUpdateLicenseHolder(tt.path, data)
			if err != nil {
				t.Fatalf("wouldUpdateLicenseHolder() should not error on directories: %v", err)
			}
			if wouldUpdate {
				t.Errorf("wouldUpdateLicenseHolder() should not update directories, got wouldUpdate=true")
			}
		})
	}
}

func TestDirectorySkippingRegressionTest(t *testing.T) {
	// Regression test for: Fix 'is a directory' error in copyright holder migration
	// This test simulates the scenario that was causing crashes where filepath.Walk
	// encounters symlinks to directories, such as version directories in test fixtures

	// Create temporary directory structure similar to test fixtures
	tmpDir := tempDir(t)
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	data := LicenseData{Holder: "IBM Corp.", Year: "2023, 2026", SPDXID: "MPL-2.0"}

	// Create test fixture structure
	testFixtureDir := filepath.Join(tmpDir, "test-fixture")
	if err := os.Mkdir(testFixtureDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a version directory (like "v1.2.3")
	versionDir := filepath.Join(testFixtureDir, "v1.2.3")
	if err := os.Mkdir(versionDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to the version directory (this was causing the crash)
	symlinkToVersion := filepath.Join(testFixtureDir, "latest")
	if err := os.Symlink("v1.2.3", symlinkToVersion); err != nil {
		t.Fatal(err)
	}

	// Add a test file with HashiCorp copyright in the version directory
	testFile := filepath.Join(versionDir, "test.go")
	testContent := "// Copyright (c) HashiCorp, Inc. 2023\n\npackage main\n\nfunc main() {}"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Walk the directory structure like the real code would
	var pathsToUpdate []string
	err := filepath.Walk(testFixtureDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories (this is where the bug was)
		isDir, dirErr := isDirectory(path)
		if dirErr != nil {
			return dirErr
		}
		if isDir {
			return nil // Skip directories and symlinks to directories
		}

		// Check if this file would be updated
		wouldUpdate, checkErr := wouldUpdateLicenseHolder(path, data)
		if checkErr != nil {
			t.Fatalf("wouldUpdateLicenseHolder failed on %s: %v", path, checkErr)
		}

		if wouldUpdate {
			pathsToUpdate = append(pathsToUpdate, path)
		}

		return nil
	})

	if err != nil {
		t.Fatalf("filepath.Walk should not fail with proper directory skipping: %v", err)
	}

	// Verify we found the test file but skipped directories
	if len(pathsToUpdate) != 1 {
		t.Fatalf("Expected to find 1 file to update, found %d: %v", len(pathsToUpdate), pathsToUpdate)
	}

	if pathsToUpdate[0] != testFile {
		t.Errorf("Expected to find test file %s, found %s", testFile, pathsToUpdate[0])
	}

	// Verify we can actually update the file without crashes
	updated, err := updateLicenseHolder(pathsToUpdate[0], 0644, data)
	if err != nil {
		t.Fatalf("updateLicenseHolder should not fail: %v", err)
	}

	if !updated {
		t.Errorf("updateLicenseHolder should have updated the file")
	}

	// Verify content was updated correctly
	result, err := os.ReadFile(pathsToUpdate[0])
	if err != nil {
		t.Fatal(err)
	}

	expectedContent := "// Copyright IBM Corp. 2023, 2026\n\npackage main\n\nfunc main() {}"
	if string(result) != expectedContent {
		t.Errorf("File content not updated correctly:\ngot:\n%s\n\nwant:\n%s", result, expectedContent)
	}
}
