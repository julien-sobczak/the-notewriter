package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"slices"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
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

func (t TagSet) AsList() []string {
	return t
}

func (t TagSet) Includes(tag string) bool {
	return slices.Contains(t, tag)
}

func (t TagSet) IncludesAll(tags []string) bool {
	for _, tag := range tags {
		if !t.Includes(tag) {
			return false
		}
	}
	return true
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
	// Already the right type?
	if IsBool(value) {
		return value.(bool), true
	}
	// Only convert from string to bool
	if IsString(value) {
		if value == "true" {
			return true, true
		} else if value == "false" {
			return false, true
		} else {
			return false, false
		}
	}
	return false, false
}

var CastDateFn CastFn[string] = func(v any) (string, bool) {
	// Only convert from string to time.Time
	if !IsString(v) {
		return "", false
	}

	value := v.(string)

	// YAML uses ISO 8601 format. Try it first
	// (using a subset RFC as Golang doesn't provide the ISO 8601 format specifically).
	// See https://symfony.com/doc/current/reference/formats/yaml.html#dates
	_, err := time.Parse(time.RFC3339, value) // Ex: "2023-10-15T14:12:00Z"
	if err == nil {
		return value, true
	}

	// Try additional common format (more specific to least specific)

	_, err = time.Parse(time.DateTime, value) // Ex: "2023-10-15 14:12:00"
	if err == nil {
		return value, true
	}

	_, err = time.Parse(time.DateOnly, value) // Ex: "2023-10-15"
	if err == nil {
		return value, true
	}

	_, err = time.Parse("2006-01", value) // Ex: "2023-10"
	if err == nil {
		return value, true
	}

	_, err = time.Parse("2006", value) // Ex: "2023"
	if err == nil {
		return value, true
	}

	return "", false
}

func NewAttributeSetFromMarkdown(md *markdown.File) (AttributeSet, error) {
	attributesMap, err := md.FrontMatter.AsMap()
	if err != nil {
		return nil, err
	}
	return AttributeSet(attributesMap).Cast(CurrentConfigFile().Attributes)
}

// SetIfMissing sets the attribute only if it is not already set.
func (a AttributeSet) SetIfMissing(key string, value any) {
	// IMPROVEMENT Avoid side-effect methods
	if _, ok := a[key]; !ok {
		a[key] = value
	}
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
var EmptyAttributes AttributeSet = make(map[string]any)

// NewAttributeSetFromYAML unmarshall attributes.
func NewAttributeSetFromYAML(rawValue string) (AttributeSet, error) {
	var attributes map[string]interface{}
	err := yaml.Unmarshal([]byte(rawValue), &attributes)
	if err != nil {
		return nil, err
	}
	if attributes == nil {
		return EmptyAttributes, nil
	}
	return attributes, nil
}

func (a AttributeSet) Merge(attributes ...AttributeSet) AttributeSet {
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
		return EmptyAttributes
	}

	return result
}

func (a AttributeSet) AsMap() map[string]any {
	return a
}

func (a AttributeSet) SetAttribute(name string, value any) {
	// IMPROVEMENT Avoid side-effect methods
	// Check if the attribute was already defined
	currentValue, ok := a[name]

	if !ok {
		a[name] = value
	}

	// If the type is a slice, append the new value instead of overriding
	switch x := currentValue.(type) {
	case []string:
		switch y := value.(type) {
		case []string:
			// Avoid duplicates when possible (ex: tags)
			for _, vy := range y {
				if !slices.Contains(x, vy) {
					x = append(x, vy)
				}
			}
			a[name] = x
		default:
			// Avoid duplicates (ex: tags)
			vy := fmt.Sprintf("%v", value)
			if !slices.Contains(x, vy) {
				x = append(x, vy)
			}
			a[name] = x
		}
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

/* Special attributes */

func (a AttributeSet) Tags() TagSet {
	if v, ok := a["tags"].([]string); ok {
		return v
	}
	return nil
}

func (a AttributeSet) AddTag(newTag string) {
	// IMPROVEMENT Avoid side-effect methods
	if _, ok := a["tags"]; !ok {
		// Not tag currently present
		a["tags"] = []string{newTag}
		return
	}
	if tags, ok := a["tags"].([]string); ok {
		for _, tag := range tags {
			if tag == newTag {
				// Already present
				return
			}
		}
		a["tags"] = append(tags, newTag)
		return
	}
}

func (a AttributeSet) Slug() (string, bool) {
	v, ok := a["slug"]
	if !ok {
		return "", false
	}
	return v.(string), true
}

func (a AttributeSet) Hooks() TagSet {
	if v, ok := a["hook"].([]string); ok {
		return v
	}
	return nil
}

func (a AttributeSet) AddHook(hookNames ...string) {
	if _, ok := a["hook"]; !ok {
		// Not hook currently present
		a["hook"] = hookNames
		return
	}
	if newHooks, ok := a["hook"].([]string); ok {
		for _, hookName := range hookNames {
			if !slices.Contains(newHooks, hookName) {
				newHooks = append(newHooks, hookName)
			}
		}
		a["hook"] = newHooks
		return
	}
}

/* Format */

func (a AttributeSet) ToJSON() (string, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	var buf bytes.Buffer
	bufEncoder := json.NewEncoder(&buf)
	bufEncoder.SetIndent("", "  ")
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

func (a AttributeSet) CastValueAsString(name string) string {
	if v, ok := a[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	// Conversion errors are ignored (we consider the requested attribute doesn't exist)
	return ""
}

// CastOrIgnore enforces the types declared in linter schemas and ignore attributes that cannot be cast.
func (a AttributeSet) CastOrIgnore(types ConfigAttributes) AttributeSet {
	result := make(AttributeSet)

	// Implementation: We ignore invalid values to avoid having
	// to cast or manage errors when reading them later.

	for key, value := range a {
		declaredType, found := types.Find(key)
		if !found {
			result[key] = value
			continue
		}
		if typedValue, ok := CastAttribute(value, *declaredType); ok {
			result[key] = typedValue
		}
	}

	return result
}

// Cast enforces the types declared in linter schemas.
func (a AttributeSet) Cast(types ConfigAttributes) (AttributeSet, error) {
	result := make(map[string]interface{})
	for key, value := range a {
		declaredType, found := types.Find(key)
		if !found {
			result[key] = value
			continue
		}

		typedValue, ok := CastAttribute(value, *declaredType)
		if !ok {
			return nil, fmt.Errorf("invalid value for attribute %s: %v", declaredType, value)
		}
		result[key] = typedValue
	}

	return result, nil
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
func CastAttribute(value any, declaredType ConfigAttribute) (any, bool) {
	if value == nil {
		return nil, true
	}

	typeName := declaredType.Type

	if strings.HasSuffix(typeName, "[]") {
		if !IsArray(value) {
			value = []any{value}
		}
		itemType := strings.TrimSuffix(typeName, "[]")
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
			fallthrough
		case "boolean":
			return CastArray(arr, CastBoolFn)
		case "date":
			return CastArray(arr, CastDateFn)
		}
	}

	switch typeName {
	case "string":
		return CastStringFn(value)
	case "object":
		return CastObjectFn(value)
	case "integer":
		return CastIntegerFn(value)
	case "float":
		return CastFloatFn(value)
	case "bool":
		fallthrough
	case "boolean":
		return CastBoolFn(value)
	case "date":
		return CastDateFn(value)
	}

	// Ignore invalid values
	return nil, false
}

/*
 * Markdown
 */

// OnlyTagsAndAttributes returns true if the line contains only tags and attributes.
func OnlyTagsAndAttributes(line string) bool {
	return regexBlockTagAttributesLine.MatchString(line)
}

// ExtractBlockTagsAndAttributes searches for all tags and attributes declared on standalone lines
// (in comparison with tags/attributes defined, for example, on To-Do list items).
func ExtractBlockTagsAndAttributes(content markdown.Document) (TagSet, AttributeSet) {

	// Collect tags and attributes
	var tags TagSet
	var attributes AttributeSet = make(map[string]interface{})

	for _, line := range content.Lines() {

		// empty or only tags and attributes?
		if text.IsBlank(line) || !OnlyTagsAndAttributes(line) {
			continue
		}

		// Append tags and attributes to collected ones
		matches := regexTags.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			tag := match[1]

			// Append new tag
			tags = append(tags, tag)
		}
		matches = regexAttributes.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			name := match[1]
			value := match[2]

			attributes[name] = value
		}
	}

	// Cast (ensure the tags attribute is an array too)
	attributes = attributes.CastOrIgnore(CurrentConfigFile().Attributes)

	tagsInAttributes := attributes.Tags()

	// The tag syntax is only syntax sugar. Tags must be added in attributes too.
	for _, tag := range tags {
		attributes.AddTag(tag)
	}

	// Add tags declared using `@tags` attributes
	tags = append(tags, tagsInAttributes...)

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
