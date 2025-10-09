package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create -- <path/to/file.md>",
	Short: "Create a new encrypted file",
	Long:  `Opens an editor to create a new file that will be encrypted when the editor is closed.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the file path (after --)
		filePath := args[len(args)-1]

		// Check if file already exists
		if _, err := os.Stat(filePath); err == nil {
			fmt.Fprintf(os.Stderr, "Error: file %s already exists\n", filePath)
			os.Exit(1)
		}

		// Load encryption key
		key, err := LoadKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading key: %v\n", err)
			os.Exit(1)
		}

		// Create a temporary file for editing
		tmpFile, err := CreateTempFile([]byte{}, ".md")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating temporary file: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(tmpFile)

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

		// Read the edited content
		content, err := os.ReadFile(tmpFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading temporary file: %v\n", err)
			os.Exit(1)
		}

		// If the file is empty, don't create it
		if len(content) == 0 {
			fmt.Println("Empty file, not creating.")
			return
		}

		// Encrypt the content
		encrypted, err := EncryptFile(content, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encrypting file: %v\n", err)
			os.Exit(1)
		}

		// Write to the target file
		if err := os.WriteFile(filePath, encrypted, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Encrypted file created: %s\n", filePath)
	},
}
