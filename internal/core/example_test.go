package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
	t.Skip() // FIXME
	// A basic test to make sure the example directory is valid

	SetUpRepositoryFromGoldenDirNamed(t, "example")

	err := CurrentRepository().Add(".")
	require.NoError(t, err)
	err = CurrentDB().Commit("Initial commit")
	require.NoError(t, err)

	notes, err := CurrentRepository().SearchNotes("kind:artwork @subject:art")
	require.NoError(t, err)
	assert.Greater(t, len(notes), 1)
}
