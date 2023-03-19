package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobPaths(t *testing.T) {
	var g GlobPaths = []GlobPath{
		"archives/",
		"!archives/index.md",

		"projects/**/*.tmp",
		"projects/*/*.png",

		"/todos/",
		"/todos.md",
	}

	assert.True(t, g.Match("archives/toto/"))
	assert.False(t, g.Match("archives.md"))       // No rule
	assert.False(t, g.Match("archives/index.md")) // Using negation

	assert.False(t, g.Match("myprojects/test.tmp"))       // No rule
	assert.True(t, g.Match("projects/test.tmp"))          // ** matches 0-n directories
	assert.True(t, g.Match("projects/sub/test.tmp"))      // ** matches 0-n directories
	assert.True(t, g.Match("projects/sub/sub/test.tmp"))  // ** matches 0-n directories
	assert.False(t, g.Match("projects/test.png"))         // matches 1 directory
	assert.True(t, g.Match("projects/sub/test.png"))      // matches 1 directory
	assert.False(t, g.Match("projects/sub/sub/test.png")) // matches 1 directory

	assert.False(t, g.Match("sub/todos/index.md")) // no† root directory
	assert.False(t, g.Match("sub/todos.md"))       // not root directory
	assert.True(t, g.Match("todos.md"))            // root
	assert.True(t, g.Match("todos/index.md"))      // root

	ignoreFile := IgnoreFile{Entries: g}
	assert.True(t, ignoreFile.MustExcludeFile("archives/toto", true))
}

func TestReadConfigFromDirectory(t *testing.T) {

	t.Run("Config present", func(t *testing.T) {
		dir := populate(t, map[string]interface{}{

			".nt/config": `
[core]
extensions=["md"]`,

			".nt/lint": `
rules:

# Enforce a minimum number of lines between notes
- name: min-lines-between-notes
  severity: warning # Default to error
  args: [2]

# Forbid untyped notes (must be able to exclude paths)
- name: no-free-note
  includes: # default to root
  - todo/
  - "!todo/misc"
`,

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
		assert.Len(t, c.LintFile.Rules, 2)
		rule1 := c.LintFile.Rules[0]
		rule2 := c.LintFile.Rules[1]
		assert.Equal(t, "min-lines-between-notes", rule1.Name)
		assert.Equal(t, "warning", rule1.Severity)
		assert.EqualValues(t, []string{"2"}, rule1.Args)
		assert.Equal(t, "no-free-note", rule2.Name)
		assert.Equal(t, "", rule2.Severity)
		assert.EqualValues(t, []GlobPath{"todo/", "!todo/misc"}, rule2.Includes)
	})

	t.Run("Config missing", func(t *testing.T) {
		dir := populate(t, map[string]interface{}{
			// missing .nt directory

			".ntignore": `README.md`,

			"journal/today": Symlink("./2022-12-24.md"),
			"journal/2022-12-24.md": `
# 2022-12-24

## Note: Nothing interesting
Blablabla`,
		})

		c, err := ReadConfigFromDirectory(filepath.Join(dir, "journal"))
		require.NoError(t, err)
		require.Nil(t, c)
	})

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
