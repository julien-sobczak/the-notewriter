package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFSRemote(t *testing.T) {
	origin := t.TempDir()

	r, err := NewFSRemote(origin)
	require.NoError(t, err)

	// Add a file
	err = r.PutObject("index", []byte(`
committed_at: 2023-01-01T01:14:30Z
`))
	require.NoError(t, err)

	// Read the wrong file
	_, err = r.GetObject("info/index")
	require.Error(t, err)

	// Read the correct file
	data, err := r.GetObject("index")
	require.NoError(t, err)
	require.Equal(t, []byte(`
committed_at: 2023-01-01T01:14:30Z
`), data)

	// Update the file
	r.PutObject("index", []byte(`
committed_at: 2023-11-11T11:14:30Z
`))
	// Reread the file
	data, err = r.GetObject("index")
	require.NoError(t, err)
	require.Equal(t, []byte(`
committed_at: 2023-11-11T11:14:30Z
`), data)

	// Delete the file
	err = r.DeleteObject("index")
	require.NoError(t, err)

	// Delete a missing file
	err = r.DeleteObject("index")
	require.Error(t, err)
}
