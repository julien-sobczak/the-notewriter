package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-jsonnet"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
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
	t.Run("With custom media converter", func(t *testing.T) {
		dir := populate(t, map[string]any{
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

		// Check that config includes core block when MediaConverter is set
		configContent, err := os.ReadFile(filepath.Join(dir, ".nt", "config.jsonnet"))
		require.NoError(t, err)
		assert.Contains(t, string(configContent), "core:")
		assert.Contains(t, string(configContent), "ffmpeg")
	})

	t.Run("Without media converter (minimal config)", func(t *testing.T) {
		dir := populate(t, map[string]any{
			// missing .nt directory
			"journal/2022-12-24.md": `# Blablabla`,
		})

		c, err := InitConfigFromDirectory(dir, ConfigOptions{
			MediaConverter: "",
		})
		require.NoError(t, err)
		require.NotNil(t, c)

		// Check generated files
		require.FileExists(t, filepath.Join(dir, ".nt", "config.jsonnet"))
		require.FileExists(t, filepath.Join(dir, ".nt", "nt.libsonnet"))
		require.FileExists(t, filepath.Join(dir, ".ntignore"))

		// Check that config doesn't include core block when MediaConverter is empty
		configContent, err := os.ReadFile(filepath.Join(dir, ".nt", "config.jsonnet"))
		require.NoError(t, err)
		assert.NotContains(t, string(configContent), "core:")
		assert.Contains(t, string(configContent), "attributes: nt.DefaultAttributes")
		assert.Contains(t, string(configContent), "noteTypes: nt.DefaultNoteTypes")

		// Verify that ffmpeg is still used as default
		assert.Equal(t, "ffmpeg", c.ConfigFile.Core.Medias.Command)
		assert.Equal(t, 1, c.ConfigFile.Core.Medias.Parallel)
		assert.Equal(t, "ultrafast", c.ConfigFile.Core.Medias.Preset)
	})
}

func TestReadConfigFromDirectory(t *testing.T) {

	t.Run("Config present", func(t *testing.T) {
		dir := populate(t, map[string]any{

			".nt/config.jsonnet": `
{
    core: {
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
		dir := populate(t, map[string]any{
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
		dir := populate(t, map[string]any{

			".nt/config.jsonnet": `
{
	linter: {
        rules: [
			{
                name: "unknown-rule",
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

	t.Run("Invalid query in queries", func(t *testing.T) {
		dir := populate(t, map[string]any{

			".nt/config.jsonnet": `
{
	queries: {
		myQuery: {
			title: "My Query",
			query: "#",
		},
	},
}`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)

		err = c.Check()
		require.ErrorContains(t, err, "invalid query \"myQuery\"")
	})

	t.Run("Invalid query in deck", func(t *testing.T) {
		dir := populate(t, map[string]any{

			".nt/config.jsonnet": `
{
	decks: [
		{
			name: "Test Deck",
			query: "#",
		},
	],
}`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)

		err = c.Check()
		require.ErrorContains(t, err, "invalid query \"#\"")
		require.ErrorContains(t, err, "invalid deck \"Test Deck\"")
	})

	t.Run("Invalid query in desk", func(t *testing.T) {
		dir := populate(t, map[string]any{

			".nt/config.jsonnet": `
{
	desks: [
		{
			name: "Test Desk",
			root: {
				layout: "container",
				query: "#",
			},
		},
	],
}`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)

		err = c.Check()
		require.ErrorContains(t, err, "invalid desk \"Test Desk\"")
	})

	t.Run("Invalid query in nested desk block", func(t *testing.T) {
		dir := populate(t, map[string]any{

			".nt/config.jsonnet": `
{
	desks: [
		{
			name: "Complex Desk",
			root: {
				layout: "vertical",
				elements: [
					{
						name: "Block 1",
						query: "type:Note",
					},
					{
						layout: "horizontal",
						elements: [
							{
								name: "Block 2",
								query: "#",
							},
						],
					},
				],
			},
		},
	],
}`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)

		err = c.Check()
		require.ErrorContains(t, err, "invalid query \"#\"")
		require.ErrorContains(t, err, "invalid desk \"Complex Desk\"")
	})

	t.Run("Invalid query in stat", func(t *testing.T) {
		dir := populate(t, map[string]any{

			".nt/config.jsonnet": `
{
	stats: [
		{
			name: "Test Stat",
			query: "#",
			groupBy: "date",
			visualization: "pie",
		},
	],
}`,
		})

		c, err := ReadConfigFromDirectory(dir)
		require.NoError(t, err)

		err = c.Check()
		require.ErrorContains(t, err, "invalid query \"#\"")
		require.ErrorContains(t, err, "invalid stat \"Test Stat\"")
	})

	t.Run("Invalid pattern in schema", func(t *testing.T) {
		dir := populate(t, map[string]any{

			".nt/config.jsonnet": `
{
    attributes: {
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
	references: [
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
	references: [
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
	decks: [
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
				dir := populate(t, map[string]any{
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

func populate(t *testing.T, files map[string]any) string {
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

func TestParseConfigFile(t *testing.T) {

	t.Run("Valid minimal config", func(t *testing.T) {
		configPath := MustWriteTempFile(t, "config.jsonnet", `{
	core: {
		extensions: ["md"]
	}
}`)

		cfg, err := ParseConfigFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Defined properties should be read
		assert.Equal(t, []string{"md"}, cfg.Core.Extensions)
		// Reserved attributes should be present
		for k := range ReservedAttributes {
			_, ok := cfg.Attributes[k]
			assert.True(t, ok, "reserved attribute %q missing", k)
		}
		// Defaults for Medias should be set
		assert.Equal(t, "ffmpeg", cfg.Core.Medias.Command)
		assert.Equal(t, 1, cfg.Core.Medias.Parallel)
		assert.Equal(t, "ultrafast", cfg.Core.Medias.Preset)
	})

	t.Run("Config with attributes and noteTypes", func(t *testing.T) {
		configPath := MustWriteTempFile(t, "config.jsonnet", `
{
	Attributes: {
		myattr: {
			name: "myattr",
			type: "string",
		},
	},
	noteTypes: {
		Note: {
			name: "Note",
		},
	},
}`)

		cfg, err := ParseConfigFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Custom attribute should be present
		attr, ok := cfg.Attributes["myattr"]
		require.True(t, ok)
		assert.Equal(t, "myattr", attr.Name)
		assert.Equal(t, "string", attr.Type)
		// Should set Inherit to true by default
		require.NotNil(t, attr.Inherit)
		assert.True(t, *attr.Inherit)

		// Custom type should be present
		noteType, ok := cfg.NoteTypes["Note"]
		require.True(t, ok)
		assert.Equal(t, "Note", noteType.Name)
		// Type pattern should be set by default
		assert.Contains(t, noteType.Pattern, "^Note:")
	})

	t.Run("Config with file types", func(t *testing.T) {
		configPath := MustWriteTempFile(t, "config.jsonnet", `
{
	fileTypes: {
		Reading: {
			name: "Reading",
			processors: ["toc"],
		},
	},
}`)

		cfg, err := ParseConfigFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Custom file type should be present
		fileType, ok := cfg.FileTypes["Reading"]
		require.True(t, ok)
		assert.Equal(t, "Reading", fileType.Name)
		assert.Equal(t, []string{"toc"}, fileType.Processors)
		// File type pattern should be set by default
		assert.Contains(t, fileType.Pattern, "^Reading:")
	})

	t.Run("Config with invalid jsonnet", func(t *testing.T) {
		configPath := MustWriteTempFile(t, "config.jsonnet", `{ invalid: [ }`)

		_, err := ParseConfigFile(configPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to evaluate config.jsonnet")
	})

	t.Run("Config with queries using tags", func(t *testing.T) {
		configPath := MustWriteTempFile(t, "config.jsonnet", `
{
	queries: {
		myQuery: {
			title: "My Query",
			query: "type:Note",
			tags: ["tag1", "tag2"],
		},
	},
}`)

		cfg, err := ParseConfigFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Check queries
		require.NotNil(t, cfg.Queries)
		query, ok := cfg.Queries["myQuery"]
		require.True(t, ok)
		assert.Equal(t, "My Query", query.Title)
		assert.Equal(t, "type:Note", query.Query)
		assert.Equal(t, []string{"tag1", "tag2"}, query.Tags)
	})

	t.Run("Config with desk", func(t *testing.T) {
		configPath := MustWriteTempFile(t, "config.jsonnet", `
{
	desks: [
		{
			name: "My Desk",
			description: "A test desk",
			root: {
				layout: "container",
				query: "type:Note",
			},
		},
	],
}`)

		cfg, err := ParseConfigFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Check desks
		require.Len(t, cfg.Desks, 1)
		desk := cfg.Desks[0]
		assert.Equal(t, "My Desk", desk.Name)
		assert.Equal(t, "A test desk", desk.Description)
		assert.Equal(t, "container", desk.Root.Layout)
		assert.Equal(t, "type:Note", desk.Root.Query)
	})

	t.Run("Config with a complex desk", func(t *testing.T) {
		configPath := MustWriteTempFile(t, "config.jsonnet", `
{
	desks: [
		{
			name: "My Project",
			root: {
				layout: "vertical",
				elements: [
					{
						name: "Notes",
						size: "70%",
						query: "path:projects",
					},
					{
						layout: "horizontal",
						elements: [
							{
								name: "Backlog",
								query: "type:Todo",
								view: "single",
								size: "30%",
							},
							{
								name: "Features",
								query: "type:Feature",
							},
						],
					},
				],
			},
		},
	],
}`)

		cfg, err := ParseConfigFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Check desks
		require.Len(t, cfg.Desks, 1)
		desk := cfg.Desks[0]
		assert.Equal(t, "My Project", desk.Name)
		assert.Equal(t, "vertical", desk.Root.Layout)
		require.Len(t, desk.Root.Elements, 2)

		// First element
		assert.Equal(t, "Notes", desk.Root.Elements[0].Name)
		assert.Equal(t, "70%", desk.Root.Elements[0].Size)
		assert.Equal(t, "path:projects", desk.Root.Elements[0].Query)

		// Second element (horizontal layout)
		assert.Equal(t, "horizontal", desk.Root.Elements[1].Layout)
		require.Len(t, desk.Root.Elements[1].Elements, 2)
		assert.Equal(t, "Backlog", desk.Root.Elements[1].Elements[0].Name)
		assert.Equal(t, "single", desk.Root.Elements[1].Elements[0].View)
		assert.Equal(t, "30%", desk.Root.Elements[1].Elements[0].Size)
	})

	t.Run("Config with a journal", func(t *testing.T) {
		configPath := MustWriteTempFile(t, "config.jsonnet", `
{
	journals: [
		{
			name: "My Diary",
			path: "journal/${year}/${year}-${month}-${day}.md",
			defaultContent: "# Journal: ${year}-${month}-${day}",
			routines: [
				{
					name: "Morning Routine",
					template: "# Good morning",
				},
				{
					name: "Shutdown Routine",
					template: "# Good night",
				},
			],
		},
	],
}`)

		cfg, err := ParseConfigFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Check journals
		require.Len(t, cfg.Journals, 1)
		journal := cfg.Journals[0]
		assert.Equal(t, "My Diary", journal.Name)
		assert.Equal(t, "journal/${year}/${year}-${month}-${day}.md", journal.Path)
		assert.Equal(t, "# Journal: ${year}-${month}-${day}", journal.DefaultContent)
		require.Len(t, journal.Routines, 2)
		assert.Equal(t, "Morning Routine", journal.Routines[0].Name)
		assert.Equal(t, "# Good morning", journal.Routines[0].Template)
		assert.Equal(t, "Shutdown Routine", journal.Routines[1].Name)
		assert.Equal(t, "# Good night", journal.Routines[1].Template)
	})

	t.Run("Config with stats", func(t *testing.T) {
		configPath := MustWriteTempFile(t, "config.jsonnet", `
{
	stats: [
		{
			name: "Quotes by nationality",
			query: "type:Quote",
			groupBy: "nationality",
			visualization: "pie",
		},
		{
			name: "Steps by day",
			query: "type:Journal",
			groupBy: "date",
			value: "steps",
			visualization: "calendar",
		},
		{
			name: "World Inspiration",
			query: "type:Quote",
			groupBy: "nationality",
			visualization: "map",
			mapping: {
				"Roman": "ITA",
				"Greek": "GRC",
			},
		},
	],
}`)

		cfg, err := ParseConfigFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Check stats
		require.Len(t, cfg.Stats, 3)

		// First stat - pie chart
		stat1 := cfg.Stats[0]
		assert.Equal(t, "Quotes by nationality", stat1.Name)
		assert.Equal(t, "type:Quote", stat1.Query)
		assert.Equal(t, "nationality", stat1.GroupBy)
		assert.Equal(t, "pie", stat1.Visualization)

		// Second stat - calendar
		stat2 := cfg.Stats[1]
		assert.Equal(t, "Steps by day", stat2.Name)
		assert.Equal(t, "date", stat2.GroupBy)
		assert.Equal(t, "steps", stat2.Value)
		assert.Equal(t, "calendar", stat2.Visualization)

		// Third stat - map with mapping
		stat3 := cfg.Stats[2]
		assert.Equal(t, "World Inspiration", stat3.Name)
		assert.Equal(t, "map", stat3.Visualization)
		require.NotNil(t, stat3.Mapping)
		assert.Equal(t, "ITA", stat3.Mapping["Roman"])
		assert.Equal(t, "GRC", stat3.Mapping["Greek"])
	})

}

func TestConfigFile(t *testing.T) {

	t.Run("GetNoteType", func(t *testing.T) {
		cfg := &ConfigFile{
			NoteTypes: ConfigNoteTypes{
				"Note": &ConfigNoteType{Name: "Note"},
			},
		}

		noteType, ok := cfg.GetNoteType("Note")
		assert.True(t, ok)
		require.NotNil(t, noteType)
		assert.Equal(t, "Note", noteType.Name)

		unknownType, ok := cfg.GetNoteType("Unknown")
		assert.False(t, ok)
		assert.Nil(t, unknownType)
	})

	t.Run("GetAttribute", func(t *testing.T) {
		cfg := &ConfigFile{
			Attributes: ConfigAttributes{
				"myattr": &ConfigAttribute{Name: "myattr"},
			},
		}

		attr, ok := cfg.GetAttribute("myattr")
		assert.True(t, ok)
		require.NotNil(t, attr)
		assert.Equal(t, "myattr", attr.Name)

		unknownAttr, ok := cfg.GetAttribute("unknown")
		assert.False(t, ok)
		assert.Nil(t, unknownAttr)
	})

	t.Run("GetAttributeDefaults", func(t *testing.T) {
		cfg := &ConfigFile{
			Attributes: ConfigAttributes{
				"attr1": &ConfigAttribute{
					Name:         "attr1",
					Type:         "string",
					DefaultValue: "value1",
				},
				"attr2": &ConfigAttribute{
					Name: "attr2",
					Type: "string",
					// no default value
				},
				"attr3": &ConfigAttribute{
					Name:         "attr3",
					Type:         "integer",
					DefaultValue: 42,
				},
			},
			NoteTypes: ConfigNoteTypes{
				"Note1": &ConfigNoteType{
					Name: "Note1",
					Attributes: []ConfigTypeAttribute{
						{Name: "attr1", Required: BoolPointer(true)},
						{Name: "attr2", Required: BoolPointer(false)},
						{Name: "attr3", Required: BoolPointer(false)}, // not required, default doesn't apply
					},
				},
				"Note2": &ConfigNoteType{
					Name: "Note2",
					Attributes: []ConfigTypeAttribute{
						{Name: "attr1", Required: BoolPointer(true)},
						{Name: "attr2", Required: BoolPointer(true)},
						{Name: "attr3", Required: BoolPointer(true)},
					},
				},
			},
		}

		defaultsNote1 := cfg.GetAttributeDefaults("Note1")
		require.Equal(t, AttributeSet(map[string]any{
			"attr1": "value1",
		}), defaultsNote1)

		defaultsNote2 := cfg.GetAttributeDefaults("Note2")
		require.Equal(t, AttributeSet(map[string]any{
			"attr1": "value1",
			"attr3": int64(42),
		}), defaultsNote2)
	})

	t.Run("GetFileType", func(t *testing.T) {
		cfg := &ConfigFile{
			FileTypes: ConfigFileTypes{
				"Reading": &ConfigFileType{Name: "Reading"},
			},
		}

		fileType, ok := cfg.GetFileType("Reading")
		assert.True(t, ok)
		require.NotNil(t, fileType)
		assert.Equal(t, "Reading", fileType.Name)

		unknownType, ok := cfg.GetFileType("Unknown")
		assert.False(t, ok)
		assert.Nil(t, unknownType)
	})

	t.Run("MatchFileType", func(t *testing.T) {
		cfg := &ConfigFile{
			FileTypes: ConfigFileTypes{
				"Reading": &ConfigFileType{
					Name:    "Reading",
					Pattern: "(?i)^Reading:\\s*(.*)$",
				},
			},
		}

		fileType, title, ok := cfg.MatchFileType("Reading: My Book Title")
		assert.True(t, ok)
		require.NotNil(t, fileType)
		assert.Equal(t, "Reading", fileType.Name)
		assert.Equal(t, markdown.Document("My Book Title"), title)

		fileType, title, ok = cfg.MatchFileType("reading: lowercase title")
		assert.True(t, ok)
		require.NotNil(t, fileType)
		assert.Equal(t, markdown.Document("lowercase title"), title)

		fileType, title, ok = cfg.MatchFileType("Not a Reading")
		assert.False(t, ok)
		assert.Nil(t, fileType)
		assert.Equal(t, markdown.Document("Not a Reading"), title)
	})

}

/* Test Helpers */

// MustWriteTempFile creates a temporary file with the given name and content.
// It doesn't required a valid directory structure, unlike 'MustWriteFile' function.
func MustWriteTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}
