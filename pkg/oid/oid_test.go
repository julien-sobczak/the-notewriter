package oid_test

import (
	"regexp"
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reOID matches the Git commit ID format
var reOID = regexp.MustCompile(`\w{40}`)

func TestNewOID(t *testing.T) {
	oid1 := oid.New()
	oid2 := oid.New()
	require.NotEqual(t, oid1, oid2)
	assert.Regexp(t, reOID, oid1)
}

func TestNewOIDFromBytes(t *testing.T) {
	bytes1 := []byte{97, 98, 99, 100, 101, 102}
	bytes2 := []byte{98, 98, 99, 100, 101, 102}
	oid1 := oid.NewFromBytes(bytes1)
	oid2 := oid.NewFromBytes(bytes2)
	require.NotEqual(t, oid1, oid2)
	require.Equal(t, oid1, oid.NewFromBytes(bytes1)) // Does not change
	assert.Regexp(t, reOID, oid1)
}

func TestOID(t *testing.T) {
	t.Run("RelativePath", func(t *testing.T) {
		var tests = []struct {
			name string  // name
			oid  oid.OID // input
			path string  // output
		}{
			{
				"Example 1",
				"f3aaf5433ec0357844d88f860c42e044fe44ee61",
				"f3/f3aaf5433ec0357844d88f860c42e044fe44ee61",
			},
			{
				"Example 2",
				"5bb55dad2b3157a81893bc25f809d85a1fab2911",
				"5b/5bb55dad2b3157a81893bc25f809d85a1fab2911",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.path, tt.oid.RelativePath())
			})
		}
	})

}
