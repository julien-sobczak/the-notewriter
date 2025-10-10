package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(viewCmd)
}

var viewCmd = &cobra.Command{
	Use:   "view <path/to/file.md>",
	Short: "View an encrypted file",
	Long:  `Opens, decrypts and views an existing vaulted file using a pager.`,
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
		tmpFile, err := os.CreateTemp("", "nt-vault-*.md")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating temporary file: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(tmpFile.Name())
		if err := file.DecryptTo(tmpFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error decrypting file: %v\n", err)
			os.Exit(1)
		}

		// Open pager
		pager := GetPager()
		pagerCmd := exec.Command(pager, tmpFile.Name())
		pagerCmd.Stdin = os.Stdin
		pagerCmd.Stdout = os.Stdout
		pagerCmd.Stderr = os.Stderr

		if err := pagerCmd.Run(); err != nil {
			// If pager fails, just cat the file
			if err := file.DecryptTo(os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "Error decrypting file: %v\n", err)
				os.Exit(1)
			}
		}
	},
}
