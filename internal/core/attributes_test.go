package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDiffKeys(t *testing.T) {
	var tests = []struct {
		name     string
		a        map[string]interface{}
		b        map[string]interface{}
		expected []string
	}{
		{
			name: "Basic",
			a: map[string]interface{}{
				"1": "toto",
				"2": []string{"toto"},
				"3": 3,
				"4": "OK",
			},
			b: map[string]interface{}{
				// 1 is missing
				"2": "toto", // different type
				"3": "3",    // different type
				"4": "OK",
			},
			expected: []string{"1", "2", "3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := DiffKeys(tt.a, tt.b)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestIsXXX(t *testing.T) {
	input := `
bool: true
integer: 10
float: 1.50
string: "This is a string"
flat_array_with_primitive_values: ["value1", "value2"]
array_with_primitive_values:
- value1
- value2
composite_array:
  - key: key1
  - key: key2
object:
  key: name
`
	data := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(input), &data)
	require.NoError(t, err)

	assert.True(t, IsBool(data["bool"]))
	assert.True(t, IsNumber(data["integer"]))
	assert.True(t, IsNumber(data["float"]))
	assert.True(t, IsString(data["string"]))
	assert.True(t, IsArray(data["flat_array_with_primitive_values"]))
	assert.True(t, IsArray(data["array_with_primitive_values"]))
	assert.True(t, IsArray(data["composite_array"]))
	assert.True(t, IsObject(data["object"]))
}

func TestCastAttributes(t *testing.T) {
	SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	// assertions
	types := GetSchemaAttributeTypes()
	require.Equal(t, "array", types["tags"])
	require.Equal(t, "string", types["isbn"])
	require.Equal(t, "array", types["references"])

	actual := CastAttributes(map[string]interface{}{
		"tags":       "favorite",    // must be converted to an array
		"isbn":       9780807014271, // must be converted to a string
		"references": []interface{}{"a book"},
	})
	expected := map[string]interface{}{
		"tags":       []string{"favorite"},
		"isbn":       "9780807014271",
		"references": []string{"a book"},
	}
	assert.Equal(t, expected, actual)
}
