package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"storj.io/common/testcontext"
	"storj.io/storj/private/testplanet"
	"storj.io/uplink"
)

func TestFSRemote(t *testing.T) {
	origin := t.TempDir()

	r, err := NewFSRemote(origin)
	require.NoError(t, err)

	// Add a file
	err = r.PutObject("info/commit-graph", []byte(`
updated_at: 2023-01-01T01:14:30Z
commits:
	- a757e67f5ae2a8df3a4634c96c16af5c8491bea2
`))
	require.NoError(t, err)

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

func TestStorjRemote(t *testing.T) {
	t.Skip() // TODO The test does not execute the closure...

	// See https://raw.githubusercontent.com/wiki/storj/storj/code/Testing.md
	testplanet.Run(t, testplanet.Config{
		SatelliteCount:   1,
		StorageNodeCount: 1,
		UplinkCount:      1,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		// See https://github.com/storj/uplink/blob/main/testsuite/object_test.go for inspiration
		project := openProject(t, ctx, planet)
		defer project.Close()

		remote, err := NewStorjRemoteFromProject("my-bucket", project)
		require.NoError(t, err)

		// Check bucket has been created
		statBucket, err := project.StatBucket(ctx, "my-bucket")
		require.NoError(t, err)
		require.Equal(t, "my-bucket", statBucket.Name)

		// Check object doesn't already exist
		obj, err := project.StatObject(ctx, "my-bucket", "my-file")
		require.NoError(t, err)
		assert.Equal(t, 10, obj.System.ContentLength)

		// Create it
		err = remote.PutObject("my-file", []byte("Hello World"))
		require.NoError(t, err)
		// Check again
		obj, err = project.StatObject(ctx, "my-bucket", "my-file")
		require.NoError(t, err)
		assert.Equal(t, "my-file", obj.Key)
	})
}

/* Test Helpers */

func openProject(t *testing.T, ctx context.Context, planet *testplanet.Planet) *uplink.Project {
	project, err := planet.Uplinks[0].OpenProject(ctx, planet.Satellites[0])
	require.NoError(t, err)
	return project
}
