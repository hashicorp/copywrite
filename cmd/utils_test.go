// Copyright IBM Corp. 2023, 2026
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"bytes"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_colorize(t *testing.T) {
	text.EnableColors()

	tests := []struct {
		name           string
		inputString    string
		inputCodes     []text.Color
		expectedOutput string
	}{
		{
			name:           "Output left alone when no codes are specified",
			inputString:    "Hello, world!",
			inputCodes:     []text.Color{},
			expectedOutput: "Hello, world!",
		},
		{
			name:           "Output wrapped with stylistic escape sequence (bold)",
			inputString:    "Hello, world!",
			inputCodes:     []text.Color{text.Bold},
			expectedOutput: "\x1b[1mHello, world!\x1b[0m",
		},
		{
			name:           "Output wrapped with colored escape sequence (FgCyan)",
			inputString:    "Hello, world!",
			inputCodes:     []text.Color{text.FgCyan},
			expectedOutput: "\x1b[36mHello, world!\x1b[0m",
		},
		{
			name:           "Output properly wrapped with multiple escape sequences",
			inputString:    "Hello, world!",
			inputCodes:     []text.Color{text.Bold, text.FgCyan, text.BgBlack},
			expectedOutput: "\x1b[1;36;40mHello, world!\x1b[0m",
		},
		{
			name:           "Empty string with color codes returns empty",
			inputString:    "",
			inputCodes:     []text.Color{text.FgRed},
			expectedOutput: "",
		},
		{
			name:           "Single color code FgGreen",
			inputString:    "test",
			inputCodes:     []text.Color{text.FgGreen},
			expectedOutput: "\x1b[32mtest\x1b[0m",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualOutput := colorize(tt.inputString, tt.inputCodes...)
			assert.Equal(t, tt.expectedOutput, actualOutput)
		})
	}
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		commit   string
		expected string
	}{
		{
			name:     "default dev version",
			version:  "dev",
			commit:   "none",
			expected: "dev-none",
		},
		{
			name:     "release version with commit hash",
			version:  "1.2.3",
			commit:   "abc1234",
			expected: "1.2.3-abc1234",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldVersion := version
			oldCommit := commit
			t.Cleanup(func() {
				version = oldVersion
				commit = oldCommit
			})

			version = tt.version
			commit = tt.commit

			result := GetVersion()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_newTableWriter(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "creates table writer with custom style"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tw := newTableWriter(&buf)

			require.NotNil(t, tw)
			assert.Equal(t, "copywrite", tw.Style().Name)
			assert.Equal(t, text.FormatDefault, tw.Style().Format.Header)
			assert.Equal(t, false, tw.Style().Options.DrawBorder)
			assert.Equal(t, false, tw.Style().Options.SeparateColumns)
			assert.Equal(t, text.Colors{text.FgGreen}, tw.Style().Color.Header)
			assert.Equal(t, text.Colors{text.FgCyan}, tw.Style().Color.IndexColumn)
		})
	}
}

func Test_newTableWriter_renders_output(t *testing.T) {
	var buf bytes.Buffer
	tw := newTableWriter(&buf)

	tw.AppendHeader(stringArrayToRow([]string{"Name", "Value"}))
	tw.AppendRow(stringArrayToRow([]string{"foo", "bar"}))
	tw.Render()

	output := buf.String()
	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Value")
	assert.Contains(t, output, "foo")
	assert.Contains(t, output, "bar")
}

func Test_stringArrayToRow(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected int
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: 0,
		},
		{
			name:     "single element",
			input:    []string{"hello"},
			expected: 1,
		},
		{
			name:     "multiple elements",
			input:    []string{"a", "b", "c"},
			expected: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := stringArrayToRow(tt.input)
			assert.Len(t, row, tt.expected)

			for i, v := range tt.input {
				assert.Equal(t, v, row[i])
			}
		})
	}
}
