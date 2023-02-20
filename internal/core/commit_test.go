package core

import (
	"bytes"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reOID matches the Git commit ID format
var reOID = regexp.MustCompile(`\w{40}`)

func TestNewOID(t *testing.T) {
	oid1 := NewOID()
	oid2 := NewOID()
	require.NotEqual(t, oid1, oid2)
	assert.Regexp(t, reOID, oid1)
}

func TestNewOIDFromBytes(t *testing.T) {
	bytes1 := []byte{97, 98, 99, 100, 101, 102}
	bytes2 := []byte{98, 98, 99, 100, 101, 102}
	oid1 := NewOIDFromBytes(bytes1)
	oid2 := NewOIDFromBytes(bytes2)
	require.NotEqual(t, oid1, oid2)
	require.Equal(t, oid1, NewOIDFromBytes(bytes1)) // Does not change
	assert.Regexp(t, reOID, oid1)
}

func TestCommitGraph(t *testing.T) {

	t.Run("New CommitGraph", func(t *testing.T) {
		now := clock.FreezeAt(time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))
		cg := NewCommitGraph()
		assert.Equal(t, now, cg.UpdatedAt)

		// Initial parent must be empty
		err := cg.AppendCommit("a757e67f5ae2a8df3a4634c96c16af5c8491bea2", "invalid parent")
		require.ErrorContains(t, err, "invalid head")

		// A succession of commits
		now = clock.FreezeAt(time.Date(2023, time.Month(1), 1, 1, 14, 30, 0, time.UTC))
		err = cg.AppendCommit("a757e67f5ae2a8df3a4634c96c16af5c8491bea2", "")
		require.NoError(t, err)
		err = cg.AppendCommit("a04d20dec96acfc2f9785802d7e3708721005d5d", "a757e67f5ae2a8df3a4634c96c16af5c8491bea2")
		require.NoError(t, err)
		err = cg.AppendCommit("52d614e255d914e2f6022689617da983381c27a3", "a04d20dec96acfc2f9785802d7e3708721005d5d")
		require.NoError(t, err)
		assert.Equal(t, now, cg.UpdatedAt)

		// Repeat the last commit must fail as head as changed
		err = cg.AppendCommit("52d614e255d914e2f6022689617da983381c27a3", "a04d20dec96acfc2f9785802d7e3708721005d5d")
		require.ErrorContains(t, err, "invalid head")

		_, err = cg.LastCommitsFrom("unknown")
		require.ErrorContains(t, err, "unknown commit")
		commits, err := cg.LastCommitsFrom("a04d20dec96acfc2f9785802d7e3708721005d5d")
		require.NoError(t, err)
		require.EqualValues(t, []string{"52d614e255d914e2f6022689617da983381c27a3"}, commits)

		buf := new(bytes.Buffer)
		err = cg.Write(buf)
		require.NoError(t, err)
		cgYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
updated_at: 2023-01-01T01:14:30Z
commits:
    - a757e67f5ae2a8df3a4634c96c16af5c8491bea2
    - a04d20dec96acfc2f9785802d7e3708721005d5d
    - 52d614e255d914e2f6022689617da983381c27a3
`), strings.TrimSpace(cgYAML))
	})

	t.Run("Existing CommitGraph", func(t *testing.T) {
		in, err := os.CreateTemp("", "commit-graph1")
		require.NoError(t, err)
		defer os.Remove(in.Name())
		out, err := os.CreateTemp("", "commit-graph2")
		require.NoError(t, err)
		defer os.Remove(out.Name())

		in.WriteString(`updated_at: 2023-01-01T01:14:30Z
commits:
    - a757e67f5ae2a8df3a4634c96c16af5c8491bea2
    - a04d20dec96acfc2f9785802d7e3708721005d5d
    - 52d614e255d914e2f6022689617da983381c27a3
`) // Caution: spaces are important as we compare hashes at the end of the test
		in.Close()

		// Read in
		in, err = os.Open(in.Name())
		require.NoError(t, err)
		cg, err := ReadCommitGraph(in)
		in.Close()
		require.NoError(t, err)
		assert.Equal(t, []string{"a757e67f5ae2a8df3a4634c96c16af5c8491bea2", "a04d20dec96acfc2f9785802d7e3708721005d5d", "52d614e255d914e2f6022689617da983381c27a3"}, cg.CommitOIDs)

		// Write out
		err = cg.Write(out)
		require.NoError(t, err)
		out.Close()

		// Files must match
		hashIn, _ := hashFromFile(in.Name())
		hashOut, _ := hashFromFile(out.Name())
		assert.Equal(t, hashIn, hashOut)
	})
}
