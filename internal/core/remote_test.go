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
	r.PutObject("info/commit-graph", []byte(`
updated_at: 2023-01-01T01:14:30Z
commits:
	- a757e67f5ae2a8df3a4634c96c16af5c8491bea2
`))

	// Read the wrong file
	_, err = r.GetObject("commit-graph")
	require.Error(t, err)

	// Read the correct file
	data, err := r.GetObject("info/commit-graph")
	require.NoError(t, err)
	require.Equal(t, []byte(`
updated_at: 2023-01-01T01:14:30Z
commits:
	- a757e67f5ae2a8df3a4634c96c16af5c8491bea2
`), data)

	// Update the file
	r.PutObject("info/commit-graph", []byte(`
updated_at: 2023-01-01T01:14:30Z
commits:
	- a757e67f5ae2a8df3a4634c96c16af5c8491bea2
	- a04d20dec96acfc2f9785802d7e3708721005d5d
`))
	// Reread the file
	data, err = r.GetObject("info/commit-graph")
	require.NoError(t, err)
	require.Equal(t, []byte(`
updated_at: 2023-01-01T01:14:30Z
commits:
	- a757e67f5ae2a8df3a4634c96c16af5c8491bea2
	- a04d20dec96acfc2f9785802d7e3708721005d5d
`), data)

	// Delete the file
	err = r.DeleteObject("info/commit-graph")
	require.NoError(t, err)

	// Delete a missing file
	err = r.DeleteObject("info/commit-graph")
	require.Error(t, err)
}
