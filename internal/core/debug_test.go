package core_test

import (
	"os"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/require"
)

func TestPersonalNotes(t *testing.T) {
	if os.Getenv("VSCODE_GO_TEST") == "" {
		// Special variable used to avoid running this kind of test from terminal
		// Add in `.vscode/settings.json`:
		//   "go.testEnvVars": {
		//     "VSCODE_GO_TEST": "1"
		//   }
		t.Skip()
	}

	originalHome := os.Getenv("NT_HOME")
	os.Setenv("NT_HOME", "/path/to/notes")
	t.Cleanup(func() {
		os.Setenv("NT_HOME", originalHome)
	})

	t.Logf("Reading notes from %s", core.CurrentConfig().RootDirectory)

	filepath := "/path/to/notes/file.md"
	md, err := markdown.ParseFile(filepath)
	require.NoError(t, err)
	file, err := core.ParseFile(md, nil)
	require.NoError(t, err)
	require.NotNil(t, file)
}
