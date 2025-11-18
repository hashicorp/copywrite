// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"github.com/hashicorp/copywrite/cmd"
	"github.com/hashicorp/go-hclog"
)

func main() {
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "hc-copywrite",
		Level: hclog.LevelFromString("DEBUG"),
		Color: hclog.AutoColor,
	})
	hclog.SetDefault(appLogger)
	cmd.Execute()
}
