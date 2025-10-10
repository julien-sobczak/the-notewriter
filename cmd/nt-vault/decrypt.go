package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(decryptCmd)
}

var decryptCmd = &cobra.Command{
	Use:   "decrypt <path/to/file.md>",
	Short: "Decrypt an encrypted file",
	Long:  `Decrypts the supplied file using the provided vault secret.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the file path
		filePath := args[len(args)-1]

		// Read the file
		file, err := markdown.ParseFileRaw(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
			os.Exit(1)
		}

		// Write to file
		if err := file.Decrypt(); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing decrypted file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Decrypted content written to: %s\n", filePath)
	},
}
