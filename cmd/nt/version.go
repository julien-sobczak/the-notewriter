package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the current version of the nt command.
// It can be overridden at build time using:
//
//	go build -ldflags "-X main.Version=<version>"
var Version = "development"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Long:  `Print the version of the nt command.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("nt version %s\n", Version)
	},
}
