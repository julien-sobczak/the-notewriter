package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

schemas:
- name: Relations
  attributes:
  - name: source
    inherit: false
    required: true
  - name: references
    type: string[]
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

		// Check .ntignore
		assert.Contains(t, c.IgnoreFile.Entries, GlobPath("README.md"))

		// Check .nt/lint rules
		require.Len(t, c.LintFile.Rules, 2)
		rule1 := c.LintFile.Rules[0]
		rule2 := c.LintFile.Rules[1]
		assert.Equal(t, "min-lines-between-notes", rule1.Name)
		assert.Equal(t, "warning", rule1.Severity)
		assert.EqualValues(t, []string{"2"}, rule1.Args)
		assert.Equal(t, "no-free-note", rule2.Name)
		assert.Equal(t, "", rule2.Severity)
		assert.EqualValues(t, []GlobPath{"todo/", "!todo/misc"}, rule2.Includes)

		// Check .nt/lint schemas
		require.Len(t, c.LintFile.Schemas, 1)
		schemaActual := c.LintFile.Schemas[0]
		schemaExpected := ConfigLintSchema{
			Name: "Relations",
			Attributes: []*ConfigLintSchemaAttribute{
				{
					Name:     "source",
					Type:     "string", // default value
					Inherit:  BoolPointer(false),
					Required: BoolPointer(true),
				},
				{
					Name:     "references",
					Type:     "array",
					Inherit:  BoolPointer(true),  // default value
					Required: BoolPointer(false), // default value
				},
			},
		}
		assert.Equal(t, schemaExpected, schemaActual)
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

	t.Run("Default files", func(t *testing.T) {
		dir := populate(t, map[string]interface{}{
			".nt/config": `
[core]
extensions=["md"]`,

			// No files .ntignore or .nt/lint defined
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)
		require.NotNil(t, c)

		// Check all default entries are present
		iEntry := 0
		for _, line := range strings.Split(DefaultIgnore, "\n") {
			if text.IsBlank(line) || strings.HasPrefix(line, "#") {
				continue
			}
			assert.Equal(t, line, string(c.IgnoreFile.Entries[iEntry]))
			iEntry++
		}

		// Check all default lint rule are present
		var defaultLint = make(map[string]interface{})
		err = yaml.Unmarshal([]byte(DefaultLint), &defaultLint)
		require.NoError(t, err)
		if rulesRaw, ok := defaultLint["rules"]; ok {
			rules := rulesRaw.([]interface{})
			assert.Len(t, c.LintFile.Rules, len(rules))
		} else {
			assert.Empty(t, c.LintFile.Rules)
		}
		// Check all default lint schemas are present
		if schemasRaw, ok := defaultLint["schemas"]; ok {
			schemas := schemasRaw.([]interface{})
			assert.Len(t, c.LintFile.Schemas, len(schemas))
		} else {
			assert.Empty(t, c.LintFile.Schemas)
		}
	})

}

func TestCheckConfig(t *testing.T) {

	t.Run("Unknown Lint Rule", func(t *testing.T) {
		dir := populate(t, map[string]interface{}{

			".nt/lint": `
rules:

- name: unknown-rule
  severity: warning
`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)

		err = c.Check()
		require.ErrorContains(t, err, "unknown lint rule")
	})

	t.Run("Invalid severity", func(t *testing.T) {
		dir := populate(t, map[string]interface{}{

			".nt/lint": `
rules:

- name: check-attribute
  severity: info
`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)

		err = c.Check()
		require.ErrorContains(t, err, "unknown severity")
	})

	t.Run("Conflicting schema types", func(t *testing.T) {
		dir := populate(t, map[string]interface{}{

			".nt/lint": `
schemas:

- name: Books
  attributes:
  - name: title
    type: string

- name: Persons
  attributes:
  - name: title
    type: string[]
`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)

		err = c.Check()
		require.ErrorContains(t, err, "conflicting type for attribute")
	})

	t.Run("Invalid pattern in schema", func(t *testing.T) {
		dir := populate(t, map[string]interface{}{

			".nt/lint": `
schemas:

- name: Books
  attributes:
  - name: isbn
    type: string
    pattern: "(\\d{10,13"
`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)

		err = c.Check()
		require.ErrorContains(t, err, "invalid pattern")
	})

	t.Run("Invalid .nt/config", func(t *testing.T) {
		tests := []struct {
			name             string
			config           string
			expectedError    string
			additionalChecks func(*testing.T, *Config)
		}{

			{
				name: "Invalid template in references",
				config: `
[reference.books]
title = "A book"
manager = "google-books"
path = """references/books/test.md"""
template = """---
title: "{{index . "title" | title
---
"""
`,
				expectedError: "invalid template for reference",
			},

			{
				name: "Invalid path in references",
				config: `
[reference.books]
title = "A book"
manager = "google-books"
path = """references/books/{{.md"""
template = """# {{index . "title" | title }}"""
`,
				expectedError: "invalid path for reference",
			},

			{
				name: "Supported SRS algorithm",
				config: `
[deck.general]
name = "General"
algorithm = "stone"
`,
				expectedError: "unsupported SRS algorithm",
			},

			{
				name: "Deck attributes",
				config: `
[deck.life]
name = "Life"
query = "path:skills"
newFlashcardsPerDay = 10
algorithmSettings.easeFactor = 3.1
`,
				additionalChecks: func(t *testing.T, c *Config) {
					require.Len(t, c.ConfigFile.Deck, 1)
					deck := c.ConfigFile.Deck["life"]

					// Check specified attributes
					assert.Equal(t, "Life", deck.Name)
					assert.Equal(t, 10, deck.NewFlashcardsPerDay)
					assert.Equal(t, "path:skills", deck.Query)

					// Check defaults
					assert.Equal(t, DefaultSRSBoostFactor, deck.BoostFactor)
					assert.Equal(t, DefaultSRSAlgorithm, deck.Algorithm)

					// Check nested attributes are correctly parsed
					assert.Equal(t, map[string]any{
						"easeFactor": 3.1,
					}, deck.AlgorithmSettings)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				dir := populate(t, map[string]interface{}{
					".nt/config": tt.config,
				})

				c, err := ReadConfigFromDirectory(dir)
				if err != nil {
					if tt.expectedError == "" {
						require.NoError(t, err)
					} else {
						require.ErrorContains(t, err, tt.expectedError)
					}
					t.Skip()
				}

				err = c.Check()
				if tt.expectedError == "" {
					assert.NoError(t, err)
				} else {
					require.ErrorContains(t, err, tt.expectedError)
				}

				if tt.additionalChecks != nil {
					tt.additionalChecks(t, c)
				}
			})
		}

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
