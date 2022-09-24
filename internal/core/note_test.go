package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttributeListFrontMatterString(t *testing.T) {
	var tests = []struct {
		name     string        // name
		input    AttributeList // input
		expected string        // expected result
	}{
		{
			"Scalar values",
			AttributeList{
				&Attribute{
					Name:          "key1",
					Value:         "value1",
					OriginalValue: "'value1'",
				},
				&Attribute{
					Name:          "key2",
					Value:         2,
					OriginalValue: "two",
				},
			},
			`
key1: value1
key2: 2`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tt.input.FrontMatterString()
			require.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(tt.expected), strings.TrimSpace(actual))
		})
	}
}
