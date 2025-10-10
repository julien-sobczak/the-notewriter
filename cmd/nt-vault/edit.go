package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:   "edit <path/to/file.md>",
	Short: "Edit an encrypted file",
	Long:  `Opens and decrypts an existing vaulted file in an editor, that will be encrypted again when closed.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the file path
		filePath := args[len(args)-1]

		file, err := markdown.ParseFileRaw(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
			os.Exit(1)
		}

		// Decrypt the file
		var buf bytes.Buffer
		if err := file.DecryptTo(&buf); err != nil {
			fmt.Fprintf(os.Stderr, "Error decrypting file: %v\n", err)
			os.Exit(1)
		}

		// Create a temporary file with decrypted content
		tmpFile, err := os.CreateTemp("", "nt-vault-*.md")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating temporary file: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(tmpFile.Name(), buf.Bytes(), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to temporary file: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(tmpFile.Name())

		// Get the original modification time
		tmpStat, err := tmpFile.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting file mod time: %v\n", err)
			os.Exit(1)
		}
		origModTime := tmpStat.ModTime()

		// Open editor
		editorCmd := GetEditorCmd(tmpFile.Name())
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
			os.Exit(1)
		}

		// Read the edited content
		editedFile, err := markdown.ParseFileRaw(tmpFile.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading edited file: %v\n", err)
			os.Exit(1)
		}
		if editedFile.MTime.Equal(origModTime) {
			fmt.Println("File not modified, not updating.")
			return
		}
		f, err := os.Create(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening output file: %v\n", err)
			os.Exit(1)
		}
		err = editedFile.EncryptTo(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encrypting file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Encrypted file updated: %s\n", filePath)
	},
}
