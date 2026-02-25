// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/stretchr/testify/assert"
)

func Test_colorize(t *testing.T) {
	// Let's take console abilities out of the picture
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualOutput := colorize(tt.inputString, tt.inputCodes...)
			assert.Equal(t, tt.expectedOutput, actualOutput)
		})
	}
}
