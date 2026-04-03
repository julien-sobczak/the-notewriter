package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
)

func CheckConfig() {
	err := core.CurrentConfig().Check()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func SaveConfig() {
	err := core.CurrentConfig().Save()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// CheckConfigIfPresent validates the config only if a .nt/config.jsonnet file is
// found in the current directory tree. It is a no-op when run outside a repository.
func CheckConfigIfPresent() {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	cfg, err := core.ReadConfigFromDirectory(cwd)
	if err != nil || cfg == nil {
		return
	}
	if err := cfg.CheckIfPresent(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// SaveConfigIfPresent saves the config only if a .nt/config.jsonnet file is
// found in the current directory tree. It is a no-op when run outside a repository.
func SaveConfigIfPresent() {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	cfg, err := core.ReadConfigFromDirectory(cwd)
	if err != nil || cfg == nil {
		return
	}
	if err := cfg.Save(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
