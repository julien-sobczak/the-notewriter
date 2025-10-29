package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(indexPackCmd)
}

var indexPackCmd = &cobra.Command{
	Use:   "index-pack <packfiles...>",
	Short: "Index pack files into the repository",
	Long:  `Index one or more pack files (.pack) into the repository database and index.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var absPaths []string
		for _, path := range args {
			abs, err := filepath.Abs(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to get absolute path for %s: %v\n", path, err)
				os.Exit(1)
			}
			absPaths = append(absPaths, abs)
		}

		err := core.CurrentRepository().IndexPackFiles(absPaths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to index pack files: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Indexed %d pack file(s)\n", len(absPaths))
	},
}
