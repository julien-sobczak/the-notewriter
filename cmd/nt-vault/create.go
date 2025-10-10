package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create <path/to/file.md>",
	Short: "Create a new encrypted file",
	Long:  `Opens an editor to create a new file that will be encrypted when the editor is closed.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the file path
		filePath := args[len(args)-1]

		// Check if file already exists
		if _, err := os.Stat(filePath); err == nil {
			fmt.Fprintf(os.Stderr, "Error: file %s already exists\n", filePath)
			os.Exit(1)
		}

		// Create a temporary file for editing
		tmpFile, err := os.CreateTemp("", "nt-vault-*.md")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating temporary file: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(tmpFile.Name())

		// Open editor
		editorCmd := GetEditorCmd(tmpFile.Name())
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr
		if err := editorCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
			os.Exit(1)
		}

		// Read the file
		file, err := markdown.ParseFile(tmpFile.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading temporary file: %v\n", err)
			os.Exit(1)
		}

		// Encrypt the file
		out, err := os.Open(filePath)
		if err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error checking output file: %v\n", err)
		}
		if err := file.EncryptTo(out); err != nil {
			fmt.Fprintf(os.Stderr, "Error encrypting file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Encrypted file created: %s\n", filePath)
	},
}
