package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyFileIfNotExists(t *testing.T) {
	t.Run("CopyNewFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create source file
		srcPath := filepath.Join(tmpDir, "source.txt")
		content := "test content"
		err := os.WriteFile(srcPath, []byte(content), 0644)
		require.NoError(t, err)
		
		// Copy to destination
		dstPath := filepath.Join(tmpDir, "dest.txt")
		err = CopyFileIfNotExists(srcPath, dstPath)
		require.NoError(t, err)
		
		// Verify destination exists and has same content
		dstContent, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, content, string(dstContent))
	})
	
	t.Run("FileAlreadyExists", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create source file
		srcPath := filepath.Join(tmpDir, "source.txt")
		err := os.WriteFile(srcPath, []byte("source content"), 0644)
		require.NoError(t, err)
		
		// Create destination file with different content
		dstPath := filepath.Join(tmpDir, "dest.txt")
		originalContent := "original content"
		err = os.WriteFile(dstPath, []byte(originalContent), 0644)
		require.NoError(t, err)
		
		// Try to copy - should not overwrite
		err = CopyFileIfNotExists(srcPath, dstPath)
		require.NoError(t, err)
		
		// Verify destination still has original content
		dstContent, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, originalContent, string(dstContent))
	})
	
	t.Run("CreatesMissingDirectories", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create source file
		srcPath := filepath.Join(tmpDir, "source.txt")
		content := "test content"
		err := os.WriteFile(srcPath, []byte(content), 0644)
		require.NoError(t, err)
		
		// Copy to nested destination that doesn't exist
		dstPath := filepath.Join(tmpDir, "nested", "dir", "dest.txt")
		err = CopyFileIfNotExists(srcPath, dstPath)
		require.NoError(t, err)
		
		// Verify destination exists
		assert.FileExists(t, dstPath)
		
		// Verify content
		dstContent, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, content, string(dstContent))
	})
	
	t.Run("SourceFileNotFound", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		srcPath := filepath.Join(tmpDir, "nonexistent.txt")
		dstPath := filepath.Join(tmpDir, "dest.txt")
		
		err := CopyFileIfNotExists(srcPath, dstPath)
		assert.Error(t, err)
	})
}
