// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package actions

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/text"
	"github.com/stretchr/testify/assert"
)

// This test feels tautological, but it guards from regressions around checks
// for environment variables to determine if running in GitHub Actions or not
func Test_New(t *testing.T) {

	var b bytes.Buffer

	tests := []struct {
		name           string
		env            map[string]string
		expectedOutput *GHA
	}{
		{
			name: "Detect if running in a GitHub Actions environment",
			env: map[string]string{
				"GITHUB_ACTIONS": "true",
			},
			expectedOutput: &GHA{
				outWriter: &b,
				isGHA:     true,
			},
		},
		{
			name: "Detect if NOT running in a GitHub Actions environment",
			env: map[string]string{
				"GITHUB_ACTIONS": "false",
			},
			expectedOutput: &GHA{
				outWriter: &b,
				isGHA:     false,
			},
		},
		{
			name: "Detect if NOT running in a GitHub Actions environment due to absence of GITHUB_ACTIONS env var",
			env:  map[string]string{},
			expectedOutput: &GHA{
				outWriter: &b,
				isGHA:     false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				os.Setenv(k, v)
				fmt.Println("Setting env var " + k + " equal to " + v)
			}

			gha := New(&b)
			assert.Equal(t, tt.expectedOutput.isGHA, gha.isGHA)

			for k := range tt.env {
				os.Unsetenv(k)
			}
		})
	}
}

func Test_StartGroup(t *testing.T) {
	// Let's take colorized output out of the picture
	text.DisableColors()

	tests := []struct {
		name           string
		isGHA          bool
		groupName      string
		expectedOutput string
	}{
		{
			name:           "Output start group workflow command if executing in GitHub Actions",
			isGHA:          true,
			groupName:      "asdf",
			expectedOutput: "::group::asdf\n",
		},
		{
			name:           "Output ordinary group name for logging purposes if not executing in GitHub Actions",
			isGHA:          false,
			groupName:      "asdf",
			expectedOutput: "asdf\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			gha := &GHA{
				outWriter: &b,
				isGHA:     tt.isGHA,
			}
			gha.StartGroup(tt.groupName)
			assert.Equal(t, tt.expectedOutput, b.String())
		})
	}
}

func Test_EndGroup(t *testing.T) {
	tests := []struct {
		name           string
		isGHA          bool
		expectedOutput string
	}{
		{
			name:           "Output end group workflow command if executing in GitHub Actions",
			isGHA:          true,
			expectedOutput: "::endgroup::\n",
		},
		{
			name:           "Don't output anything if not executing in GitHub Actions",
			isGHA:          false,
			expectedOutput: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			gha := &GHA{
				outWriter: &b,
				isGHA:     tt.isGHA,
			}
			gha.EndGroup()
			assert.Equal(t, tt.expectedOutput, b.String())
		})
	}
}

func Test_appendToFile(t *testing.T) {
	// Let's take colorized output out of the picture
	text.DisableColors()

	tests := []struct {
		name           string
		isGHA          bool
		input          string
		expectedOutput string
		expectedError  error
	}{
		{
			name:           "Empty input doesn't write anything",
			isGHA:          true,
			input:          "",
			expectedOutput: "",
			expectedError:  nil,
		},
		{
			name:           "Newline gets added to input",
			isGHA:          true,
			input:          "key=value",
			expectedOutput: "key=value\n",
			expectedError:  nil,
		},
		{
			name:           "Already present newline gets left alone",
			isGHA:          true,
			input:          "key=value\n",
			expectedOutput: "key=value\n",
			expectedError:  nil,
		},
		{
			name:           "Function does nothing if not in GitHub Actions",
			isGHA:          false,
			input:          "key=value\n",
			expectedOutput: "",
			expectedError:  ErrorNotInGHA,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			gha := &GHA{
				outWriter: &b,
				isGHA:     tt.isGHA,
			}

			// Create a temporary file to write to
			f, err := os.CreateTemp("", "*")
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")

			// Store the name of that file in an env var
			fileEnvVar := fmt.Sprintf("TEST_APPEND_%v", f.Name())
			err = os.Setenv(fileEnvVar, f.Name())
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")

			// Call appendToFile() with the new env var with test case input
			err = gha.appendToFile(fileEnvVar, tt.input)
			assert.ErrorIs(t, err, tt.expectedError)

			// read file
			var out strings.Builder
			_, err = io.Copy(&out, f)
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")
			assert.Equal(t, tt.expectedOutput, out.String())

			// clean up
			err = os.Remove(f.Name())
			assert.Nil(t, err, fmt.Sprintf("Unable to clean up temporary test file: %s", f.Name()))
		})
	}
}

func Test_SetOutput(t *testing.T) {
	// Let's take colorized output out of the picture
	text.DisableColors()

	tests := []struct {
		name           string
		inputName      string
		inputValue     string
		expectedOutput string
		expectedError  error
	}{
		{
			name:           "Key-value pair gets written appropriately",
			inputName:      "key",
			inputValue:     "value",
			expectedOutput: "key=value\n",
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			gha := &GHA{
				outWriter: &b,
				isGHA:     true,
			}

			// Create a temporary file to write to
			f, err := os.CreateTemp("", "*")
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")

			// Store the name of that file in the GITHUB_OUTPUT env var
			err = os.Setenv("GITHUB_OUTPUT", f.Name())
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")

			// Call appendToFile() with the new env var with test case input
			err = gha.SetOutput(tt.inputName, tt.inputValue)
			assert.ErrorIs(t, err, tt.expectedError)

			// read file
			var out strings.Builder
			_, err = io.Copy(&out, f)
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")
			assert.Equal(t, tt.expectedOutput, out.String())

			// clean up
			err = os.Remove(f.Name())
			assert.Nil(t, err, fmt.Sprintf("Unable to clean up temporary test file: %s", f.Name()))
		})
	}
}

func Test_ExportVariable(t *testing.T) {
	// Let's take colorized output out of the picture
	text.DisableColors()

	tests := []struct {
		name           string
		inputName      string
		inputValue     string
		expectedOutput string
		expectedError  error
	}{
		{
			name:           "Key-value pair gets exported appropriately",
			inputName:      "key",
			inputValue:     "value",
			expectedOutput: "key=value\n",
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			gha := &GHA{
				outWriter: &b,
				isGHA:     true,
			}

			// Create a temporary file to write to
			f, err := os.CreateTemp("", "*")
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")

			// Store the name of that file in the GITHUB_ENV env var
			err = os.Setenv("GITHUB_ENV", f.Name())
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")

			// Call ExportVariable() with the new env var with test case input
			err = gha.ExportVariable(tt.inputName, tt.inputValue)
			assert.ErrorIs(t, err, tt.expectedError)

			// read file
			var out strings.Builder
			_, err = io.Copy(&out, f)
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")
			assert.Equal(t, tt.expectedOutput, out.String())

			// clean up
			err = os.Remove(f.Name())
			assert.Nil(t, err, fmt.Sprintf("Unable to clean up temporary test file: %s", f.Name()))
		})
	}
}

func Test_SetJobSummary(t *testing.T) {
	// Let's take colorized output out of the picture
	text.DisableColors()

	tests := []struct {
		name           string
		input          string
		expectedOutput string
		expectedError  error
	}{
		{
			name:           "Markdown content gets written appropriately",
			input:          "# Title\nThis is `markdown`!",
			expectedOutput: "# Title\nThis is `markdown`!\n",
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			gha := &GHA{
				outWriter: &b,
				isGHA:     true,
			}

			// Create a temporary file to write to
			f, err := os.CreateTemp("", "*")
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")

			// Store the name of that file in the GITHUB_STEP_SUMMARY env var
			err = os.Setenv("GITHUB_STEP_SUMMARY", f.Name())
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")

			// Call SetJobSummary() with the new env var with test case input
			err = gha.SetJobSummary(tt.input)
			assert.ErrorIs(t, err, tt.expectedError)

			// read file
			var out strings.Builder
			_, err = io.Copy(&out, f)
			assert.Nil(t, err, "If this broke, the test is wrong, not the function under test")
			assert.Equal(t, tt.expectedOutput, out.String())

			// clean up
			err = os.Remove(f.Name())
			assert.Nil(t, err, fmt.Sprintf("Unable to clean up temporary test file: %s", f.Name()))
		})
	}
}

func Test_newAnnotation(t *testing.T) {
	// Let's take colorized output out of the picture
	text.DisableColors()

	tests := []struct {
		name           string
		annotationType string
		isGHA          bool
		input          Annotation
		expectedOutput string
	}{
		{
			name:           "Empty annotation works properly",
			annotationType: "error",
			isGHA:          true,
			input:          Annotation{},
			expectedOutput: "::error ::\n",
		},
		{
			name:           "Message text prints properly",
			annotationType: "error",
			isGHA:          true,
			input: Annotation{
				Message: "Lorem ipsum dolar sit",
			},
			expectedOutput: "::error ::Lorem ipsum dolar sit\n",
		},
		{
			name:           "Single annotation attribute prints without join separator",
			annotationType: "error",
			isGHA:          true,
			input: Annotation{
				Title:   "Hello World!",
				Message: "Lorem ipsum dolar sit",
			},
			expectedOutput: "::error title=Hello World!::Lorem ipsum dolar sit\n",
		},
		{
			name:           "Zero-initialized int does not get printed",
			annotationType: "warning",
			isGHA:          true,
			input: Annotation{
				Line:    0,
				EndLine: 0,
			},
			expectedOutput: "::warning ::\n",
		},
		{
			name:           "Zero-initialized string does not get printed",
			annotationType: "warning",
			isGHA:          true,
			input: Annotation{
				Title:   "",
				File:    "",
				Message: "",
			},
			expectedOutput: "::warning ::\n",
		},
		{
			name:           "Fully filled Annotation prints properly",
			annotationType: "notice",
			isGHA:          true,
			input: Annotation{
				Title:   "Syntax Error",
				File:    "app.js",
				Message: "Missing semicolon",
				Line:    7,
				EndLine: 7,
			},
			expectedOutput: "::notice title=Syntax Error,file=app.js,line=7,endLine=7::Missing semicolon\n",
		},
		{
			name:           "Nothing gets printed if we aren't in GitHub Actions",
			annotationType: "notice",
			isGHA:          false,
			input: Annotation{
				Title:   "Syntax Error",
				File:    "app.js",
				Message: "Missing semicolon",
				Line:    7,
				EndLine: 7,
			},
			expectedOutput: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			gha := &GHA{
				outWriter: &b,
				isGHA:     tt.isGHA,
			}
			gha.newAnnotation(tt.annotationType, tt.input)
			assert.Equal(t, tt.expectedOutput, b.String())
		})
	}
}

// Test_newAnnotation covers most test cases - this is here primarily to
// ensure that the annotation type is passed properly to newAnnotation()
func Test_Notice(t *testing.T) {
	// Let's take colorized output out of the picture
	text.DisableColors()

	tests := []struct {
		name           string
		input          Annotation
		expectedOutput string
	}{
		{
			name:           "Empty annotation works properly",
			input:          Annotation{},
			expectedOutput: "::notice ::\n",
		},
		{
			name: "Fully filled Annotation prints properly",
			input: Annotation{
				Title:   "Syntax Error",
				File:    "app.js",
				Message: "Missing semicolon",
				Line:    7,
				EndLine: 7,
			},
			expectedOutput: "::notice title=Syntax Error,file=app.js,line=7,endLine=7::Missing semicolon\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			gha := &GHA{
				outWriter: &b,
				isGHA:     true,
			}
			gha.Notice(tt.input)
			assert.Equal(t, tt.expectedOutput, b.String())
		})
	}
}

// Test_newAnnotation covers most test cases - this is here primarily to
// ensure that the annotation type is passed properly to newAnnotation()
func Test_Warning(t *testing.T) {
	// Let's take colorized output out of the picture
	text.DisableColors()

	tests := []struct {
		name           string
		input          Annotation
		expectedOutput string
	}{
		{
			name:           "Empty annotation works properly",
			input:          Annotation{},
			expectedOutput: "::warning ::\n",
		},
		{
			name: "Fully filled Annotation prints properly",
			input: Annotation{
				Title:   "Syntax Error",
				File:    "app.js",
				Message: "Missing semicolon",
				Line:    7,
				EndLine: 7,
			},
			expectedOutput: "::warning title=Syntax Error,file=app.js,line=7,endLine=7::Missing semicolon\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			gha := &GHA{
				outWriter: &b,
				isGHA:     true,
			}
			gha.Warning(tt.input)
			assert.Equal(t, tt.expectedOutput, b.String())
		})
	}
}

// Test_newAnnotation covers most test cases - this is here primarily to
// ensure that the annotation type is passed properly to newAnnotation()
func Test_Error(t *testing.T) {
	// Let's take colorized output out of the picture
	text.DisableColors()

	tests := []struct {
		name           string
		input          Annotation
		expectedOutput string
	}{
		{
			name:           "Empty annotation works properly",
			input:          Annotation{},
			expectedOutput: "::error ::\n",
		},
		{
			name: "Fully filled Annotation prints properly",
			input: Annotation{
				Title:   "Syntax Error",
				File:    "app.js",
				Message: "Missing semicolon",
				Line:    7,
				EndLine: 7,
			},
			expectedOutput: "::error title=Syntax Error,file=app.js,line=7,endLine=7::Missing semicolon\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			gha := &GHA{
				outWriter: &b,
				isGHA:     true,
			}
			gha.Error(tt.input)
			assert.Equal(t, tt.expectedOutput, b.String())
		})
	}
}
