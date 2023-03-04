package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOIDToPath(t *testing.T) {
	var tests = []struct {
		name string // name
		oid  string // input
		path string // output
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
			assert.Equal(t, tt.path, OIDToPath(tt.oid))
		})
	}
}
