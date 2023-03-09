package helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHash(t *testing.T) {
	// Same content = same hash
	assert.Equal(t, Hash([]byte("same")), Hash([]byte("same")))
	// Different contents = different hashes
	assert.NotEqual(t, Hash([]byte("same")), Hash([]byte("different")))
}

func TestHashFromFile(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "file1")
	file2 := filepath.Join(dir, "file2")
	file3 := filepath.Join(dir, "file3")

	err := os.WriteFile(file1, []byte("Hello"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("Hello"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(file3, []byte("Bonjour"), 0644)
	require.NoError(t, err)

	hash1, err := HashFromFile(file1)
	require.NoError(t, err)
	hash2, err := HashFromFile(file2)
	require.NoError(t, err)
	hash3, err := HashFromFile(file3)
	require.NoError(t, err)

	// Same file = same hash
	assert.Equal(t, hash1, hash1)
	// Same file content = same hash
	assert.Equal(t, hash1, hash2)
	// Different file coontent = different hashes
	assert.NotEqual(t, hash1, hash3)
}

func TestHashFromFileName(t *testing.T) {
	// Same filename = same hash
	assert.Equal(t, HashFromFileName("/tmp/file1.go"), HashFromFileName("/tmp/file1.go"))
	// Different filenames = different hashes
	assert.NotEqual(t, HashFromFileName("/tmp/file1.go"), HashFromFileName("/tmp/file2.go"))
}
