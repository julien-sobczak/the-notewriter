package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/spf13/cobra"
)

func init() {
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
		_, err = core.InitConfigFromDirectory(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while initializing configuration: %v", err)
			os.Exit(1)
		}
	},
}
