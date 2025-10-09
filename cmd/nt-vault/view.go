package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(viewCmd)
}

var viewCmd = &cobra.Command{
	Use:   "view -- <path/to/file.md>",
	Short: "View an encrypted file",
	Long:  `Opens, decrypts and views an existing vaulted file using a pager.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get the file path (after --)
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

		// Open pager
		pager := GetPager()
		pagerCmd := exec.Command(pager, tmpFile)
		pagerCmd.Stdin = os.Stdin
		pagerCmd.Stdout = os.Stdout
		pagerCmd.Stderr = os.Stderr

		if err := pagerCmd.Run(); err != nil {
			// If pager fails, just cat the file
			fmt.Print(string(decrypted))
		}
	},
}
