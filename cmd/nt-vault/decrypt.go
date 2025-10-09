package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/vault"
	"github.com/spf13/cobra"
)

var decryptOutput string

func init() {
	decryptCmd.Flags().StringVarP(&decryptOutput, "output", "o", "", "output file path (default: stdout)")
	rootCmd.AddCommand(decryptCmd)
}

var decryptCmd = &cobra.Command{
	Use:   "decrypt <path/to/file.md>",
	Short: "Decrypt an encrypted file",
	Long:  `Decrypts the supplied file using the provided vault secret. Outputs to stdout if no --output is specified.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the file path
		filePath := args[len(args)-1]

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: file %s does not exist\n", filePath)
			os.Exit(1)
		}

		// Read the encrypted file
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		// Check if file is encrypted
		if !vault.IsEncrypted(content) {
			fmt.Fprintf(os.Stderr, "Error: file %s is not encrypted\n", filePath)
			os.Exit(1)
		}

		// Load key
		key, err := vault.LoadKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading key: %v\n", err)
			os.Exit(1)
		}

		// Decrypt the file
		decrypted, err := vault.DecryptFile(content, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decrypting file: %v\n", err)
			os.Exit(1)
		}

		// Output
		if decryptOutput == "" || decryptOutput == "stdout" {
			// Write to stdout
			fmt.Print(string(decrypted))
		} else {
			// Write to file
			if err := os.WriteFile(decryptOutput, decrypted, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Decrypted content written to: %s\n", decryptOutput)
		}
	},
}
