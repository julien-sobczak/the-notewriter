package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(encryptCmd)
}

var encryptCmd = &cobra.Command{
	Use:   "encrypt <path/to/file.md>",
	Short: "Encrypt a plaintext file",
	Long:  `Encrypts the supplied file using the provided vault secret. The file is encrypted in-place.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the file path
		filePath := args[len(args)-1]

		// Check if file exists
		file, err := markdown.ParseFileRaw(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing file %s: %v\n", filePath, err)
			os.Exit(1)
		}

		// Encrypt the content
		if err := file.Encrypt(); err != nil {
			fmt.Fprintf(os.Stderr, "Error encrypting file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File encrypted: %s\n", filePath)
	},
}
