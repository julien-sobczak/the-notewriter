// +build integration

package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnkiImport(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	
	// Find the fixtures.apkg file (in repository root)
	repoRoot := ".."
	apkgSource := filepath.Join(repoRoot, "fixtures.apkg")
	
	// Verify source file exists
	_, err := os.Stat(apkgSource)
	require.NoError(t, err, "fixtures.apkg not found in repository root")
	
	// Copy to temp directory for tests
	apkgPath := filepath.Join(tmpDir, "fixtures.apkg")
	sourceData, err := os.ReadFile(apkgSource)
	require.NoError(t, err)
	err = os.WriteFile(apkgPath, sourceData, 0644)
	require.NoError(t, err)
	
	// Build the ntanki binary
	binaryPath := filepath.Join(tmpDir, "ntanki")
	repoRootAbs, err := filepath.Abs(repoRoot)
	require.NoError(t, err)
	
	buildCmd := exec.Command("go", "build", "--tags", "fts5", "-o", binaryPath, "./cmd/nt-anki")
	buildCmd.Dir = repoRootAbs
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", output)
	}
	require.NoError(t, err, "Failed to build ntanki binary")
	
	t.Run("BasicImport", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "basic")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)
		
		// Run import with default mapping
		cmd := exec.Command(binaryPath, "import", apkgPath, "-m", "default:output.md")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		t.Logf("Import output: %s", output)
		require.NoError(t, err)
		
		// Check output file was created
		outputPath := filepath.Join(testDir, "output.md")
		assert.FileExists(t, outputPath)
		
		// Read and verify content
		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		contentStr := string(content)
		
		// Verify heading
		assert.Contains(t, contentStr, "# Output")
		
		// Verify flashcard structure
		assert.Contains(t, contentStr, "## Flashcard:")
		assert.Contains(t, contentStr, "`@cid:")
		assert.Contains(t, contentStr, "`@slug: anki-")
		
		// Verify front/back sections
		assert.Contains(t, contentStr, "**Front:**")
		assert.Contains(t, contentStr, "**Back:**")
	})
	
	t.Run("ImportWithMediaDir", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "media")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)
		
		// Run import with media directory
		cmd := exec.Command(binaryPath, "import", apkgPath, "-m", "default:flashcards.md", "--media-dir=assets")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		t.Logf("Import output: %s", output)
		require.NoError(t, err)
		
		// Check output file
		outputPath := filepath.Join(testDir, "flashcards.md")
		assert.FileExists(t, outputPath)
		
		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		contentStr := string(content)
		
		// Verify heading from filename
		assert.Contains(t, contentStr, "# Flashcards")
	})
	
	t.Run("ImportWithSubdirectory", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "subdir")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)
		
		// Run import with subdirectory path
		cmd := exec.Command(binaryPath, "import", apkgPath, "-m", "default:skills/web/general.md")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		t.Logf("Import output: %s", output)
		require.NoError(t, err)
		
		// Check output file was created in subdirectory
		outputPath := filepath.Join(testDir, "skills/web/general.md")
		assert.FileExists(t, outputPath)
		
		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		contentStr := string(content)
		
		// Verify heading from filename
		assert.Contains(t, contentStr, "# General")
	})
	
	t.Run("AppendToExistingFile", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "append")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)
		
		// Create existing file
		outputPath := filepath.Join(testDir, "existing.md")
		existingContent := "# Existing\n\n## Some existing content\n\nHere is some text.\n"
		err = os.WriteFile(outputPath, []byte(existingContent), 0644)
		require.NoError(t, err)
		
		// Run import
		cmd := exec.Command(binaryPath, "import", apkgPath, "-m", "default:existing.md")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		t.Logf("Import output: %s", output)
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
		testDir := filepath.Join(tmpDir, "scheduling")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)
		
		// Run import with --ignore-scheduling flag
		cmd := exec.Command(binaryPath, "import", apkgPath, "-m", "default:output.md", "--ignore-scheduling")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		t.Logf("Import output: %s", output)
		require.NoError(t, err)
		
		// The test passes if import succeeds
		// In fixtures.apkg, there are no reviews, so there's nothing to ignore
		outputPath := filepath.Join(testDir, "output.md")
		assert.FileExists(t, outputPath)
	})
}

func TestAnkiImportErrors(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Find the fixtures.apkg file
	repoRoot := ".."
	apkgSource := filepath.Join(repoRoot, "fixtures.apkg")
	sourceData, err := os.ReadFile(apkgSource)
	require.NoError(t, err)
	
	// Build the binary
	binaryPath := filepath.Join(tmpDir, "ntanki")
	repoRoot2 := ".."
	repoRootAbs2, err := filepath.Abs(repoRoot2)
	require.NoError(t, err)
	
	buildCmd := exec.Command("go", "build", "--tags", "fts5", "-o", binaryPath, "./cmd/nt-anki")
	buildCmd.Dir = repoRootAbs2
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", output)
	}
	require.NoError(t, err, "Failed to build ntanki binary")
	
	t.Run("FileNotFound", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "notfound")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)
		
		// Try to import non-existent file
		cmd := exec.Command(binaryPath, "import", "nonexistent.apkg", "-m", "default:output.md")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		
		// Should fail
		assert.Error(t, err)
		assert.Contains(t, string(output), "file not found")
	})
	
	t.Run("InvalidMapping", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "invalid")
		err := os.MkdirAll(testDir, 0755)
		require.NoError(t, err)
		
		apkgPath := filepath.Join(tmpDir, "fixtures.apkg")
		err = os.WriteFile(apkgPath, sourceData, 0644)
		require.NoError(t, err)
		
		// Try with invalid mapping format
		cmd := exec.Command(binaryPath, "import", apkgPath, "-m", "invalid-format")
		cmd.Dir = testDir
		output, err := cmd.CombinedOutput()
		
		// Should fail
		assert.Error(t, err)
		assert.Contains(t, string(output), "invalid mapping format")
	})
}
