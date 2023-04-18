package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
	// A basic test to make sure the example directory is valid

	SetUpCollectionFromGoldenDirNamed(t, "example")

	err := CurrentCollection().Add(".")
	require.NoError(t, err)
	err = CurrentDB().Commit("Initial commit")
	require.NoError(t, err)

	notes, err := CurrentCollection().SearchNotes("kind:artwork @subject:art")
	require.NoError(t, err)
	assert.Greater(t, len(notes), 1)
}
