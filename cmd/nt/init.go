package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	initCmd.Flags().BoolVarP(&interactive, "i", "", false, "ask before pulling")
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Init new notebook",
	Long:  `Set up local directory as the root of a new notebook.`,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to read current working directory: %v", err)
			os.Exit(1)
		}

		_, err = core.InitConfigFromDirectory(cwd, core.DefaultConfigOptions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while initializing configuration: %v", err)
			os.Exit(1)
		}

		// Check media converter immediately (better to install dependencies now than later)
		if core.CurrentConfigFile().Core.Medias.Command != "" {
			// Check executable is present in PATH
			mediaCmd := core.CurrentConfigFile().Core.Medias.Command
			if strings.Contains(mediaCmd, string(filepath.Separator)) {
				// mediaCmd contains a path separator, treat as a path
				_, err := os.Stat(core.CurrentConfigFile().Core.Medias.Command)
				if os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "Error: media command not found: %s\n", mediaCmd)
					os.Exit(1)
				}
			} else {
				// mediaCmd is just a command name, look in PATH
				_, err := exec.LookPath(mediaCmd)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: media command not found in PATH: %s\n", mediaCmd)
					os.Exit(1)
				}
			}
		}

	},
}
