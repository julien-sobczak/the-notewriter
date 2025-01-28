package core

import (
	"strings"
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestTagSet(t *testing.T) {

	t.Run("NewTagSet", func(t *testing.T) {
		var tests = []struct {
			name     string // name
			inputA   TagSet
			inputB   TagSet
			expected TagSet
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
				actual := NewTagSet(tt.inputA).Merge(tt.inputB)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

}

func TestCastFn(t *testing.T) {

	t.Run("CastStringFn", func(t *testing.T) {
		v, ok := CastStringFn("string")
		// String is OK
		assert.True(t, ok)
		assert.Equal(t, "string", v)

		v, ok = CastStringFn("")
		assert.True(t, ok)
		assert.Equal(t, "", v)

		// Primitive types are OK
		v, ok = CastStringFn(false)
		assert.True(t, ok)
		assert.Equal(t, "false", v)

		v, ok = CastStringFn(10)
		assert.True(t, ok)
		assert.Equal(t, "10", v)

		v, ok = CastStringFn(10)
		assert.True(t, ok)
		assert.Equal(t, "10", v)

		// Other types are KO
		_, ok = CastStringFn(map[string]any{"key": "value"})
		assert.False(t, ok)
		_, ok = CastStringFn(struct{ id int }{id: 1})
		assert.False(t, ok)
	})

	t.Run("CastObjectFn", func(t *testing.T) {
		v, ok := CastObjectFn(map[string]any{"key": "value"})
		assert.True(t, ok)
		assert.Equal(t, map[string]any{"key": "value"}, v)
		v, ok = CastObjectFn(struct{ id int }{id: 1})
		assert.True(t, ok)
		assert.NotNil(t, v)

		// Other types cannot be casted
		_, ok = CastObjectFn("test")
		assert.False(t, ok)
		_, ok = CastObjectFn(10)
		assert.False(t, ok)
	})

	t.Run("CastIntegerFn", func(t *testing.T) {
		// Integers are OK
		v, ok := CastIntegerFn(int(10))
		assert.True(t, ok)
		assert.Equal(t, int64(10), v)
		v, ok = CastIntegerFn(int8(10))
		assert.True(t, ok)
		assert.Equal(t, int64(10), v)
		v, ok = CastIntegerFn(int16(10))
		assert.True(t, ok)
		assert.Equal(t, int64(10), v)
		v, ok = CastIntegerFn(int32(10))
		assert.True(t, ok)
		assert.Equal(t, int64(10), v)
		v, ok = CastIntegerFn(int64(10))
		assert.True(t, ok)
		assert.Equal(t, int64(10), v)
		v, ok = CastIntegerFn(uint(10))
		assert.True(t, ok)
		assert.Equal(t, int64(10), v)

		// String are OK if integer
		v, ok = CastIntegerFn("10")
		assert.True(t, ok)
		assert.Equal(t, int64(10), v)

		_, ok = CastIntegerFn("10.0")
		assert.False(t, ok)
		_, ok = CastIntegerFn("not an integer")
		assert.False(t, ok)
	})

	t.Run("CastFloatFn", func(t *testing.T) {
		// Floats are KO
		v, ok := CastFloatFn(float32(10.0))
		assert.True(t, ok)
		assert.Equal(t, float64(10), v)
		v, ok = CastFloatFn(float64(10.0))
		assert.True(t, ok)
		assert.Equal(t, float64(10), v)

		// Integer are OK
		v, ok = CastFloatFn(10)
		assert.True(t, ok)
		assert.Equal(t, float64(10), v)

		// Strings are OK if integer or float
		v, ok = CastFloatFn("10")
		assert.True(t, ok)
		assert.Equal(t, float64(10), v)
		v, ok = CastFloatFn("10.0")
		assert.True(t, ok)
		assert.Equal(t, float64(10), v)
		_, ok = CastFloatFn("invalid")
		assert.False(t, ok)
	})

	t.Run("CastBoolFn", func(t *testing.T) {
		// Booleans are OK
		v, ok := CastBoolFn(true)
		assert.True(t, ok)
		assert.Equal(t, true, v)
		v, ok = CastBoolFn(false)
		assert.True(t, ok)
		assert.Equal(t, false, v)

		// Strings are OK if true|false
		v, ok = CastBoolFn("true")
		assert.True(t, ok)
		assert.Equal(t, true, v)
		v, ok = CastBoolFn("false")
		assert.True(t, ok)
		assert.Equal(t, false, v)
		_, ok = CastBoolFn("vrai")
		assert.False(t, ok)
	})

	t.Run("CastDateFn", func(t *testing.T) {
		// dates are OK
		v, ok := CastDateFn("2024-12-31")
		assert.True(t, ok)
		assert.NotZero(t, v)
		assert.Equal(t, time.Date(2024, time.Month(12), 31, 0, 0, 0, 0, time.UTC), v)

		// datetimes are OK
		v, ok = CastDateFn("2024-12-31 12:32:00")
		assert.True(t, ok)
		assert.NotZero(t, v)
		assert.Equal(t, time.Date(2024, time.Month(12), 31, 12, 32, 0, 0, time.UTC), v)

		// RFC datetimes are OK
		v, ok = CastDateFn("2024-12-31T12:32:00Z")
		assert.True(t, ok)
		assert.NotZero(t, v)
		assert.Equal(t, time.Date(2024, time.Month(12), 31, 12, 32, 0, 0, time.UTC), v)

		// RFC with a different timezone datetimes are OK
		v, ok = CastDateFn("2024-12-31T12:32:00-05:00")
		assert.True(t, ok)
		assert.NotZero(t, v)
		assert.Equal(t, time.Date(2024, time.Month(12), 31, 17, 32, 0, 0, time.UTC), v.In(time.UTC))
	})

}

func TestAttributeSet(t *testing.T) {

	t.Run("NewAttributeSetFromYAML", func(t *testing.T) {
		frontMatter := `
title: "A notebook"
tags: favorite
`
		actual, err := NewAttributeSetFromYAML(frontMatter)
		require.NoError(t, err)
		expected := AttributeSet(map[string]any{
			"title": "A notebook",
			"tags":  "favorite",
		})
		assert.Equal(t, expected, actual)
	})

	t.Run("DiffKeys", func(t *testing.T) {
		var tests = []struct {
			name     string
			a        AttributeSet
			b        AttributeSet
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
				actual := tt.a.DiffKeys(tt.b)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

	t.Run("Merge", func(t *testing.T) {
		var tests = []struct {
			name     string // name
			inputA   AttributeSet
			inputB   AttributeSet
			expected AttributeSet
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
				actual := tt.inputA.Merge(tt.inputB)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

	t.Run("ToJSON", func(t *testing.T) {
		attributes := AttributeSet(map[string]any{
			"key1": 10,
			"key2": []string{"value1", "value2"},
			"key3": map[string]any{
				"subkey1": 1.5,
				"subkey2": true,
			},
		})
		actual, err := attributes.ToJSON()
		require.NoError(t, err)
		expected := `
{
  "key1": 10,
  "key2": [
    "value1",
    "value2"
  ],
  "key3": {
    "subkey1": 1.5,
    "subkey2": true
  }
}
`
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("ToYAML", func(t *testing.T) {
		attributes := AttributeSet(map[string]any{
			"key1": 10,
			"key2": []string{"value1", "value2"},
			"key3": map[string]any{
				"subkey1": 1.5,
				"subkey2": true,
			},
		})
		actual, err := attributes.ToYAML()
		require.NoError(t, err)
		expected := `
key1: 10
key2:
  - value1
  - value2
key3:
  subkey1: 1.5
  subkey2: true
`
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(actual))
	})

	t.Run("Cast", func(t *testing.T) {
		attributes := AttributeSet(map[string]any{
			"key1": 10,
			"key2": []any{"value1", "value2"},
			"key3": 15.5,
			"key4": "single",
		})

		schemaCompliant := map[string]string{
			"key1": "string",
			"key2": "string[]",
			"key3": "integer",
			"key4": "string[]",
		}
		schemaNonCompliant := map[string]string{
			"key1": "boolean", // Not possible
			"key2": "string[]",
			"key3": "integer",
			"key4": "string[]",
		}

		// Attributes are casted if possible
		actual, err := attributes.Cast(schemaCompliant)
		require.NoError(t, err)
		expected := AttributeSet(map[string]any{
			"key1": "10",
			"key2": []string{"value1", "value2"},
			"key3": int64(15),
			"key4": []string{"single"},
		})
		assert.Equal(t, expected, actual)

		// Errors are returned when not possible
		actual, err = attributes.Cast(schemaNonCompliant)
		require.ErrorContains(t, err, "invalid value")
		assert.Nil(t, actual)

		// Errors can be ignored
		actual = attributes.CastOrIgnore(schemaNonCompliant)
		expected = AttributeSet(map[string]any{
			// key1 is missing to avoid working with a wrong type in queries
			"key2": []string{"value1", "value2"},
			"key3": int64(15),
			"key4": []string{"single"},
		})
		assert.Equal(t, expected, actual)
	})

}

func TestMarkdownAttributes(t *testing.T) {

	t.Run("StripBlockTagsAndAttributes", func(t *testing.T) {
		tests := []struct {
			name     string
			md       markdown.Document // input
			expected markdown.Document // output
		}{
			{
				name: "Basic",
				md: "" +
					"`#favorite` `#life-changing`\n" +
					"`@isbn: 0671244221`\n" +
					"\n" +
					"My note\n",
				expected: "My note",
			},
			{
				name: "Code Block",
				md: "" +
					"```go" +
					"fmt.Println(`Hello`)\n" +
					"```\n",
				expected: "" +
					"```go" +
					"fmt.Println(`Hello`)\n" +
					"```",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual := StripBlockTagsAndAttributes(tt.md)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

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

/* Learning Tests */

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
