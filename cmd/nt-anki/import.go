package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/julien-sobczak/the-notewriter/pkg/anki"
	"github.com/spf13/cobra"
)

var (
	mappings         []string
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
  nt-anki import col.apkg -m "web:skills/web/general.md" -m "default:unclassified.md" --staged
  nt-anki import col.apkg --ignore-scheduling --media-dir="medias"`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().StringSliceVarP(&mappings, "mapping", "m", nil, "Tag to file mapping (format: tag:filepath)")
	importCmd.Flags().BoolVar(&ignoreScheduling, "ignore-scheduling", false, "Ignore scheduling information from revlog table")
	importCmd.Flags().BoolVar(&staged, "staged", false, "Stage packfiles in repository index")
	importCmd.Flags().StringVar(&mediaDir, "media-dir", "", "Subdirectory for media files (relative to output file)")
}

func runImport(cmd *cobra.Command, args []string) error {
	apkgPath := args[0]

	// Validate the file exists
	if _, err := os.Stat(apkgPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", apkgPath)
	}

	// Parse mappings
	tagMappings, err := parseMappings(mappings)
	if err != nil {
		return err
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
	collection.Import(
		anki.WithTagMappings(tagMappings),
		anki.WithMediaDir(mediaDir),
		anki.WithIgnoreScheduling(ignoreScheduling),
		anki.WithStaged(staged))
	if err := collection.Import(); err != nil {
		return fmt.Errorf("failed to import: %w", err)
	}

	fmt.Println("✅ Import completed successfully")
	return nil
}

// parseMappings parses the mapping flags into a map
func parseMappings(mappings []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, mapping := range mappings {
		parts := strings.SplitN(mapping, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid mapping format: %s (expected tag:filepath)", mapping)
		}
		tag := strings.TrimSpace(parts[0])
		filepath := strings.TrimSpace(parts[1])
		if tag == "" || filepath == "" {
			return nil, fmt.Errorf("invalid mapping format: %s (tag and filepath cannot be empty)", mapping)
		}
		result[tag] = filepath
	}
	return result, nil
}
