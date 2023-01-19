// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package actions

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/text"
)

///////////////////////////////////
//     GitHub Actions Helpers    //
///////////////////////////////////

// GHA helps write output for GitHub Actions-specific cases
type GHA struct {
	outWriter io.Writer

	isGHA bool
}

// Annotation represents a message that can optionally be attributed to a
// specific file location in GitHub. It is shown in the Actions Workflow Run UI
type Annotation struct {
	// The annotation's content body
	Message string

	// (optional) Custom title
	Title string

	// (optional) Filename
	File string

	// (optional) Line number, starting at 1
	Line int

	// (optional) Ending line number, starting at 1
	EndLine int

	// Col and EndColumn currently left out
}

// ErrorNotInGHA is the error returned when a function can only
// execute in GitHub Actions, but the current execution
// environment is NOT GitHub Actions
var ErrorNotInGHA = errors.New("Not in GitHub Actions")

// New returns a new GitHub Actions Writer
func New(out io.Writer) *GHA {
	// Default to looking up if we're running in GitHub Actions
	isGHA := os.Getenv("GITHUB_ACTIONS") == "true"
	return &GHA{outWriter: out, isGHA: isGHA}
}

// IsGHA returns true if the program is executing inside of GitHub Actions
func (gha *GHA) IsGHA() bool {
	return gha.isGHA
}

// DisableGHAOutput forcibly disables GitHub Actions-specific output types (e.g., groups)
func (gha *GHA) DisableGHAOutput(in bool) {
	gha.isGHA = false
}

// EnableGHAOutput forcibly enables GitHub Actions-specific output types (e.g., groups)
func (gha *GHA) EnableGHAOutput() {
	gha.isGHA = true
}

// StartGroup creates a GitHub Actions logging group
// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#grouping-log-lines
func (gha *GHA) StartGroup(name string) {
	if !gha.IsGHA() {
		gha.println(text.Bold.Sprint(name))
		return
	}

	out := "::group::" + name
	gha.println(out)
}

// EndGroup ends a GitHub Actions logging group
// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#grouping-log-lines
func (gha *GHA) EndGroup() {
	if !gha.IsGHA() {
		return
	}

	gha.println("::endgroup::")
}

// SetOutput generates a GitHub Actions output for the current job
// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#setting-an-output-parameter
func (gha *GHA) SetOutput(name, value string) error {
	content := fmt.Sprintf("%s=%s", name, value)
	return gha.appendToFile("GITHUB_OUTPUT", content)
}

// ExportVariable makes an environment variable available to subsequent steps
// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#setting-an-environment-variable
func (gha *GHA) ExportVariable(name, value string) error {
	content := fmt.Sprintf("%s=%s", name, value)
	return gha.appendToFile("GITHUB_ENV", content)
}

// SetJobSummary appends markdown displayed on the summary page of a workflow run
// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#setting-an-environment-variable
func (gha *GHA) SetJobSummary(content string) error {
	return gha.appendToFile("GITHUB_STEP_SUMMARY", content)
}

// appendToFile is an internal helper for adding content to a given file in a
// safe and consistent way. It can be used for populating GHA Environment Files.
// A newline will automatically be added to the content string if not present
//
// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#environment-files
func (gha *GHA) appendToFile(fileEnvVar string, content string) error {
	path, exists := os.LookupEnv(fileEnvVar)
	if !gha.IsGHA() || !exists {
		return fmt.Errorf("Unable to set modify GitHub Actions environment file %s: %w", fileEnvVar, ErrorNotInGHA)
	}

	// Short cut if no content is provided
	if content == "" {
		return nil
	}

	// append a newline if not currently present
	if !strings.HasSuffix(content, "\n") {
		content = content + "\n"
	}

	// Open the file and attempt to write to it
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

// Notice creates a notice message and prints the message to the log
// This message will create an annotation, which can associate the message with
// a particular file in your repository. Optionally, your message can specify a
// position within the file.
func (gha *GHA) Notice(a Annotation) { gha.newAnnotation("notice", a) }

// Warning creates a warning message and prints the message to the log.
// This message will create an annotation, which can associate the message with
// a particular file in your repository. Optionally, your message can specify a
// position within the file.
func (gha *GHA) Warning(a Annotation) { gha.newAnnotation("warning", a) }

// Error creates an error message and prints the message to the log.
// This message will create an annotation, which can associate the message with
// a particular file in your repository. Optionally, your message can specify a
// position within the file.
func (gha *GHA) Error(a Annotation) { gha.newAnnotation("error", a) }

// newAnnotation is an internal helper for creating notice, warning, and error
// annotations given a well-formed Annotation struct as input
//
// T specifies the annotation type, usually "notice", "warning", or "error"
//
// a specifies the content of the annotation
func (gha *GHA) newAnnotation(T string, a Annotation) {
	if !gha.IsGHA() {
		return
	}

	// TODO: maybe reflect would be cleaner for this?
	attributes := []string{}
	if a.Title != "" {
		attributes = append(attributes, fmt.Sprintf("title=%s", a.Title))
	}
	if a.File != "" {
		attributes = append(attributes, fmt.Sprintf("file=%s", a.File))
	}
	if a.Line != 0 {
		attributes = append(attributes, fmt.Sprintf("line=%d", a.Line))
	}
	if a.EndLine != 0 {
		attributes = append(attributes, fmt.Sprintf("endLine=%d", a.EndLine))
	}

	// General format should be:
	// "::error file={name},line={line},endLine={endLine},title={title}::{message}"
	// "::error file=app.js,line=1,title=Syntax Error::Missing semicolon"
	str := fmt.Sprintf("::%s %s::%s", T, strings.Join(attributes, ","), a.Message)
	gha.println(str)
}

// println is an internal helper for printing to the expected output io.Writer
func (gha *GHA) println(i ...interface{}) {
	fmt.Fprintln(gha.outWriter, i...)
}
