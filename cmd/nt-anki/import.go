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
	staged           bool
	mediaDir         string
)

var importCmd = &cobra.Command{
	Use:   "import [apkg-file]",
	Short: "Import Anki flashcards from an .apkg file",
	Long: `Import Anki flashcards from an .apkg file.

The .apkg file is a ZIP archive containing:
- collection.anki2: SQLite database with notes and cards
- media: JSON file mapping media IDs to filenames
- Numbered files (0, 1, etc.): Media files

Example usage:
  nt-anki import col.apkg "web:skills/web/general.md" --staged
  nt-anki import col.apkg --ignore-scheduling --media-dir="medias"`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().BoolVar(&ignoreScheduling, "ignore-scheduling", false, "Ignore scheduling information from revlog table")
	importCmd.Flags().BoolVar(&staged, "staged", false, "Stage packfiles in repository index")
	importCmd.Flags().StringVar(&mediaDir, "media-dir", "", "Subdirectory for media files (relative to output file)")
}

func runImport(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("please provide the path to the .apkg file and the output markdown file")
	}

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
	if err := collection.Import(markdownPath,
		anki.WithMediaDir(mediaDir),
		anki.WithIgnoreScheduling(ignoreScheduling),
		anki.WithStaged(staged)); err != nil {
		return fmt.Errorf("failed to import: %w", err)
	}

	fmt.Println("✅ Import completed successfully")
	return nil
}
