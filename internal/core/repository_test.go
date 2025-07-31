package core

import (
	"path/filepath"
	"testing"

	"slices"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexFilesFirst(t *testing.T) {
	var tests = []struct {
		name   string
		paths  []string // input
		result []string // output
	}{

		{
			name: "Without index file",
			paths: []string{
				"projects/A/features.md",
				"todo/quarter.md",
				"todo/today.md",
			},
			result: []string{
				"projects/A/features.md",
				"todo/quarter.md",
				"todo/today.md",
			},
		},

		{
			name: "With index.md",
			paths: []string{
				"todo/do.md",
				"todo/index.md",
				"todo/quarter.md",
				"todo/today.md",
			},
			result: []string{
				"todo/index.md", // UP
				"todo/do.md",
				"todo/quarter.md",
				"todo/today.md",
			},
		},

		{
			name: "With INDEX.markdown",
			paths: []string{
				"todo/do.md",
				"todo/INDEX.markdown",
				"todo/quarter.md",
				"todo/today.md",
			},
			result: []string{
				"todo/INDEX.markdown", // UP
				"todo/do.md",
				"todo/quarter.md",
				"todo/today.md",
			},
		},

		{
			name: "With mulitple index.md",
			paths: []string{
				"appendix.md",
				"index.md",
				"references/books/a.md",
				"references/books/index.md",
				"references/index.md",
				"todo/do.md",
				"todo/index.md",
				"todo/quarter.md",
				"todo/today.md",
			},
			result: []string{
				"index.md",
				"appendix.md",
				"references/books/index.md",
				"references/books/a.md",
				"references/index.md",
				"todo/index.md",
				"todo/do.md",
				"todo/quarter.md",
				"todo/today.md",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slices.SortFunc(tt.paths, IndexFilesFirst)
			assert.Equal(t, tt.result, tt.paths)
		})
	}
}

func TestRepositoryType(t *testing.T) {

	t.Run("GetNoteRelativePath", func(t *testing.T) {
		var tests = []struct {
			name                   string // name
			referencePath          string // input
			noteRelativePath       string // input
			repositoryRelativePath string // output
		}{
			{
				name:                   "Same directory",
				referencePath:          "./projects/the-notewriter/todo.md",
				noteRelativePath:       "ideas.md",
				repositoryRelativePath: "projects/the-notewriter/ideas.md",
			},
			{
				name:                   "Medias file",
				referencePath:          "./skills/programming.md",
				noteRelativePath:       "./medias/go.svg",
				repositoryRelativePath: "skills/medias/go.svg",
			},
			{
				name:                   "Move to parent directory",
				referencePath:          "./projects/the-notewriter/todo.md",
				noteRelativePath:       "../../skills/programming.md",
				repositoryRelativePath: "skills/programming.md",
			},
		}

		tr := NewTestRepository(t)
		require.Equal(t, tr.Root, CurrentConfig().RootDirectory)

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				relpath, err := CurrentRepository().GetNoteRelativePath(filepath.Join(tr.Root, tt.referencePath), tt.noteRelativePath)
				require.NoError(t, err)
				assert.Equal(t, tt.repositoryRelativePath, relpath)
			})
		}
	})

	t.Run("GetFileRelativePath", func(t *testing.T) {
		tr := NewTestRepository(t)

		var tests = []struct {
			name             string // name
			fileAbsolutePath string // input
			expected         string // output
		}{
			{
				name:             "File in root directory",
				fileAbsolutePath: filepath.Join(tr.Root, "README.md"),
				expected:         "README.md",
			},
			{
				name:             "File in subdirectory",
				fileAbsolutePath: filepath.Join(tr.Root, "docs/guide.md"),
				expected:         "docs/guide.md",
			},
			{
				name:             "File in nested subdirectory",
				fileAbsolutePath: filepath.Join(tr.Root, "docs/tutorials/go/intro.md"),
				expected:         "docs/tutorials/go/intro.md",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual := CurrentRepository().GetFileRelativePath(tt.fileAbsolutePath)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

	t.Run("GetFileAbsolutePath", func(t *testing.T) {
		tr := NewTestRepository(t)

		var tests = []struct {
			name             string // name
			fileRelativePath string // input
			expected         string // output
		}{
			{
				name:             "File in root directory",
				fileRelativePath: "README.md",
				expected:         filepath.Join(tr.Root, "README.md"),
			},
			{
				name:             "File in subdirectory",
				fileRelativePath: "docs/guide.md",
				expected:         filepath.Join(tr.Root, "/docs/guide.md"),
			},
			{
				name:             "File in nested subdirectory",
				fileRelativePath: "docs/tutorials/go/intro.md",
				expected:         filepath.Join(tr.Root, "docs/tutorials/go/intro.md"),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual := CurrentRepository().GetFileAbsolutePath(tt.fileRelativePath)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

	t.Run("Walk", func(t *testing.T) {
		NewTestRepository(t,
			WithFile("index.md", `# Index`),
			WithFile(".git/index", ""), // Skip
			WithFile("index.md", ""),
			WithFile("skills/medias/go-logo.svg", ""), // Skip
			WithFile("skills/index.md", "# Skills"),
			WithFile("skills/programming/index.md", "# Programming"),
			WithFile("skills/programming/go.md", "# Go"),
			WithFile("skills/drawing.md", "# Drawing"),
			WithFile("projects/the-notewriter.md", "# The NoteWriter"),
			WithFile("projects/ignore.md", "---\ntags: ignore\n---\n# Ignore Me"), // Skip
			WithFile("todo.md", "# TODO"),
		)

		var tests = []struct {
			name      string
			pathSpecs PathSpecs
			expected  []string
		}{
			{
				name:      "All markdown files",
				pathSpecs: AnyPath,
				expected: []string{
					"index.md",
					"projects/the-notewriter.md",
					"skills/index.md", // index.md are processed first (required for inheritance)
					"skills/drawing.md",
					"skills/programming/index.md",
					"skills/programming/go.md",
					"todo.md",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var actual []string
				err := CurrentRepository().Walk(tt.pathSpecs, func(md *markdown.File) error {
					actual = append(actual, CurrentRepository().GetFileRelativePath(md.AbsolutePath))
					return nil
				})
				require.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

}

func TestStatsInDB(t *testing.T) {
	NewTestRepository(t, FromGoldenDirNamed("TestMinimal"))

	stats, err := CurrentRepository().StatsInDB()
	require.NoError(t, err)
	assert.Equal(t, 0, stats.Objects["file"])
	assert.Equal(t, 0, stats.Objects["note"])
	assert.Equal(t, 0, stats.Objects["flashcard"])
	assert.Equal(t, 0, stats.Objects["media"])
	assert.Equal(t, 0, stats.Objects["link"])
	assert.Equal(t, 0, stats.Objects["reminder"])

	_, err = CurrentRepository().Add(AnyPath)
	require.NoError(t, err)

	stats, err = CurrentRepository().StatsInDB()
	require.NoError(t, err)
	assert.Greater(t, stats.Objects["file"], 0)
	assert.Greater(t, stats.Objects["note"], 0)
	assert.Greater(t, stats.Objects["flashcard"], 0)
	assert.Greater(t, stats.Objects["media"], 0)
	assert.Greater(t, stats.Objects["link"], 0)
	assert.Greater(t, stats.Objects["reminder"], 0)

	assert.Equal(t, map[string]int{
		"go":      3,
		"history": 1,
	}, stats.Tags)

	assert.Equal(t, map[string]int{
		"source": 1,
		"tags":   3,
	}, stats.Attributes)
}
