package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrentConfigPresent(t *testing.T) {
	dir := populate(t, map[string]interface{}{

		".nt/config": `
[core]
extensions=["md"]`,

		".ntignore": `README.md`,

		"journal/today": Symlink("./2022-12-24.md"),
		"journal/2022-12-24.md": `
# 2022-12-24

## Note: Nothing interesting
Blablabla`,
	})

	c, err := ReadConfigFromDirectory(filepath.Join(dir, "journal"))
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Contains(t, c.IgnoreFile.Entries, GlobPath("README.md"))

	c, err = ReadConfigFromDirectory(dir)
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Contains(t, c.IgnoreFile.Entries, GlobPath("README.md"))
}

func TestCurrentConfigMissing(t *testing.T) {
	dir := populate(t, map[string]interface{}{
		// missing .nt directory

		".ntignore": `README.md`,

		"journal/today": Symlink("./2022-12-24.md"),
		"journal/2022-12-24.md": `
# 2022-12-24

## Note: Nothing interesting
Blablabla`,
	})

	// DEBUG WHY infinite loop
	c, err := ReadConfigFromDirectory(filepath.Join(dir, "journal"))
	require.NoError(t, err)
	require.Nil(t, c)
}

func TestInitConfiguration(t *testing.T) {
	dir := populate(t, map[string]interface{}{
		// missing .nt directory
		"journal/2022-12-24.md": `# Blablabla`,
	})

	c, err := InitConfigFromDirectory(dir)
	require.NoError(t, err)
	require.NotNil(t, c)

	// Check generated files
	b, err := os.ReadFile(filepath.Join(dir, ".nt", "config"))
	require.NoError(t, err)
	assert.Equal(t, string(b), DefaultConfig)
	b, err = os.ReadFile(filepath.Join(dir, ".ntignore"))
	require.NoError(t, err)
	assert.Equal(t, string(b), DefaultIgnore)
}

/* Test Helpers */

func populate(t *testing.T, files map[string]interface{}) string {
	dir := t.TempDir()

	for relpath, content := range files {
		dirpath := filepath.Join(dir, filepath.Dir(relpath))
		err := os.MkdirAll(dirpath, 0755)
		require.NoError(t, err)

		abspath := filepath.Join(dir, relpath)
		switch v := content.(type) {
		case Symlink:
			t.Logf("Create symlink %s", abspath)
			os.Symlink(string(v), abspath)
		case string:
			t.Logf("Create text file %s", abspath)
			os.WriteFile(abspath, []byte(v), 0644)
		default:
			t.Errorf("Invalid file type: %v", v)
		}
	}

	return dir
}

type Symlink string
