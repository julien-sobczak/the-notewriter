package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-jsonnet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathSpecs(t *testing.T) {

	t.Run(".ntignore", func(t *testing.T) {
		var g PathSpecs = []PathSpec{
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
	})

	t.Run("Args", func(t *testing.T) {
		assert.True(t, AnyPath.MatchAll())
		assert.True(t, PathSpecs{"."}.MatchAll())
		assert.True(t, PathSpecs{"projects/", "."}.MatchAll())
		// But not if one path spec is excluded
		assert.False(t, PathSpecs{"projects/", ".", "!todos"}.MatchAll())
		// Not if . is missing
		assert.False(t, PathSpecs{"projects/", "todos"}.MatchAll())
	})

}

func TestInitConfiguration(t *testing.T) {
	dir := populate(t, map[string]interface{}{
		// missing .nt directory
		"journal/2022-12-24.md": `# Blablabla`,
	})

	c, err := InitConfigFromDirectory(dir, DefaultConfigOptions)
	require.NoError(t, err)
	require.NotNil(t, c)

	// Check generated files
	require.FileExists(t, filepath.Join(dir, ".nt", "config.jsonnet"))
	require.FileExists(t, filepath.Join(dir, ".nt", "nt.libsonnet"))
	require.FileExists(t, filepath.Join(dir, ".ntignore"))
}

func TestReadConfigFromDirectory(t *testing.T) {

	t.Run("Config present", func(t *testing.T) {
		dir := populate(t, map[string]interface{}{

			".nt/config.jsonnet": `
{
    Core: {
        extensions: ["md"]
    }
}`,

			".ntignore": `README.md`,

			"journal/today": Symlink("./2022-12-24.md"),
			"journal/2022-12-24.md": `
# 2022-12-24

## Note: Nothing interesting
Blablabla`,
		})

		c1, err := ReadConfigFromDirectory(filepath.Join(dir, "journal"))
		require.NoError(t, err)
		require.NotNil(t, c1)

		c2, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)
		require.NotNil(t, c2)

		// Check .ntignore
		assert.Equal(t, c1.ConfigFile, c2.ConfigFile)
		assert.Equal(t, c1.IgnoreFile, c2.IgnoreFile)
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

func TestCheckConfig(t *testing.T) {

	t.Run("Unknown Lint Rule", func(t *testing.T) {
		dir := populate(t, map[string]interface{}{

			".nt/config.jsonnet": `
{
	Linter: {
        Rules: [
			{
                name: "unknown-rule",
				severity: "warning",
            },
		]
	},
}`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)
		require.NotNil(t, c)

		err = c.Check()
		require.ErrorContains(t, err, "unknown lint rule")
	})

	t.Run("Invalid severity", func(t *testing.T) {
		dir := populate(t, map[string]interface{}{

			".nt/config.jsonnet": `
{
	Linter: {
        Rules: [
			{
                name: "no-duplicate-slug",
				severity: "oops",
            },
		]
	},
}`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)

		err = c.Check()
		require.ErrorContains(t, err, "unknown severity")
	})

	t.Run("Invalid pattern in schema", func(t *testing.T) {
		dir := populate(t, map[string]interface{}{

			".nt/config.jsonnet": `
{
    Attributes: {
        mydate: {
            name: "isbn",
            type: "string",
            pattern: "(\\d{10,13",
        },
	},
}
`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)

		err = c.Check()
		require.ErrorContains(t, err, "invalid pattern")
	})

	t.Run("Invalid .nt/config.jsonnet", func(t *testing.T) {
		tests := []struct {
			name             string
			config           string
			expectedError    string
			additionalChecks func(*testing.T, *Config)
		}{

			{
				name: "Invalid template in references",
				config: `
{
	References: [
		{
			title: "A book",
			manager: "google-books",
			path: 'references/books/test.md',
			template: |||
				---
				title: "{{index . "title" | title
				---
			|||
		},
	],
}
`,
				expectedError: "invalid template for reference",
			},

			{
				name: "Invalid path in references",
				config: `
{
	References: [
		{
			title: "A book",
			manager: "google-books",
			path: 'references/books/{{.md',
			template: |||
				# {{index . "title" | title }}
			|||
		},
	],
}
`,
				expectedError: "invalid path for reference",
			},

			{
				name: "Deck attributes",
				config: `
{
	Decks: [
		{
			name: "Life",
			query: "path:skills",
			newFlashcardsPerDay: 10,
			algorithmSettings: {
				easeFactor: 3.1,
			},
		},
	],
}`,
				additionalChecks: func(t *testing.T, c *Config) {
					require.Len(t, c.ConfigFile.Decks, 1)
					deck := c.ConfigFile.Decks[0]

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
					".nt/config.jsonnet": tt.config,
				})

				c, err := ReadConfigFromDirectory(dir)
				if err != nil {
					if tt.expectedError == "" {
						require.NoError(t, err)
					} else {
						require.ErrorContains(t, err, tt.expectedError)
					}
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

// Learning test
// See https://github.com/google/go-jsonnet
func TestJsonnet(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		// This example simply try to convert a Jsonnet snippet to JSON
		vm := jsonnet.MakeVM()

		snippet := `{
		person1: {
		    name: "Alice",
		    welcome: "Hello " + self.name + "!",
		},
		person2: self.person1 { name: "Bob" },
	}`

		actualJSON, err := vm.EvaluateAnonymousSnippet("example.jsonnet", snippet)
		require.NoError(t, err)

		expectedJSON := `{
   "person1": {
      "name": "Alice",
      "welcome": "Hello Alice!"
   },
   "person2": {
      "name": "Bob",
      "welcome": "Hello Bob!"
   }
}
`
		assert.Equal(t, expectedJSON, actualJSON)
	})

	t.Run("Vargs", func(t *testing.T) {
		// This example illustrates how arrays with different values are passed
		vm := jsonnet.MakeVM()

		snippet := `{
	args: [2, "Alice", true],
}`

		actualJSON, err := vm.EvaluateAnonymousSnippet("example.jsonnet", snippet)
		require.NoError(t, err)

		expectedJSON := `{
   "args": [
      2,
      "Alice",
      true
   ]
}
`
		assert.Equal(t, expectedJSON, actualJSON)

		type Example struct {
			Args []any `json:"args"`
		}
		var example Example
		err = json.NewDecoder(strings.NewReader(expectedJSON)).Decode(&example)
		require.NoError(t, err)
		assert.Equal(t, []any{float64(2), "Alice", true}, example.Args)
	})
}

func TestStaticConfigFiles(t *testing.T) {
	// Copy static files to temporary directory
	dir := t.TempDir()
	err := InitConfigFileFromDirectory(dir, DefaultConfigOptions)
	require.NoError(t, err)

	// Evaluate generated configuration file
	vm := jsonnet.MakeVM()
	actual, err := vm.EvaluateFile(filepath.Join(dir, ".nt", "config.jsonnet"))
	require.NoError(t, err)
	t.Log(actual)
}
