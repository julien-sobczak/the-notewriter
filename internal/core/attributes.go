package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

var (
	regexTags                   = regexp.MustCompile("`#(\\S+)`")                          // Ex: `#favorite`
	regexAttributes             = regexp.MustCompile("`@([a-zA-Z0-9_.-]+)\\s*:\\s*(.+?)`") // Ex: `@source: _A Book_`, `@isbn: 9780807014271`
	regexBlockTagAttributesLine = regexp.MustCompile("^\\s*(`.*?`\\s+)*`.*?`\\s*$")        // Ex: `#favorite` `@isbn: 9780807014271`
)

/*
 * TagSet
 */

type TagSet []string

var EmptyTags TagSet

// NewTagSet creates a new tag set removing duplicate tags.
func NewTagSet(tags []string) TagSet {
	return EmptyTags.Merge(tags)
}

func (t TagSet) Merge(tagSets ...TagSet) TagSet {
	var result TagSet

	// Start with initial set
	result = append(result, t...)

	// Append new tag in other sets
	for _, tags := range tagSets {
		for _, tag := range tags {
			if !slices.Contains(result, tag) {
				result = append(result, tag)
			}
		}
	}
	return result
}

/*
 * AttributeSet
 */

type AttributeSet map[string]any

type CastFn[T any] func(v any) (T, bool)

var CastStringFn CastFn[string] = func(value any) (string, bool) {
	if IsPrimitive(value) {
		return fmt.Sprintf("%v", value), true
	}
	return "", false
}

var CastObjectFn CastFn[any] = func(value any) (any, bool) {
	if IsObject(value) {
		return value, true
	}
	return nil, false
}

var CastIntegerFn CastFn[int64] = func(value any) (int64, bool) {
	if IsString(value) {
		stringValue := fmt.Sprintf("%v", value)
		typedValue, err := strconv.ParseInt(stringValue, 10, 64)
		if err == nil {
			return typedValue, true
		}
		return 0, false
	}
	if IsInteger(value) {
		switch v := value.(type) {
		case int:
			return int64(v), true
		case int8:
			return int64(v), true
		case int16:
			return int64(v), true
		case int32:
			return int64(v), true
		case int64:
			return int64(v), true
		case uint:
			return int64(v), true
		case uint8:
			return int64(v), true
		case uint16:
			return int64(v), true
		case uint32:
			return int64(v), true
		case uint64:
			return int64(v), true
		case uintptr:
			return int64(v), true
		}
	}

	if IsFloat(value) {
		switch v := value.(type) {
		case float32:
			return int64(v), true
		case float64:
			return int64(v), true
		}
	}

	return 0, false
}

var CastFloatFn CastFn[float64] = func(value any) (float64, bool) {
	if IsString(value) {
		stringValue := fmt.Sprintf("%v", value)
		typedValue, err := strconv.ParseFloat(stringValue, 64)
		if err == nil {
			return typedValue, true
		}
		return 0, false
	}

	if IsInteger(value) {
		return float64(value.(int)), true
	}

	if IsFloat(value) {
		switch v := value.(type) {
		case float32:
			return float64(v), true
		case float64:
			return v, true
		}
	}
	return 0, false
}

var CastBoolFn CastFn[bool] = func(value any) (bool, bool) {
	if IsString(value) {
		if value == "true" {
			return true, true
		} else if value == "false" {
			return false, true
		} else {
			return false, false
		}
	}
	if IsBool(value) {
		return value.(bool), true
	}
	return false, false
}

// DiffKeys returns the keys present in only one of the attribute sets.
func (a AttributeSet) DiffKeys(other AttributeSet) []string {
	b := other
	var results []string
	for key, valueA := range a {
		valueB, ok := b[key]
		if !ok || !reflect.DeepEqual(valueA, valueB) {
			results = append(results, key)
		}
	}
	for key := range b {
		_, ok := b[key]
		if !ok {
			results = append(results, key)
		}
	}
	slices.Sort(results)
	return results
}

// Const to represent an empty set of attributes
var EmptyAttributes AttributeSet

// NewAttributeSetFromYAML unmarshall attributes.
func NewAttributeSetFromYAML(rawValue string) (AttributeSet, error) {
	var attributes map[string]interface{}
	err := yaml.Unmarshal([]byte(rawValue), &attributes)
	if err != nil {
		return nil, err
	}
	return attributes, nil
}

func (a AttributeSet) Merge(attributes ...AttributeSet) AttributeSet {
	// Implementation: Attribute lists must already have been casted correctly to avoid incompatible types

	var result AttributeSet = make(map[string]any)
	for newKey, newValue := range a {
		result.SetAttribute(newKey, newValue)
	}
	for _, m := range attributes {
		for newKey, newValue := range m {
			result.SetAttribute(newKey, newValue)
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func (a AttributeSet) SetAttribute(name string, value any) {
	// Check if the attribute was already defined
	currentValue, ok := a[name]

	if !ok {
		a[name] = value
	}

	// If the type is a slice, append the new value instead of overriding
	switch x := currentValue.(type) {
	case []any:
		switch y := value.(type) {
		case []any:
			a[name] = append(x, y...)
		default:
			a[name] = append(x, value)
		}
	default:
		// override
		a[name] = value
	}
}

func (a AttributeSet) ToJSON() (string, error) {
	var buf bytes.Buffer
	bufEncoder := json.NewEncoder(&buf)
	err := bufEncoder.Encode(a)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (a AttributeSet) ToYAML() (string, error) {
	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(2)
	err := bufEncoder.Encode(a)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CastAttributes enforces the types declared in linter schemas.
func (a AttributeSet) Cast(types map[string]string) AttributeSet {
	result := make(map[string]interface{})

	// Implementation: We ignore invalid values to avoid having to cast or manage errors
	// when reading them later.

	for key, value := range a {
		declaredType, found := types[key]
		if !found {
			result[key] = value
			continue
		}
		if typedValue, ok := CastAttribute(value, declaredType); ok {
			result[key] = typedValue
		}
	}

	return result
}

func CastArray[T any](arr []any, castFn CastFn[T]) (results []T, ok bool) {
	for _, itemValue := range arr {
		v, ok := castFn(itemValue)
		if !ok {
			return nil, false
		}
		results = append(results, v)
	}
	return results, true
}

// CastAttribute enforces the type declared in linter schemas.
func CastAttribute(value any, declaredType string) (any, bool) {
	if value == nil {
		return nil, true
	}

	if strings.HasSuffix(declaredType, "[]") {
		if !IsArray(value) {
			value = []any{value}
		}
		itemType := strings.TrimSuffix(declaredType, "[]")
		arr := UnpackArray(value)
		switch itemType {
		case "string":
			return CastArray(arr, CastStringFn)
		case "object":
			return CastArray(arr, CastObjectFn)
		case "integer":
			return CastArray(arr, CastIntegerFn)
		case "float":
			return CastArray(arr, CastFloatFn)
		case "bool":
			return CastArray(arr, CastBoolFn)
		}
	}

	switch declaredType {
	case "string":
		return CastStringFn(value)
	case "object":
		return CastObjectFn(value)
	case "integer":
		return CastIntegerFn(value)
	case "float":
		return CastFloatFn(value)
	case "bool":
		return CastBoolFn(value)
	}

	// Ignore invalid values
	return nil, false
}

/*
 * Markdown
 */

// ExtractBlockTagsAndAttributes searches for all tags and attributes declared on standalone lines
// (in comparison with tags/attributes defined, for example, on To-Do list items).
func ExtractBlockTagsAndAttributes(content markdown.Document, types map[string]string) (TagSet, AttributeSet) {

	// Collect tags and attributes
	var tags TagSet
	var attributes AttributeSet = make(map[string]interface{})

	for _, line := range content.Lines() {

		// only tags and attributes?
		if text.IsBlank(line) || !regexBlockTagAttributesLine.MatchString(line) {
			continue
		}

		// Append tags and attributes to collected ones
		matches := regexTags.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			tag := match[1]

			// Append new tag
			tags = append(tags, tag)

			// Append tags in attributes too (tags are attributes with syntaxic sugar)
			if _, ok := attributes["tags"]; !ok {
				attributes["tags"] = []interface{}{}
			}
			attributes["tags"] = append(attributes["tags"].([]interface{}), tag)
		}
		matches = regexAttributes.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			name := match[1]
			value := match[2]

			declaredType := "string"
			if types[name] != "" {
				declaredType = types[name]
			}
			if typedValue, ok := CastAttribute(value, declaredType); ok {
				attributes.SetAttribute(name, typedValue)
			}

			// Tags can also be set as attributes (= longer syntax)
			if name == "tags" {
				attributes["tags"] = append(attributes["tags"].([]interface{}), value)
			}
		}
	}

	return tags, attributes
}

// StripTagsAndAttributes remove all tags and attributes.
func StripBlockTagsAndAttributes(content markdown.Document) markdown.Document {
	var res bytes.Buffer

	for _, line := range content.Lines() {
		// not only tags and attributes?
		if text.IsBlank(line) || strings.HasPrefix(line, "```") || !regexBlockTagAttributesLine.MatchString(line) {
			res.WriteString(line + "\n")
		}
	}

	return markdown.Document(text.SquashBlankLines(res.String())).TrimSpace()
}

// StripAllTagsAndAttributes removes all tags and attributes from a text.
func StripAllTagsAndAttributes(content markdown.Document) markdown.Document {
	var res bytes.Buffer
	for _, line := range content.Lines() {
		newLine := regexTags.ReplaceAllLiteralString(line, "")
		newLine = regexAttributes.ReplaceAllLiteralString(newLine, "")
		if !text.IsBlank(newLine) {
			res.WriteString(newLine + "\n")
		}
	}
	return markdown.Document(text.SquashBlankLines(res.String())).TrimSpace()
}

/* Helpers */

var primitiveDataTypeKinds = []reflect.Kind{
	reflect.Bool,
	reflect.Int,
	reflect.Int8,
	reflect.Int16,
	reflect.Int32,
	reflect.Int64,
	reflect.Uint,
	reflect.Uint8,
	reflect.Uint16,
	reflect.Uint32,
	reflect.Uint64,
	reflect.Uintptr,
	reflect.Float32,
	reflect.Float64,
	reflect.Complex64,
	reflect.Complex128,
	reflect.String,
}
var compositeDataTypeKinds = []reflect.Kind{
	reflect.Array,
	reflect.Map,
	reflect.Slice,
	reflect.Struct,
}

var arrayDataTypeKinds = []reflect.Kind{
	reflect.Array,
	reflect.Slice,
}

var objectDataTypeKinds = []reflect.Kind{
	reflect.Map,
	reflect.Struct,
}

var numberDataTypeKinds = []reflect.Kind{
	reflect.Int,
	reflect.Int8,
	reflect.Int16,
	reflect.Int32,
	reflect.Int64,
	reflect.Uint,
	reflect.Uint8,
	reflect.Uint16,
	reflect.Uint32,
	reflect.Uint64,
	reflect.Uintptr,
	reflect.Float32,
	reflect.Float64,
}

var integerDataTypeKinds = []reflect.Kind{
	reflect.Int,
	reflect.Int8,
	reflect.Int16,
	reflect.Int32,
	reflect.Int64,
	reflect.Uint,
	reflect.Uint8,
	reflect.Uint16,
	reflect.Uint32,
	reflect.Uint64,
	reflect.Uintptr,
}

var floatDataTypeKinds = []reflect.Kind{
	reflect.Float32,
	reflect.Float64,
}

// IsPrimitive returns if a variable is a primitive type.
func IsPrimitive(value interface{}) bool {
	return slices.Contains(primitiveDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsComposite returns if a variable is a composite type.
func IsComposite(value interface{}) bool {
	return slices.Contains(compositeDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsArray returns if a variable is a JSON array.
func IsArray(value interface{}) bool {
	return slices.Contains(arrayDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsObject returns if a variable is a JSON map.
func IsObject(value interface{}) bool {
	return slices.Contains(objectDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsNumber returns if a variable is a JSON number.
func IsNumber(value interface{}) bool {
	return slices.Contains(numberDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsInteger returns if a variable is a JSON number of integer type.
func IsInteger(value interface{}) bool {
	return slices.Contains(integerDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsFloat returns if a variable is a JSON number of float type.
func IsFloat(value interface{}) bool {
	return slices.Contains(floatDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsBool returns if a variable is a JSON boolean.
func IsBool(value interface{}) bool {
	return reflect.Bool == reflect.TypeOf(value).Kind()
}

// IsString returns if a variable is a JSON string.
func IsString(value interface{}) bool {
	return reflect.String == reflect.TypeOf(value).Kind()
}

func UnpackArray(value any) []any {
	v := reflect.ValueOf(value)
	r := make([]any, v.Len())
	for i := 0; i < v.Len(); i++ {
		r[i] = v.Index(i).Interface()
	}
	return r
}
