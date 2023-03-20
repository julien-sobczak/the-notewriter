package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

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

// FIXME remove
func mergeAttributes(attributes ...map[string]interface{}) map[string]interface{} {
	// Implementation: THe code is obscure due to untyped elements.
	// We don't want to always replace old values when the old value is a slice
	// that can accept these new values too.
	//
	// Examples:
	//   ---
	//   tags: [a]
	//   references: []
	//   ---
	//
	//   `#b`
	//   `@references: https://example.org`
	//
	// Should be the same as:
	//   ---
	//   tags: [a, b]
	//   references: [https://example.org]
	//   ---
	//
	// Most of the code tries to manage this use case.

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
					case []string:
						for _, item := range y {
							result[newKey] = append(x, fmt.Sprintf("%v", item))
						}
					default:
						result[newKey] = append(x, newValue)
					}
				case []string:
					switch y := newValue.(type) {
					case []interface{}:
						for _, item := range y {
							result[newKey] = append(x, fmt.Sprintf("%v", item))
						}
					case []string:
						result[newKey] = append(x, y...)
					default:
						result[newKey] = append(x, fmt.Sprintf("%v", newValue))
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
	return CastAttributes(attributes), nil
}

// CastAttributes enforces the map only used common types (no []interface{}, but []string)
// and converts raw values to their declared type as defined in linter schemas.
func CastAttributes(attributes map[string]interface{}) map[string]interface{} {
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

// IsBool returns if a variable is a JSON boolean.
func IsBool(value interface{}) bool {
	return reflect.Bool == reflect.TypeOf(value).Kind()
}

// IsString returns if a variable is a JSON string.
func IsString(value interface{}) bool {
	return reflect.String == reflect.TypeOf(value).Kind()
}
