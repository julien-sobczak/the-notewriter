// +build integration

package e2e_test

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/anki"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/fixtures.apkg
var fixturesAPKG []byte

func TestAnkiImport(t *testing.T) {
	// Write embedded fixtures to a temporary file
	tmpFile, err := os.CreateTemp("", "fixtures-*.apkg")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	
	_, err = tmpFile.Write(fixturesAPKG)
	require.NoError(t, err)
	tmpFile.Close()
	
	apkgSource := tmpFile.Name()
	
	t.Run("BasicImport", func(t *testing.T) {
		testDir := t.TempDir()
		
		// Extract collection
		collection, err := anki.ExtractCollection(apkgSource)
		require.NoError(t, err)
		defer collection.Close()
		
		// Verify collection loaded
		assert.Greater(t, len(collection.Notes), 0)
		assert.Greater(t, len(collection.Cards), 0)
		
		// Create importer with default mapping
		outputPath := filepath.Join(testDir, "output.md")
		importer := &anki.Importer{
			Collection: collection,
			TagMappings: map[string]string{
				"default": outputPath,
			},
		}
		
		// Run import
		err = importer.Import()
		require.NoError(t, err)
		
		// Check output file was created
		assert.FileExists(t, outputPath)
		
		// Read and verify content
		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		
		// Expected content - compare full file content
		expectedContent := "# Output\n\n## Flashcard: Untitled `@cid: 1761228936249` `@slug: anki-1761228936249`\n\n**Front:**\n\nThis file requires a newer version of Anki.\n\n**Back:**\n\nThis file requires a newer version of Anki.\n\n<hr id=answer>\n\n\n\n"
		assert.Equal(t, expectedContent, string(content))
	})
	
	t.Run("ImportWithMediaDir", func(t *testing.T) {
		testDir := t.TempDir()
		
		// Extract collection
		collection, err := anki.ExtractCollection(apkgSource)
		require.NoError(t, err)
		defer collection.Close()
		
		// Create importer with media directory
		outputPath := filepath.Join(testDir, "flashcards.md")
		importer := &anki.Importer{
			Collection: collection,
			TagMappings: map[string]string{
				"default": outputPath,
			},
			MediaDir: "assets",
		}
		
		// Run import
		err = importer.Import()
		require.NoError(t, err)
		
		// Check output file
		assert.FileExists(t, outputPath)
		
		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		contentStr := string(content)
		
		// Verify heading from filename
		assert.Contains(t, contentStr, "# Flashcards")
	})
	
	t.Run("ImportWithSubdirectory", func(t *testing.T) {
		testDir := t.TempDir()
		
		// Extract collection
		collection, err := anki.ExtractCollection(apkgSource)
		require.NoError(t, err)
		defer collection.Close()
		
		// Create importer with subdirectory path
		outputPath := filepath.Join(testDir, "skills/web/general.md")
		importer := &anki.Importer{
			Collection: collection,
			TagMappings: map[string]string{
				"default": outputPath,
			},
		}
		
		// Run import
		err = importer.Import()
		require.NoError(t, err)
		
		// Check output file was created in subdirectory
		assert.FileExists(t, outputPath)
		
		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		contentStr := string(content)
		
		// Verify heading from filename
		assert.Contains(t, contentStr, "# General")
	})
	
	t.Run("AppendToExistingFile", func(t *testing.T) {
		testDir := t.TempDir()
		
		// Create existing file
		outputPath := filepath.Join(testDir, "existing.md")
		existingContent := "# Existing\n\n## Some existing content\n\nHere is some text.\n"
		err := os.WriteFile(outputPath, []byte(existingContent), 0644)
		require.NoError(t, err)
		
		// Extract collection
		collection, err := anki.ExtractCollection(apkgSource)
		require.NoError(t, err)
		defer collection.Close()
		
		// Create importer
		importer := &anki.Importer{
			Collection: collection,
			TagMappings: map[string]string{
				"default": outputPath,
			},
		}
		
		// Run import
		err = importer.Import()
		require.NoError(t, err)
		
		// Read file and verify both old and new content
		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		contentStr := string(content)
		
		// Check existing content is preserved
		assert.Contains(t, contentStr, "## Some existing content")
		assert.Contains(t, contentStr, "Here is some text.")
		
		// Check new content was appended
		assert.Contains(t, contentStr, "## Flashcard:")
		
		// Count occurrences of heading - should only be once
		headingCount := strings.Count(contentStr, "# Existing")
		assert.Equal(t, 1, headingCount, "File heading should appear only once")
	})
	
	t.Run("IgnoreScheduling", func(t *testing.T) {
		testDir := t.TempDir()
		
		// Extract collection
		collection, err := anki.ExtractCollection(apkgSource)
		require.NoError(t, err)
		defer collection.Close()
		
		// Create importer with IgnoreScheduling flag
		outputPath := filepath.Join(testDir, "output.md")
		importer := &anki.Importer{
			Collection: collection,
			TagMappings: map[string]string{
				"default": outputPath,
			},
			IgnoreScheduling: true,
		}
		
		// Run import
		err = importer.Import()
		require.NoError(t, err)
		
		// The test passes if import succeeds
		// In fixtures.apkg, there are no reviews, so there's nothing to ignore
		assert.FileExists(t, outputPath)
	})
	
	t.Run("NoMappingError", func(t *testing.T) {
		// Extract collection
		collection, err := anki.ExtractCollection(apkgSource)
		require.NoError(t, err)
		defer collection.Close()
		
		// Create importer without default mapping
		importer := &anki.Importer{
			Collection: collection,
			TagMappings: map[string]string{
				// No mappings defined
			},
		}
		
		// Run import - should fail with error
		err = importer.Import()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no mapping found")
	})
}
