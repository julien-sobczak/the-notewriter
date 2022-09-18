package core

import (
	"testing"
)

func TestCompactYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Basic syntax",
			input: `
parent:
  - key1: value1
    key2: value2
`,
			expected: `
parent:
- key1: value1
  key2: value2
`,
		},
		{
			name: "Inner syntax",
			input: `
parent:
  child:
    - key1: value1
      key2: value2
`,
			expected: `
parent:
  child:
  - key1: value1
    key2: value2
`,
		},
		{
			name: "Other properties behind",
			input: `
parent:
  child1:
    - key1: value1
      key2: value2
  child2: value3
`,
			expected: `
parent:
  child1:
  - key1: value1
    key2: value2
  child2: value3
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := CompactYAML(tt.input)
			if actual != tt.expected {
				t.Errorf("Difference found. Got: \n%v", actual)
			}
		})
	}
}
