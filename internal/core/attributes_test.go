package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMergeTags(t *testing.T) {
	var tests = []struct {
		name     string // name
		inputA   []string
		inputB   []string
		expected []string
	}{
		{
			"empty slices",
			nil,
			nil,
			nil,
		},
		{
			"empty slice",
			[]string{"favorite"},
			nil,
			[]string{"favorite"},
		},
		{
			"single value",
			[]string{"favorite"},
			[]string{"life-changing"},
			[]string{"favorite", "life-changing"},
		},
		{
			"multiple values",
			[]string{"a", "b"},
			[]string{"c", "d"},
			[]string{"a", "b", "c", "d"},
		},
		{
			"duplicates",
			[]string{"a", "b"},
			[]string{"b", "c"},
			[]string{"a", "b", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := mergeTags(tt.inputA, tt.inputB)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestMergeAttributes(t *testing.T) {
	var tests = []struct {
		name     string // name
		inputA   map[string]interface{}
		inputB   map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil maps",
			inputA:   nil,
			inputB:   nil,
			expected: nil,
		},
		{
			name: "append in slices",
			inputA: map[string]interface{}{
				"tags": []string{"a", "b"},
			},
			inputB: map[string]interface{}{
				"tags": "c", // Must not happen
			},
			expected: map[string]interface{}{
				"tags": "c", // Last definition wins
			},
		},
		{
			name: "append in slices",
			inputA: map[string]interface{}{
				"tags": []interface{}{"a", "b"},
			},
			inputB: map[string]interface{}{
				"tags": []interface{}{"c", "d"},
			},
			expected: map[string]interface{}{
				"tags": []interface{}{"a", "b", "c", "d"},
			},
		},
		{
			name: "override basic value",
			inputA: map[string]interface{}{
				"tags": "a",
			},
			inputB: map[string]interface{}{
				"tags": "b",
			},
			expected: map[string]interface{}{
				"tags": "b",
			},
		},
		{
			name: "add new keys",
			inputA: map[string]interface{}{
				"a": "a",
			},
			inputB: map[string]interface{}{
				"b": "b",
			},
			expected: map[string]interface{}{
				"a": "a",
				"b": "b",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := MergeAttributes(tt.inputA, tt.inputB)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

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

func TestCastAttributesOld(t *testing.T) {
	SetUpCollectionFromGoldenDirNamed(t, "TestLint")

	// assertions
	types := GetSchemaAttributeTypes()
	require.Equal(t, "array", types["tags"])
	require.Equal(t, "string", types["isbn"])
	require.Equal(t, "array", types["references"])

	actual := CastAttributesOld(map[string]interface{}{
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

func TestYAMLListWithDifferentTypes(t *testing.T) {
	// Learning test to ensure a YAML list can contains variables of different types.
	input := `
key:
- 10
- string
- true
`
	data := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(input), data)
	require.NoError(t, err)
	assert.Len(t, data["key"], 3)
	assert.IsType(t, []interface{}{}, data["key"])
	values := data["key"].([]interface{})
	assert.Equal(t, 10, values[0])
	assert.Equal(t, "string", values[1])
	assert.Equal(t, true, values[2])
}

func TestYAMLUnmarshallOnInlineAttributes(t *testing.T) {
	// Learning test to ensure YAML Unmarshall returns the correct variable type
	// when parsing inline attributes (in the same way we would parse them
	// if present in the Front Matter of a file).
	tests := []struct {
		input string // Important: Use "key" as key name
		value interface{}
	}{
		{
			"@key: http://google.com",
			"http://google.com",
		},
		{
			"@key: 10",
			10,
		},
		{
			"@key: 3.14",
			3.14,
		},
		{
			"@key: true",
			true,
		},
		{
			"@key: false",
			false,
		},
	}
	for _, tt := range tests {
		yamlDoc := strings.TrimPrefix(tt.input, "@")
		data := make(map[string]interface{})
		err := yaml.Unmarshal([]byte(yamlDoc), &data)
		require.NoError(t, err)
		assert.Equal(t, tt.value, data["key"])
	}
}

func TestCastAttributes(t *testing.T) {
	types := map[string]string{
		"tags":        "array",
		"isbn":        "string",
		"references":  "array",
		"ease-factor": "number",
		"lapses":      "number",
	}

	actual := CastAttributes(map[string]interface{}{
		"tags":        "favorite",    // must be converted to an array
		"isbn":        9780807014271, // must be converted to a string
		"references":  []interface{}{"a book"},
		"ease-factor": "2.5",
		"lapses":      "10",
	}, types)
	expected := map[string]interface{}{
		"tags":        []interface{}{"favorite"},
		"isbn":        "9780807014271",
		"references":  []interface{}{"a book"},
		"ease-factor": float64(2.5),
		"lapses":      int64(10),
	}
	assert.Equal(t, expected, actual)
}
