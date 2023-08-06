package core

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegression(t *testing.T) {
	root := SetUpCollectionFromGoldenDirNamed(t, "TestComplex")

	err := CurrentCollection().Add(".")
	require.NoError(t, err)

	err = CurrentDB().Commit("initial commit")
	require.NoError(t, err)

	currentStats, err := CurrentCollection().Stats()
	require.NoError(t, err)

	editLine := &UpdateLine{
		RelativePath: "syntax.md",
		Line:         31,
		Old:          "Example:",
		New:          "Example(s):",
	}
	revertLine := &UpdateLine{
		RelativePath: "syntax.md",
		Line:         31,
		Old:          "Example(s):",
		New:          "Example:",
	}

	var editions []*Edition
	for i := 0; i <= 10; i++ {
		// Apply and revert successively the same change
		// "Nothing" must have changed
		change := editLine
		if i%2 == 1 {
			change = revertLine
		}
		editions = append(editions, &Edition{
			Changes: []Change{change},
			RunGC:   true,
			Check: func(t *testing.T, last, current *Stats) {
				// A new commit file
				assert.Equal(t, last.OnDisk.Commits + 1, current.OnDisk.Commits)
				// No new OIDs
				assert.Equal(t, last.OnDisk.IndexObjects, current.OnDisk.IndexObjects)
				// No new blobs
				assert.Equal(t, last.OnDisk.Blobs, current.OnDisk.Blobs)

				// No new line in DB
				assert.Equal(t, last.InDB.Objects, current.InDB.Objects)
				assert.Equal(t, last.InDB.Attributes, current.InDB.Attributes)
				assert.Equal(t, last.InDB.Tags, current.InDB.Tags)
				assert.Equal(t, last.InDB.Kinds, current.InDB.Kinds)
			},
		})
	}

	for i, edition := range editions {
		fmt.Printf("Applying edition %d/%d...\n", i+1, len(editions))

		for _, change := range edition.Changes {
			fmt.Printf("\tApplying change %q...\n", change)
			change.Apply(t, root)
		}

		err := CurrentCollection().Add(".")
		require.NoError(t, err)

		err = CurrentDB().Commit("commit")
		require.NoError(t, err)

		if edition.RunGC {
			err = CurrentDB().GC()
			require.NoError(t, err)
		}

		lastStats := currentStats
		currentStats, err = CurrentCollection().Stats()
		require.NoError(t, err)
		edition.Check(t, lastStats, currentStats)
	}

	fmt.Println("ici")
}

/* Test Helpers */

type Edition struct {
	Changes []Change
	RunGC   bool
	Check   func(*testing.T, *Stats, *Stats)
}

type Change interface {
	Apply(*testing.T, string)
}

type UpdateLine struct {
	RelativePath string
	Line         int
	Old          string
	New          string
}

func (c *UpdateLine) Apply(t *testing.T, root string) {
	ReplaceLine(t, filepath.Join(root, c.RelativePath), c.Line, c.Old, c.New)
}

func (c UpdateLine) String() string {
	return fmt.Sprintf("Update line %d in %s", c.Line, c.RelativePath)
}

type AppendContent struct {
	RelativePath string
	Content      string
}

func (c *AppendContent) Apply(t *testing.T, root string) {
	AppendLines(t, filepath.Join(root, c.RelativePath), c.Content)
}

func (c AppendContent) String() string {
	newLines := len(strings.Split(c.Content, "\n"))
	return fmt.Sprintf("Append %d line(s) in %s", newLines, c.RelativePath)
}
