package main

import (
	"github.com/hashicorp/copyright-notice-automation/cmd"
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
