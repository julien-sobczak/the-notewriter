package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build information. Overridden at build time via -ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Long:  `Print the version of the nt command.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("nt version %s (commit %s) built on %s\n", Version, Commit, BuildDate)
	},
}
