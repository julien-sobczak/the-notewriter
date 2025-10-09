package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/vault"
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
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: file %s does not exist\n", filePath)
			os.Exit(1)
		}

		// Load encryption key
		key, err := vault.LoadKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading key: %v\n", err)
			os.Exit(1)
		}

		// Read the plaintext file
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		// Check if file is already encrypted
		if vault.IsEncrypted(content) {
			fmt.Fprintf(os.Stderr, "Error: file %s is already encrypted\n", filePath)
			os.Exit(1)
		}

		// Encrypt the content
		encrypted, err := vault.EncryptFile(content, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encrypting file: %v\n", err)
			os.Exit(1)
		}

		// Write back to the same file
		if err := os.WriteFile(filePath, encrypted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File encrypted: %s\n", filePath)
	},
}
