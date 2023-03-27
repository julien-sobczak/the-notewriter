package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/julien-sobczak/the-notetaker/pkg/text"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

func DiffKeys(a, b map[string]interface{}) []string {
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

func AttributesJSON(attributes map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	bufEncoder := json.NewEncoder(&buf)
	err := bufEncoder.Encode(attributes)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func AttributesYAML(attributes map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(Indent)
	err := bufEncoder.Encode(attributes)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func mergeTags(tags ...[]string) []string {
	var result []string
	for _, items := range tags {
		for _, item := range items {
			found := false
			for _, existingItem := range result {
				if existingItem == item {
					found = true
					break
				}
			}
			if !found {
				result = append(result, item)
			}
		}
	}
	return result
}

func MergeAttributes(attributes ...map[string]interface{}) map[string]interface{} {
	// Implementation: Attribute lists must already have been casted correctly
	// using the function CastAttributes.

	result := make(map[string]interface{})
	empty := true

	// Iterate over maps
	for _, m := range attributes {
		for newKey, newValue := range m {

			// Check if the attribute was already defined
			if currentValue, ok := result[newKey]; ok {

				// If the tyoe is a slice, append the new value instead of overriding
				switch x := currentValue.(type) {
				case []interface{}:
					switch y := newValue.(type) {
					case []interface{}:
						result[newKey] = append(x, y...)
					default:
						result[newKey] = append(x, newValue)
					}
				default:
					// override
					result[newKey] = newValue
				}
			} else {
				result[newKey] = newValue
			}
			empty = false
		}
	}
	if empty {
		return nil
	}
	return result
}

// UnmarshalAttributes unmarshall attributes and ensure the right types are used.
func UnmarshalAttributes(rawValue string) (map[string]interface{}, error) {
	var attributes map[string]interface{}
	err := yaml.Unmarshal([]byte(rawValue), &attributes)
	if err != nil {
		return nil, err
	}
	types := GetSchemaAttributeTypes()
	return CastAttributes(attributes, types), nil
}

// CastAttributes enforces the types declared in linter schemas.
func CastAttributes(attributes map[string]interface{}, types map[string]string) map[string]interface{} {
	result := make(map[string]interface{})

	// Implementation: We ignore invalid values to avoid having to cast or manage errors
	// when reading them later.

	for key, value := range attributes {
		declaredType, found := types[key]
		if !found {
			result[key] = value
			continue
		}
		switch declaredType {
		case "array":
			if !IsArray(value) {
				if IsString(value) {
					result[key] = []interface{}{fmt.Sprintf("%s", value)}
				} else {
					result[key] = []interface{}{value}
				}
			} else {
				result[key] = value
			}
		case "string":
			if IsPrimitive(value) {
				typedValue := fmt.Sprintf("%v", value)
				result[key] = typedValue
			}
		case "object":
			if IsObject(value) {
				result[key] = value
			}
		case "number":
			if IsString(value) {
				stringValue := fmt.Sprintf("%v", value)
				if strings.Contains(stringValue, ".") { // decimal point
					typedValue, err := strconv.ParseFloat(stringValue, 64)
					if err == nil {
						result[key] = typedValue
					}
				} else {
					typedValue, err := strconv.ParseInt(stringValue, 10, 64)
					if err == nil {
						result[key] = typedValue
					}
				}
			} else if IsInteger(value) {
				switch v := value.(type) {
				case int:
					result[key] = int64(v)
				case int8:
					result[key] = int64(v)
				case int16:
					result[key] = int64(v)
				case int32:
					result[key] = int64(v)
				case int64:
					result[key] = int64(v)
				case uint:
					result[key] = int64(v)
				case uint8:
					result[key] = int64(v)
				case uint16:
					result[key] = int64(v)
				case uint32:
					result[key] = int64(v)
				case uint64:
					result[key] = int64(v)
				case uintptr:
					result[key] = int64(v)
				}
			} else if IsFloat(value) {
				switch v := value.(type) {
				case float32:
					result[key] = float64(v)
				case float64:
					result[key] = v
				}
			}
		case "bool":
			if IsBool(value) {
				result[key] = value
			}
		}
	}
	return result
}

// CastAttributesOld enforces that the map only uses common types
// (ex: no []interface{}, but []string if all values are string values).
// The function also converts raw values to their declared type as defined in linter schemas.
func CastAttributesOld(attributes map[string]interface{}) map[string]interface{} {
	types := GetSchemaAttributeTypes()
	result := make(map[string]interface{})
	for key, value := range attributes {
		declaredType, found := types[key]
		if !found {
			result[key] = value
			continue
		}
		switch declaredType {
		case "array":
			if !IsArray(value) {
				if IsString(value) {
					result[key] = []string{fmt.Sprintf("%s", value)}
				} else {
					// Create an array of the right type
					TypeElem := reflect.TypeOf(value)
					slice := reflect.MakeSlice(reflect.SliceOf(TypeElem), 0, 1)
					elemValue := reflect.ValueOf(value)
					reflect.Append(slice, elemValue)
					result[key] = slice.Interface()
				}
			} else {
				switch v := value.(type) {
				case []interface{}:
					var typeElem reflect.Type
					sameType := true
					// Check if all elements have the same type
					for _, elem := range v {
						newTypeElem := reflect.TypeOf(elem)
						if typeElem == nil {
							typeElem = newTypeElem
						} else if typeElem != newTypeElem {
							sameType = false
						}
					}
					if sameType {
						// Recreate a new slice using the right type
						slice := reflect.MakeSlice(reflect.SliceOf(typeElem), 0, len(v))
						for _, elem := range v {
							slice = reflect.Append(slice, reflect.ValueOf(elem))
						}
						result[key] = slice.Interface()
					} else {
						// Stay with the []interface{} type...
						result[key] = value
					}
				default:
					// Not []interface{}, nothing to do
					result[key] = value
				}
			}
		case "string":
			if IsPrimitive(value) {
				typedValue := fmt.Sprintf("%v", value)
				result[key] = typedValue
			} else {
				// Casting not possible
				result[key] = value
			}
		default: // "object", "number", "bool"
			// Nothing can be done
			result[key] = value
		}
	}
	return result
}

// NonInheritableAttributes returns the attributes that must not be inherited.
func NonInheritableAttributes(relativePath string, kind NoteKind) []string {
	var results []string
	definitions := GetSchemaAttributes(relativePath, kind)
	for _, definition := range definitions {
		if !*definition.Inherit {
			results = append(results, definition.Name)
		}
	}
	return results
}

// FilterNonInheritableAttributes removes from the list all non-inheritable attributes.
func FilterNonInheritableAttributes(attributes map[string]interface{}, relativePath string, kind NoteKind) map[string]interface{} {
	nonInheritableAttributes := NonInheritableAttributes(relativePath, kind)
	result := make(map[string]interface{})
	for key, value := range attributes {
		if slices.Contains(nonInheritableAttributes, key) {
			// non-inheritable
			continue
		}
		result[key] = value
	}
	return result
}

// ExtractBlockTagsAndAttributes searches for all tags and attributes declared on standalone lines
// (in comparison with tags/attributes defined, for example, on To-Do list items).
func ExtractBlockTagsAndAttributes(content string) ([]string, map[string]interface{}) {

	// Collect tags and attributes
	var tags []string
	var attributes map[string]interface{} = make(map[string]interface{})

	lines := strings.Split(content, "\n")
	for _, line := range lines {

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
			attributes[name] = value

			// Tags can also be set as attributes (= longer syntax)
			if name == "tags" {
				attributes["tags"] = append(attributes["tags"].([]interface{}), value)
			}
		}
	}

	return tags, attributes
}

// StripTagsAndAttributes remove all tags and attributes.
func StripBlockTagsAndAttributes(content string) string {
	var res bytes.Buffer

	lines := strings.Split(content, "\n")
	for _, line := range lines {

		// not only tags and attributes?
		if text.IsBlank(line) || !regexBlockTagAttributesLine.MatchString(line) {
			res.WriteString(line + "\n")
		}
	}

	return strings.TrimSpace(text.SquashBlankLines(res.String()))
}

// RemoveTagsAndAttributes removes all tags and attributes from a text.
func RemoveTagsAndAttributes(content string) string {
	var res bytes.Buffer
	for _, line := range strings.Split(content, "\n") {
		newLine := regexTags.ReplaceAllLiteralString(line, "")
		newLine = regexAttributes.ReplaceAllLiteralString(newLine, "")
		if !text.IsBlank(newLine) {
			res.WriteString(newLine + "\n")
		}
	}
	return strings.TrimSpace(text.SquashBlankLines(res.String()))
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
