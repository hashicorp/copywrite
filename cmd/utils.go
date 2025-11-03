// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

var (
	version = "dev"
	commit  = "none"
)

// GetVersion returns a version string corresponding to the current release.
// The version and commit SHA are dynamically provided at build-time via
// go-releaser's version/commit ldflags
// https://goreleaser.com/cookbooks/using-main.version
func GetVersion() string {
	return fmt.Sprintf("%v-%v", version, commit)
}

///////////////////////////////////
//     Table Output Helpers      //
///////////////////////////////////

// Return a new table writer with style 😎
func newTableWriter(out io.Writer) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(out)

	t.SetStyle(table.StyleLight)

	t.Style().Name = "copywrite"

	// Headers are UPPERCASE by default, but let's lend that decision the caller
	t.Style().Format.Header = text.FormatDefault

	// Coloring it!
	t.Style().Color.Header = text.Colors{text.FgGreen}
	t.Style().Color.IndexColumn = text.Colors{text.FgCyan}

	// Borders are ugly, so let's get rid of them!
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false

	return t
}

func stringArrayToRow(m []string) table.Row {
	row := make([]interface{}, 0)
	for _, v := range m {
		row = append(row, v)
	}
	return row
}

///////////////////////////////////
//  Pretty Print Output Helpers  //
///////////////////////////////////

// colorize expands the jedib0t/go-pretty/v6/text package by letting you supply
// multiple ANSI codes to be escaped together. For example, this allows you to
// make text both bold _and_ colored, instead of just one or the other.
//
// Example:
// escaped := colorize("Hello, world!", text.Bold, text.FgCyan, text.BgBlack)
// fmt.Println(escaped)
func colorize(s string, colors ...text.Color) string {
	if len(colors) == 0 {
		return s // short circuit
	}
	codes := []string{}
	for _, c := range colors {
		codes = append(codes, strconv.Itoa(int(c)))
	}
	escSeq := fmt.Sprintf("\x1b[%vm", strings.Join(codes, ";"))
	return text.Escape(s, escSeq)
}
