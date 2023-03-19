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

	notes, err := SearchNotes("kind:artwork @subject:art")
	// BUG: SELECT note_fts.rowid FROM note_fts JOIN note on note.oid = note_fts.oid WHERE note.oid IS NOT NULL AND note.kind IN ("artwork") AND (   json_extract(note.attributes_json, '$.subject') = 'art' ) ORDER BY rank LIMIT 10;
	// The file attributes are not persisted in notes attributes...
	// TODO FIXME rework attributes management
	require.NoError(t, err)
	assert.Greater(t, len(notes), 1)
}
