package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:   "edit -- <path/to/file.md>",
	Short: "Edit an encrypted file",
	Long:  `Opens and decrypts an existing vaulted file in an editor, that will be encrypted again when closed.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the file path (after --)
		filePath := args[len(args)-1]

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: file %s does not exist\n", filePath)
			os.Exit(1)
		}

		// Load encryption key
		key, err := LoadKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading key: %v\n", err)
			os.Exit(1)
		}

		// Read the encrypted file
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		// Check if file is encrypted
		if !IsEncrypted(content) {
			fmt.Fprintf(os.Stderr, "Error: file %s is not encrypted\n", filePath)
			os.Exit(1)
		}

		// Decrypt the file
		decrypted, err := DecryptFile(content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decrypting file: %v\n", err)
			os.Exit(1)
		}

		// Create a temporary file with decrypted content
		tmpFile, err := CreateTempFile(decrypted, ".md")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating temporary file: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(tmpFile)

		// Get the original modification time
		origModTime, err := GetFileModTime(tmpFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting file mod time: %v\n", err)
			os.Exit(1)
		}

		// Open editor
		editor := GetEditor()
		editorCmd := exec.Command(editor, tmpFile)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
			os.Exit(1)
		}

		// Check if file was modified
		newModTime, err := GetFileModTime(tmpFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting file mod time: %v\n", err)
			os.Exit(1)
		}

		if newModTime.Equal(origModTime) {
			fmt.Println("File not modified, not updating.")
			return
		}

		// Read the edited content
		editedContent, err := os.ReadFile(tmpFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading temporary file: %v\n", err)
			os.Exit(1)
		}

		// Encrypt the content
		encrypted, err := EncryptFile(editedContent, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encrypting file: %v\n", err)
			os.Exit(1)
		}

		// Write back to the original file
		if err := os.WriteFile(filePath, encrypted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Encrypted file updated: %s\n", filePath)
	},
}
