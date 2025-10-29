package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/julien-sobczak/the-notewriter/pkg/anki"
	"github.com/spf13/cobra"
)

var (
	ignoreScheduling bool
	mediaDir         string
)

var importCmd = &cobra.Command{
	Use:   "import [apkg-file] [markdown-file]",
	Short: "Import Anki flashcards from an .apkg file",
	Long: `Import Anki flashcards from an .apkg file.

The .apkg file is a ZIP archive containing:
- collection.anki2: SQLite database with notes and cards
- media: JSON file mapping media IDs to filenames
- Numbered files (0, 1, etc.): Media files

Example usage:
  nt-anki import col.apkg "web:skills/web/general.md"
  nt-anki import col.apkg --ignore-scheduling --media-dir="medias"`,
	Args: cobra.ExactArgs(2),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().BoolVar(&ignoreScheduling, "ignore-scheduling", false, "Ignore scheduling information from revlog table")
	importCmd.Flags().StringVar(&mediaDir, "media-dir", "", "Subdirectory for media files (relative to output file)")
}

func runImport(cmd *cobra.Command, args []string) error {
	apkgPath := args[0]
	markdownPath := args[1]

	// Validate the file exists
	if _, err := os.Stat(apkgPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", apkgPath)
	}

	// Extract and parse the Anki collection
	fmt.Printf("📦 Extracting %s...\n", filepath.Base(apkgPath))
	collection, err := anki.ExtractCollection(apkgPath)
	if err != nil {
		return fmt.Errorf("failed to extract collection: %w", err)
	}
	defer collection.Close()

	fmt.Printf("📚 Found %d notes and %d cards\n", len(collection.Notes), len(collection.Cards))

	// Convert and write flashcards
	packfilePath, err := collection.Import(markdownPath,
		anki.WithMediaDir(mediaDir),
		anki.WithIgnoreScheduling(ignoreScheduling))
	if err != nil {
		return fmt.Errorf("failed to import: %w", err)
	}

	fmt.Printf("\nThe file %s has been successfully updated.\n", markdownPath)
	fmt.Println("Review and edit the generated file. Then, run the commands to finish the import:")
	fmt.Printf("$ nt add %s\n", markdownPath)
	fmt.Println()
	if packfilePath != "" {
		fmt.Printf("$ nt index-pack %s\n", packfilePath)
	}
	fmt.Println()
	return nil
}
