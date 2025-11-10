package core

import (
	"strings"
	"testing"

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
				NewEmptyTagSet(),
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

	t.Run("ToMarkdownNotation", func(t *testing.T) {
		// Create a test repository for context
		tr := NewTestRepository(t)
		_ = tr // Use the repository to initialize context

		tags := TagSet{"favorite", "must-read"}
		result := tags.ToMarkdownNotation()
		expected := " `#favorite` `#must-read`"
		assert.Equal(t, expected, result)
	})

	t.Run("ToMarkdownNotation empty", func(t *testing.T) {
		// Create a test repository for context
		tr := NewTestRepository(t)
		_ = tr // Use the repository to initialize context

		tags := TagSet{}
		result := tags.ToMarkdownNotation()
		assert.Equal(t, "", result)
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
		assert.Equal(t, "2024-12-31", v)

		// datetimes are OK
		v, ok = CastDateFn("2024-12-31 12:32:00")
		assert.True(t, ok)
		assert.NotZero(t, v)
		assert.Equal(t, "2024-12-31 12:32:00", v)

		// RFC datetimes are OK
		v, ok = CastDateFn("2024-12-31T12:32:00Z")
		assert.True(t, ok)
		assert.NotZero(t, v)
		assert.Equal(t, "2024-12-31T12:32:00Z", v)

		// RFC with a different timezone datetimes are OK
		v, ok = CastDateFn("2024-12-31T12:32:00-05:00")
		assert.True(t, ok)
		assert.NotZero(t, v)
		assert.Equal(t, "2024-12-31T12:32:00-05:00", v)

		// Year and month are OK
		v, ok = CastDateFn("2024-12")
		assert.True(t, ok)
		assert.NotZero(t, v)
		assert.Equal(t, "2024-12", v)

		// Year is OK
		v, ok = CastDateFn("2024")
		assert.True(t, ok)
		assert.NotZero(t, v)
		assert.Equal(t, "2024", v)
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
				a: map[string]any{
					"1": "toto",
					"2": []string{"toto"},
					"3": 3,
					"4": "OK",
				},
				b: map[string]any{
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
				expected: NewEmptyAttributeSet(),
			},
			{
				name: "append in slices",
				inputA: map[string]any{
					"tags": []any{"a", "b"},
				},
				inputB: map[string]any{
					"tags": []any{"c", "d"},
				},
				expected: map[string]any{
					"tags": []any{"a", "b", "c", "d"},
				},
			},
			{
				name: "override basic value",
				inputA: map[string]any{
					"tags": "a",
				},
				inputB: map[string]any{
					"tags": "b",
				},
				expected: map[string]any{
					"tags": "b",
				},
			},
			{
				name: "add new keys",
				inputA: map[string]any{
					"a": "a",
				},
				inputB: map[string]any{
					"b": "b",
				},
				expected: map[string]any{
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
		NewTestRepository(t) // Force a .nt to have default types

		attributes := AttributeSet(map[string]any{
			"key1": 10,
			"key2": []any{"value1", "value2"},
			"key3": 15.5,
			"key4": "single",
		})

		schemaCompliant := ConfigAttributes{
			"key1": {
				Name: "key1",
				Type: "string",
			},
			"key2": {
				Name: "key2",
				Type: "string[]",
			},
			"key3": {
				Name: "key3",
				Type: "integer",
			},
			"key4": {
				Name: "key4",
				Type: "string[]",
			},
		}
		schemaNonCompliant := ConfigAttributes{
			"key1": {
				Name: "key1",
				Type: "bool", // Not possible
			},
			"key2": {
				Name: "key2",
				Type: "string[]",
			},
			"key3": {
				Name: "key3",
				Type: "integer",
			},
			"key4": {
				Name: "key4",
				Type: "string[]",
			},
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

	t.Run("SetAttribute", func(t *testing.T) {
		set := AttributeSet(map[string]any{
			"source": "A Book",
			"tags":   []string{"favorite"},
		})

		// Override basic types
		set = set.SetAttribute("source", "Another Book")
		assert.Equal(t, "Another Book", set["source"])

		// Append in slices
		set = set.SetAttribute("tags", "life-changing")
		assert.Equal(t, []string{"favorite", "life-changing"}, set["tags"])
		set = set.SetAttribute("tags", []string{"living"})
		assert.Equal(t, []string{"favorite", "life-changing", "living"}, set["tags"])
		// Avoid duplicates
		set = set.SetAttribute("tags", []string{"living"})
		assert.Equal(t, []string{"favorite", "life-changing", "living"}, set["tags"])
	})

	t.Run("ToMarkdownNotation", func(t *testing.T) {
		// Create a test repository for context
		tr := NewTestRepository(t)
		_ = tr // Use the repository to initialize context

		configAttributes := CurrentConfigFile().Attributes

		// Create test attributes
		attributes := AttributeSet{
			"priority": "high",
			"rating":   int64(4), // Should render as ★★
		}

		result := attributes.ToMarkdownNotation(configAttributes)
		// Should contain the priority attribute and rating shorthand
		assert.Contains(t, result, "❗")  // priority: high shorthand
		assert.Contains(t, result, "★★") // rating: 4 shorthand
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
				actual, err := StripBlockTagsAndAttributes()(tt.md)
				require.NoError(t, err)
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
	data := make(map[string]any)
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
	data := make(map[string]any)
	err := yaml.Unmarshal([]byte(input), data)
	require.NoError(t, err)
	assert.Len(t, data["key"], 3)
	assert.IsType(t, []any{}, data["key"])
	values := data["key"].([]any)
	assert.Equal(t, 10, values[0])
	assert.Equal(t, "string", values[1])
	assert.Equal(t, true, values[2])
}

func TestAttributeSetSpecialAttributes(t *testing.T) {

	t.Run("Attribution", func(t *testing.T) {
		tests := []struct {
			name       string
			attributes AttributeSet
			expected   string
		}{
			{
				name: "All fields present",
				attributes: AttributeSet{
					"name":        "Jane Doe",
					"occupation":  "writer",
					"nationality": "Canadian",
				},
				expected: "― Jane Doe, Canadian writer",
			},
			{
				name: "Only name present",
				attributes: AttributeSet{
					"name": "Jane Doe",
				},
				expected: "― Jane Doe",
			},
			{
				name: "Name and occupation present",
				attributes: AttributeSet{
					"name":       "Jane Doe",
					"occupation": "writer",
				},
				expected: "― Jane Doe, writer",
			},
			{
				name: "Name and nationality present",
				attributes: AttributeSet{
					"name":        "Jane Doe",
					"nationality": "Canadian",
				},
				expected: "― Jane Doe",
			},
			{
				name: "Missing name",
				attributes: AttributeSet{
					"occupation":  "writer",
					"nationality": "Canadian",
				},
				expected: "",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual := tt.attributes.Attribution()
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

}

func TestAttributeSetImmutability(t *testing.T) {
	t.Run("SetIfMissing immutability", func(t *testing.T) {
		original := AttributeSet{"key1": "value1"}
		result := original.SetIfMissing("key2", "value2")

		// Original should be unchanged
		assert.Equal(t, AttributeSet{"key1": "value1"}, original)
		// Result should have both keys
		assert.Equal(t, AttributeSet{"key1": "value1", "key2": "value2"}, result)

		// Setting existing key should not change original
		result2 := original.SetIfMissing("key1", "newvalue")
		assert.Equal(t, AttributeSet{"key1": "value1"}, original)
		assert.Equal(t, AttributeSet{"key1": "value1"}, result2)
	})

	t.Run("SetAttribute immutability", func(t *testing.T) {
		original := AttributeSet{"source": "A Book"}
		result := original.SetAttribute("source", "Another Book")

		// Original should be unchanged
		assert.Equal(t, AttributeSet{"source": "A Book"}, original)
		// Result should have new value
		assert.Equal(t, AttributeSet{"source": "Another Book"}, result)
	})

	t.Run("AddTag immutability", func(t *testing.T) {
		original := AttributeSet{"tags": []string{"favorite"}}
		result := original.AddTag("new-tag")

		// Original should be unchanged
		assert.Equal(t, AttributeSet{"tags": []string{"favorite"}}, original)
		// Result should have both tags
		assert.Equal(t, AttributeSet{"tags": []string{"favorite", "new-tag"}}, result)
	})

	t.Run("AddTags immutability", func(t *testing.T) {
		original := AttributeSet{"tags": []string{"favorite"}}
		result := original.AddTags(TagSet{"new-tag", "another-tag"})

		// Original should be unchanged
		assert.Equal(t, AttributeSet{"tags": []string{"favorite"}}, original)
		// Result should have all tags
		assert.Equal(t, AttributeSet{"tags": []string{"favorite", "new-tag", "another-tag"}}, result)
	})

	t.Run("AddHook immutability", func(t *testing.T) {
		original := AttributeSet{"hook": []string{"hook1"}}
		result := original.AddHook("hook2", "hook3")

		// Original should be unchanged
		assert.Equal(t, AttributeSet{"hook": []string{"hook1"}}, original)
		// Result should have all hooks
		assert.Equal(t, AttributeSet{"hook": []string{"hook1", "hook2", "hook3"}}, result)
	})

	t.Run("Chained operations immutability", func(t *testing.T) {
		original := AttributeSet{"key1": "value1"}
		result := original.
			SetIfMissing("key2", "value2").
			AddTag("tag1").
			AddHook("hook1")

		// Original should be unchanged
		assert.Equal(t, AttributeSet{"key1": "value1"}, original)
		// Result should have all modifications
		assert.Contains(t, result, "key1")
		assert.Contains(t, result, "key2")
		assert.Contains(t, result, "tags")
		assert.Contains(t, result, "hook")
	})
}

func TestTags(t *testing.T) {
	tests := []struct {
		name         string
		text         markdown.Document
		expectedText markdown.Document
		expectedTags TagSet
	}{
		{
			name:         "No tags",
			text:         "This text has no tags.",
			expectedText: "This text has no tags.",
			expectedTags: NewEmptyTagSet(),
		},
		{
			name:         "Single tag",
			text:         "Great Book `#favorite`",
			expectedText: "Great Book",
			expectedTags: NewTagSet([]string{"favorite"}),
		},
		{
			name:         "Multiple consecutive tags",
			text:         "Great book `#favorite` `#life-changing`",
			expectedText: "Great book",
			expectedTags: NewTagSet([]string{"favorite", "life-changing"}),
		},
		{
			name:         "Multiple separated tags",
			text:         "Great `#favorite` book `#life-changing`",
			expectedText: "Great book",
			expectedTags: NewTagSet([]string{"favorite", "life-changing"}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test extraction
			actualText, err := test.text.Transform(StripTags())
			require.NoError(t, err)
			actualTags := ExtractTags(test.text)
			assert.Equal(t, test.expectedText, actualText)
			assert.Equal(t, test.expectedTags, actualTags)
		})
	}
}

func TestAttributesOnly(t *testing.T) {
	configAttributes := ConfigAttributes{
		"rating": &ConfigAttribute{
			Name: "rating",
			Type: "integer",
		},
	}

	tests := []struct {
		name               string
		text               markdown.Document
		expectedText       markdown.Document
		expectedAttributes AttributeSet
	}{
		{
			name:               "No attribute",
			text:               "This text has no attribute.",
			expectedText:       "This text has no attribute.",
			expectedAttributes: NewEmptyAttributeSet(),
		},
		{
			name:         "Single attribute",
			text:         "Great Book `@rating: 6`",
			expectedText: "Great Book",
			expectedAttributes: map[string]any{
				"rating": int64(6),
			},
		},
		{
			name:         "Multiple consecutive attributes",
			text:         "Great book `@rating: 6` `@source: _A Book_`",
			expectedText: "Great book",
			expectedAttributes: map[string]any{
				"rating": int64(6),
				"source": "_A Book_",
			},
		},
		{
			name:         "Multiple separated attributes",
			text:         "Great `@rating: 6` book `@source: _A Book_`",
			expectedText: "Great book",
			expectedAttributes: map[string]any{
				"rating": int64(6),
				"source": "_A Book_",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test extraction
			actualText, err := test.text.Transform(StripOnlyAttributes())
			require.NoError(t, err)
			actualAttributes := ExtractOnlyAttributes(test.text, configAttributes)
			assert.Equal(t, test.expectedText, actualText)
			assert.Equal(t, test.expectedAttributes, actualAttributes)
		})
	}
}

func TestShorthands(t *testing.T) {
	configAttributes := ConfigAttributes{
		"status": &ConfigAttribute{
			Name: "status",
			Type: "string",
			Shorthands: map[string]any{
				"📋": "todo",
				"🕒": "in-progress",
				"⛔": "blocked",
				"✅": "done",
			},
			PreserveShorthand: BoolPointer(false), // Remove from title
		},
		"rating": &ConfigAttribute{
			Name: "rating",
			Type: "string",
			Shorthands: map[string]any{
				"★":   "★",
				"★★":  "★★",
				"★★★": "★★★",
			},
			PreserveShorthand: BoolPointer(true), // Keep in title
		},
		"steps": &ConfigAttribute{
			Name:              "steps",
			Type:              "integer",
			ShorthandPattern:  "🚶‍➡️\\s+(\\d+)",
			PreserveShorthand: BoolPointer(true),
		},
		"deep_work": &ConfigAttribute{
			Name:              "deep_work",
			Type:              "integer",
			ShorthandPattern:  "dw:(\\d+)",
			PreserveShorthand: BoolPointer(false),
		},
	}

	tests := []struct {
		name               string
		text               markdown.Document
		expectedText       markdown.Document
		expectedAttributes AttributeSet
	}{
		{
			name:         "Status shorthand removed",
			text:         "Add Zen Mode 🕒",
			expectedText: "Add Zen Mode",
			expectedAttributes: map[string]any{
				"status": "in-progress",
			},
		},
		{
			name:         "Rating shorthand preserved",
			text:         "Great Book ★★★",
			expectedText: "Great Book ★★★",
			expectedAttributes: map[string]any{
				"rating": "★★★",
			},
		},
		{
			name:         "Status shorthand at beginning",
			text:         "Add 🕒 Zen Mode",
			expectedText: "Add Zen Mode",
			expectedAttributes: map[string]any{
				"status": "in-progress",
			},
		},
		{
			name:         "Multiple emojis",
			text:         "Add Zen Mode 🕒 ★★",
			expectedText: "Add Zen Mode ★★",
			expectedAttributes: map[string]any{
				"status": "in-progress",
				"rating": "★★",
			},
		},
		{
			name:         "Shorthand pattern to preserve",
			text:         "🚶‍➡️ 1000 steps 👍",
			expectedText: "🚶‍➡️ 1000 steps 👍",
			expectedAttributes: map[string]any{
				"steps": int64(1000),
			},
		},
		{
			name:         "Shorthand pattern to remove",
			text:         "🚶‍➡️ 10000 dw:3 👍",
			expectedText: "🚶‍➡️ 10000 👍",
			expectedAttributes: map[string]any{
				"steps":     int64(10000),
				"deep_work": int64(3),
			},
		},
		{
			name:               "No shorthand",
			text:               "Add Zen Mode",
			expectedText:       "Add Zen Mode",
			expectedAttributes: map[string]any{
				// No attributes
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test extraction
			actualText, err := test.text.Transform(StripShorthands(configAttributes))
			require.NoError(t, err)
			actualAttributes := ExtractShorthands(test.text, configAttributes)
			assert.Equal(t, test.expectedText, actualText)
			assert.Equal(t, test.expectedAttributes, actualAttributes)
		})
	}
}

func TestExtractAllTagsAndAttributesAndEmojis(t *testing.T) {
	configAttributes := ConfigAttributes{
		"status": &ConfigAttribute{
			Name: "status",
			Type: "string",
			Shorthands: map[string]any{
				"📋": "todo",
				"🕒": "in-progress",
				"⛔": "blocked",
				"✅": "done",
			},
			PreserveShorthand: BoolPointer(false), // Remove from title
		},
		"rating": &ConfigAttribute{
			Name: "rating",
			Type: "string",
			Shorthands: map[string]any{
				"★":   "★",
				"★★":  "★★",
				"★★★": "★★★",
			}, PreserveShorthand: BoolPointer(true), // Keep in title
		},
	}

	tests := []struct {
		name               string
		text               markdown.Document
		expectedText       markdown.Document
		expectedTags       TagSet
		expectedAttributes AttributeSet
		expectedEmojis     EmojiSet
	}{
		{
			name:         "All elements",
			text:         "Great Book `#favorite` 🕒 ★★★ `@source: _A Book_` 👍",
			expectedText: "Great Book ★★★ 👍",
			expectedTags: []string{"favorite"},
			expectedAttributes: AttributeSet{
				"status": "in-progress",
				"source": "_A Book_",
				"rating": "★★★",
				"tags":   []string{"favorite"},
			},
			expectedEmojis: []string{"★", "👍"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualTags, actualAttributes, actualEmojis := ExtractAllTagsAndAttributesAndEmojis(test.text, configAttributes)
			assert.Equal(t, test.expectedTags, actualTags)
			assert.Equal(t, test.expectedAttributes, actualAttributes)
			assert.Equal(t, test.expectedEmojis, actualEmojis)
		})
	}
}
