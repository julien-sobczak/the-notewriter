package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
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

// Const to represent an empty set of tags
func NewEmptyTagSet() TagSet {
	return []string{}
}

// NewTagSet creates a new tag set removing duplicate tags.
func NewTagSet(tags []string) TagSet {
	return NewEmptyTagSet().Merge(tags)
}

// NewTagSetFromText extracts tags from a text, including those defined via shorthands.
func NewTagSetFromText(content string, configAttributes ConfigAttributes, configTags ConfigTags) TagSet {
	tags := ExtractTags(markdown.Document(content))
	_, shorthandTags := ExtractShorthands(markdown.Document(content), configAttributes, configTags)
	return tags.Merge(shorthandTags)
}

func (t TagSet) Merge(tagSets ...TagSet) TagSet {
	result := NewEmptyTagSet()

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

// ToMarkdownNotation renders tags using The NoteWriter notation
func (t TagSet) ToMarkdownNotation() string {
	if len(t) == 0 {
		return ""
	}

	var rendered []string
	for _, tag := range t {
		rendered = append(rendered, "`#"+tag+"`")
	}
	return " " + strings.Join(rendered, " ")
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
	switch value {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
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

// NewEmptyAttributes creates an empty attribute set.
func NewEmptyAttributeSet() AttributeSet {
	return make(AttributeSet)
}

// NewAttributeSet creates an attribute set from an existing map.
func NewAttributeSet(attributes map[string]any) AttributeSet {
	result := make(AttributeSet)
	for k, v := range attributes {
		result[k] = v
	}
	return result
}

// NewAttributeSetFromYAML unmarshall attributes.
func NewAttributeSetFromYAML(rawValue string) (AttributeSet, error) {
	var attributes map[string]any
	err := yaml.Unmarshal([]byte(rawValue), &attributes)
	if err != nil {
		return nil, err
	}
	if attributes == nil {
		return NewEmptyAttributeSet(), nil
	}
	return attributes, nil
}

// NewAttributeSetFromMarkdown extracts attributes from the front-matter of a markdown file.
func NewAttributeSetFromMarkdown(md *markdown.File) (AttributeSet, error) {
	attributesMap, err := md.FrontMatter.AsMap()
	if err != nil {
		return nil, err
	}
	return AttributeSet(attributesMap).Cast(CurrentConfigFile().Attributes)
}

// NewAttributeSetFromText extracts attributes from a text.
func NewAttributeSetFromText(content string, configAttributes ConfigAttributes, configTags ConfigTags) AttributeSet {
	return ExtractAttributes(markdown.Document(content), configAttributes, configTags)
}

// SetIfMissing creates a new AttributeSet with the attribute set only if it is not already present.
func (a AttributeSet) SetIfMissing(key string, value any) AttributeSet {
	result := make(AttributeSet)
	for k, v := range a {
		result[k] = v
	}
	if _, ok := result[key]; !ok {
		result[key] = value
	}
	return result
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

func (a AttributeSet) Merge(attributes ...AttributeSet) AttributeSet {
	var result AttributeSet = make(map[string]any)
	for newKey, newValue := range a {
		result = result.SetAttribute(newKey, newValue)
	}
	for _, m := range attributes {
		for newKey, newValue := range m {
			result = result.SetAttribute(newKey, newValue)
		}
	}

	if len(result) == 0 {
		return NewEmptyAttributeSet()
	}

	return result
}

// Remove creates a new AttributeSet without the specified keys
func (a AttributeSet) Remove(keys []string) AttributeSet {
	result := make(AttributeSet)
	for k, v := range a {
		shouldRemove := false
		for _, removeKey := range keys {
			if k == removeKey {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			result[k] = v
		}
	}
	return result
}

// Includes checks if the attribute set includes the specified attribute name.
func (a AttributeSet) Includes(name string) bool {
	_, ok := a[name]
	return ok
}

// Keys returns the list of attribute names.
func (a AttributeSet) Keys() []string {
	keys := maps.Keys(a)
	return slices.Sorted(keys)
}

func (a AttributeSet) AsMap() map[string]any {
	return a
}

func (a AttributeSet) SetAttribute(name string, value any) AttributeSet {
	result := make(AttributeSet)
	for k, v := range a {
		result[k] = v
	}

	// Check if the attribute was already defined
	currentValue, ok := result[name]

	if !ok {
		result[name] = value
		return result
	}

	// If the type is a slice, append the new value instead of overriding
	switch x := currentValue.(type) {
	case []string:
		switch y := value.(type) {
		case []string:
			// Avoid duplicates when possible (ex: tags)
			newSlice := make([]string, len(x))
			copy(newSlice, x)
			for _, vy := range y {
				if !slices.Contains(newSlice, vy) {
					newSlice = append(newSlice, vy)
				}
			}
			result[name] = newSlice
		default:
			// Avoid duplicates (ex: tags)
			vy := fmt.Sprintf("%v", value)
			newSlice := make([]string, len(x))
			copy(newSlice, x)
			if !slices.Contains(newSlice, vy) {
				newSlice = append(newSlice, vy)
			}
			result[name] = newSlice
		}
	case []any:
		newSlice := make([]any, len(x))
		copy(newSlice, x)
		switch y := value.(type) {
		case []any:
			result[name] = append(newSlice, y...)
		default:
			result[name] = append(newSlice, value)
		}
	default:
		// override
		result[name] = value
	}
	return result
}

/* Special attributes */

func (a AttributeSet) Tags() TagSet {
	if v, ok := a["tags"].([]string); ok {
		return v
	}
	return nil
}

func (a AttributeSet) AddTag(newTag string) AttributeSet {
	result := make(AttributeSet)
	for k, v := range a {
		result[k] = v
	}

	if _, ok := result["tags"]; !ok {
		// No tag currently present
		result["tags"] = []string{newTag}
		return result
	}
	if tags, ok := result["tags"].([]string); ok {
		for _, tag := range tags {
			if tag == newTag {
				// Already present
				return result
			}
		}
		newTags := make([]string, len(tags))
		copy(newTags, tags)
		result["tags"] = append(newTags, newTag)
		return result
	}
	return result
}

func (a AttributeSet) AddTags(tags TagSet) AttributeSet {
	result := a
	for _, tag := range tags {
		result = result.AddTag(tag)
	}
	return result
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

func (a AttributeSet) AddHook(hookNames ...string) AttributeSet {
	result := make(AttributeSet)
	for k, v := range a {
		result[k] = v
	}

	if _, ok := result["hook"]; !ok {
		// No hook currently present
		result["hook"] = hookNames
		return result
	}
	if hooks, ok := result["hook"].([]string); ok {
		newHooks := make([]string, len(hooks))
		copy(newHooks, hooks)
		for _, hookName := range hookNames {
			if !slices.Contains(newHooks, hookName) {
				newHooks = append(newHooks, hookName)
			}
		}
		result["hook"] = newHooks
		return result
	}
	return result
}

// Attribution returns a formatted attribution string based on available attributes.
func (a AttributeSet) Attribution() string {
	name := a.CastValueAsString("name")
	occupation := a.CastValueAsString("occupation")
	nationality := a.CastValueAsString("nationality")

	if name == "" {
		return ""
	}

	var res strings.Builder
	res.WriteString("― ")
	res.WriteString(name)
	if occupation != "" {
		res.WriteString(", ")
		if nationality != "" {
			res.WriteString(nationality)
			res.WriteString(" ")
		}
		res.WriteString(occupation)
	}
	return res.String()
}

// ToMarkdownNotation renders attributes using The NoteWriter notation with shorthand support
func (a AttributeSet) ToMarkdownNotation(configAttributes ConfigAttributes) string {
	if len(a) == 0 {
		return ""
	}

	var rendered []string

	for name, value := range a {
		// Check if there's a shorthand available
		if attrDef, exists := configAttributes[name]; exists && attrDef.Shorthands != nil {
			// Look for a shorthand that matches the value
			valueStr := fmt.Sprintf("%v", value)
			found := false
			for shorthand, shorthandValue := range attrDef.Shorthands {
				if fmt.Sprintf("%v", shorthandValue) == valueStr {
					rendered = append(rendered, " "+shorthand)
					found = true
					break
				}
			}
			if found {
				continue
			}
		}

		// No shorthand found, use full attribute notation
		rendered = append(rendered, fmt.Sprintf(" `@%s: %v`", name, value))
	}

	return strings.Join(rendered, "")
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
	if len(a) == 0 {
		return "", nil
	}
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
	result := make(map[string]any)
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

// MustCastAttribute is like CastAttribute but panics if the value cannot be cast.
func MustCastAttribute(value any, declaredType ConfigAttribute) any {
	typedValue, ok := CastAttribute(value, declaredType)
	if !ok {
		panic(fmt.Sprintf("invalid value for attribute %v: %v", declaredType, value))
	}
	return typedValue
}

/*
 * EmojiSet
 */

type EmojiSet []string

// NewEmptyEmojiSet creates an empty emoji set.
func NewEmptyEmojiSet() EmojiSet {
	return []string{}
}

// NewEmojiSet creates a new emoji set removing duplicate emojis.
func NewEmojiSet(emojis []string) EmojiSet {
	result := NewEmptyEmojiSet().Merge(emojis)
	slices.Sort(result)
	return result
}

// NewEmojiSetFromText extracts emojis from a text.
func NewEmojiSetFromText(content string) EmojiSet {
	var emojis EmojiSet

	runes := []rune(content)

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Check for flag sequences (regional indicators)
		if r >= 0x1F1E0 && r <= 0x1F1FF && i+1 < len(runes) {
			next := runes[i+1]
			if next >= 0x1F1E0 && next <= 0x1F1FF {
				// This is a country flag emoji (two regional indicators)
				emojis = append(emojis, string([]rune{r, next}))
				i++ // Skip the next rune since we processed it
				continue
			}
		}

		// Check for other emoji ranges
		if (r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
			(r >= 0x1F300 && r <= 0x1F5FF) || // Misc Symbols and Pictographs
			(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map
			(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols and Pictographs
			(r >= 0x2600 && r <= 0x26FF) || // Misc symbols
			(r >= 0x2700 && r <= 0x27BF) { // Dingbats
			emojis = append(emojis, string(r))
		}
	}

	return NewEmojiSet(emojis)
}

// Merge merges multiple emoji sets into a new one, removing duplicates.
func (e EmojiSet) Merge(emojiSets ...EmojiSet) EmojiSet {
	result := NewEmptyEmojiSet()

	// Start with initial set
	result = append(result, e...)

	// Append new emojis in other sets
	for _, emojis := range emojiSets {
		for _, emoji := range emojis {
			if !slices.Contains(result, emoji) {
				result = append(result, emoji)
			}
		}
	}
	return result
}

func (e EmojiSet) AsList() []string {
	return e
}

func (e EmojiSet) Includes(emoji string) bool {
	return slices.Contains(e, emoji)
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
func ExtractBlockTagsAndAttributes(content markdown.Document, configAttributes ConfigAttributes) (TagSet, AttributeSet) {

	// Collect tags and attributes
	var tags TagSet
	var attributes AttributeSet = make(map[string]any)

	for _, line := range content.Lines() {

		// empty or only tags and attributes?
		if line.IsBlank() || !OnlyTagsAndAttributes(line.Text) {
			continue
		}

		// Append tags and attributes to collected ones
		matches := regexTags.FindAllStringSubmatch(line.Text, -1)
		for _, match := range matches {
			tag := match[1]

			// Append new tag
			tags = append(tags, tag)
		}
		matches = regexAttributes.FindAllStringSubmatch(line.Text, -1)
		for _, match := range matches {
			name := match[1]
			value := match[2]

			if _, exists := attributes[name]; exists {
				// Attribute already exists, convert to array
				currentValue := attributes[name]
				if IsArray(currentValue) {
					// Already an array, append
					attributes[name] = append(UnpackArray(currentValue), value)
				} else {
					// Convert to array
					attributes[name] = []any{currentValue, value}
				}
			} else {
				attributes[name] = value
			}
		}
	}

	// Cast (ensure the tags attribute is an array too)
	attributes = attributes.CastOrIgnore(configAttributes)

	tagsInAttributes := attributes.Tags()

	// The tag syntax is only syntax sugar. Tags must be added in attributes too.
	for _, tag := range tags {
		attributes = attributes.AddTag(tag)
	}

	// Add tags declared using `@tags` attributes
	tags = append(tags, tagsInAttributes...)

	return tags, attributes
}

// ExtractTags extracts tags from a text
func ExtractTags(doc markdown.Document) TagSet {
	content := string(doc)
	matches := regexTags.FindAllStringSubmatch(content, -1)
	var tags TagSet
	for _, match := range matches {
		tag := match[1]
		// Append new tag
		tags = append(tags, tag)
	}
	return NewTagSet(tags)
}

// ExtractOnlyAttributes extracts attributes from a text (ignoring tags and shorthands)
func ExtractOnlyAttributes(doc markdown.Document, configAttributes ConfigAttributes) AttributeSet {
	content := string(doc)
	matches := regexAttributes.FindAllStringSubmatch(content, -1)
	attributes := make(AttributeSet)
	for _, match := range matches {
		name := match[1]
		value := match[2]
		attributes[name] = value
	}
	// Cast (ensure the tags attribute is an array too)
	attributes = attributes.CastOrIgnore(configAttributes)
	return attributes
}

// ExtractAttributes extracts all attributes from a text (including tags and shorthands)
func ExtractAttributes(doc markdown.Document, configAttributes ConfigAttributes, configTags ConfigTags) AttributeSet {
	attributes := ExtractOnlyAttributes(doc, configAttributes)
	tags := ExtractTags(doc)
	shorthandAttributes, shorthandTags := ExtractShorthands(doc, configAttributes, configTags)
	allTags := tags.Merge(shorthandTags)
	attributes = attributes.AddTags(allTags)
	return attributes.Merge(shorthandAttributes)
}

// ExtractShorthands extracts shorthand attributes and tags from a string
func ExtractShorthands(doc markdown.Document, configAttributes ConfigAttributes, configTags ConfigTags) (AttributeSet, TagSet) {
	content := string(doc)

	shorthandAttributes := make(AttributeSet)

	for _, attribute := range configAttributes {
		if len(attribute.Shorthands) == 0 && attribute.ShorthandPattern == "" {
			continue
		}

		// If a regex pattern is defined, use it to find the shorthand value
		if attribute.ShorthandPattern != "" {
			re := regexp.MustCompile(attribute.ShorthandPattern)
			matches := re.FindStringSubmatch(content)
			if len(matches) > 1 {
				shorthandValue := matches[1]
				typedValue := MustCastAttribute(shorthandValue, *attribute)
				shorthandAttributes[attribute.Name] = typedValue
			}
		}

		// Sort shorthand keys by length (longest first) to match longer patterns first
		var sortedKeys []string
		for shorthandKey := range attribute.Shorthands {
			sortedKeys = append(sortedKeys, shorthandKey)
		}
		// Simple sort by length (longest first)
		for i := 0; i < len(sortedKeys); i++ {
			for j := i + 1; j < len(sortedKeys); j++ {
				if len(sortedKeys[i]) < len(sortedKeys[j]) {
					sortedKeys[i], sortedKeys[j] = sortedKeys[j], sortedKeys[i]
				}
			}
		}

		// Look for each shorthand key in the text (longest first)
		for _, shorthandKey := range sortedKeys {
			if strings.Contains(content, shorthandKey) {
				shorthandValue := attribute.Shorthands[shorthandKey]
				typedValue := MustCastAttribute(shorthandValue, *attribute)
				shorthandAttributes[attribute.Name] = typedValue
				break // Only match the first shorthand for this attribute
			}
		}
	}

	// Cast (ensure the tags attribute is an array too)
	shorthandAttributes = shorthandAttributes.CastOrIgnore(configAttributes)

	// Extract tag shorthands
	var shorthandTags TagSet
	for _, tag := range configTags {
		if tag.Shorthand == "" {
			continue
		}
		if strings.Contains(content, tag.Shorthand) {
			shorthandTags = append(shorthandTags, tag.Name)
		}
	}

	return shorthandAttributes, shorthandTags
}

// StripShorthands removes shorthands from text only if marked as non-preservable.
func StripShorthands(attributes ConfigAttributes, tags ConfigTags) markdown.DocumentTransformer {
	return func(document markdown.Document) (markdown.Document, error) {
		modifiedText := string(document)

		for _, attribute := range attributes {
			if attribute.PreserveShorthand == nil || *attribute.PreserveShorthand {
				continue
			}

			if len(attribute.Shorthands) == 0 && attribute.ShorthandPattern == "" {
				continue
			}

			// Strip using the regex pattern first
			if attribute.ShorthandPattern != "" {
				re := regexp.MustCompile(attribute.ShorthandPattern)
				modifiedText = re.ReplaceAllString(modifiedText, "")
			}

			// Sort shorthand keys by length (longest first) to remove longer patterns first
			var sortedKeys []string
			for shorthandKey := range attribute.Shorthands {
				sortedKeys = append(sortedKeys, shorthandKey)
			}
			// Simple sort by length (longest first)
			for i := 0; i < len(sortedKeys); i++ {
				for j := i + 1; j < len(sortedKeys); j++ {
					if len(sortedKeys[i]) < len(sortedKeys[j]) {
						sortedKeys[i], sortedKeys[j] = sortedKeys[j], sortedKeys[i]
					}
				}
			}

			// Remove shorthand keys from text (longest first)
			for _, shorthandKey := range sortedKeys {
				// Important: Shorthand keys must use the Unicode invisible codepoint U+FE0F
				// (used to request the emoji presentation of a character that can be displayed either as text or emoji)
				// See https://unicode.org/faq/emoji_dingbats.html)
				if strings.Contains(modifiedText, shorthandKey) {
					modifiedText = strings.ReplaceAll(modifiedText, shorthandKey, "")
					break // Only remove the first match for this attribute
				}
			}
		}

		// Strip tag shorthands that are not preserved
		for _, tag := range tags {
			if tag.PreserveShorthand == nil || *tag.PreserveShorthand {
				continue
			}
			if tag.Shorthand == "" {
				continue
			}
			if strings.Contains(modifiedText, tag.Shorthand) {
				modifiedText = strings.ReplaceAll(modifiedText, tag.Shorthand, "")
			}
		}

		// Clean up consecutive spaces and trim
		modifiedText = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(modifiedText, " "))

		return markdown.Document(modifiedText), nil
	}
}

func ExtractAllTagsAndAttributesAndEmojis(doc markdown.Document, configAttributes ConfigAttributes, configTags ConfigTags) (TagSet, AttributeSet, EmojiSet) {
	content := string(doc)

	tags := NewTagSetFromText(content, configAttributes, configTags)
	attributes := NewAttributeSetFromText(content, configAttributes, configTags)
	attributes = attributes.AddTags(tags)

	trimmedText := doc.MustTransform(
		StripTags(),
		StripAttributes(configAttributes, configTags)).
		TrimSpace()
	emojis := NewEmojiSetFromText(string(trimmedText))

	return tags, attributes, emojis
}

// StripTagsAndAttributes remove all tags and attributes.
func StripBlockTagsAndAttributes() markdown.DocumentTransformer {
	return func(document markdown.Document) (markdown.Document, error) {
		var res bytes.Buffer
		for _, line := range document.Lines() {
			// not only tags and attributes?
			if line.IsBlank() || line.InsideCodeBlock || !regexBlockTagAttributesLine.MatchString(line.Text) {
				res.WriteString(line.Text + "\n")
			}
		}
		return markdown.Document(text.SquashBlankLines(res.String())).TrimSpace(), nil
	}
}

// StripTags removes all tags from a text.
func StripTags() markdown.DocumentTransformer {
	return func(document markdown.Document) (markdown.Document, error) {
		var res bytes.Buffer
		for _, line := range document.Lines() {
			newLine := line.Text
			if !line.InsideCodeBlock {
				newLine = regexTags.ReplaceAllLiteralString(newLine, "")
				newLine = text.SquashConsecutiveSpaces(newLine)
			}
			if !text.IsBlank(newLine) {
				res.WriteString(newLine + "\n")
			}
		}
		return markdown.Document(text.SquashBlankLines(res.String())).TrimSpace(), nil
	}
}

// StripOnlyAttributes removes all attributes from a text.
func StripOnlyAttributes() markdown.DocumentTransformer {
	return func(document markdown.Document) (markdown.Document, error) {
		var res bytes.Buffer
		for _, line := range document.Lines() {
			newLine := line.Text
			if !line.InsideCodeBlock {
				newLine = regexAttributes.ReplaceAllLiteralString(newLine, "")
				newLine = text.SquashConsecutiveSpaces(newLine)
			}
			if !text.IsBlank(newLine) {
				res.WriteString(newLine + "\n")
			}
		}
		newContent := markdown.Document(text.SquashBlankLines(res.String())).TrimSpace()
		return newContent, nil
	}
}

// StripAttributes removes all attributes from a text.
func StripAttributes(attributes ConfigAttributes, tags ConfigTags) markdown.DocumentTransformer {
	return func(document markdown.Document) (markdown.Document, error) {
		return document.MustTransform(
			StripOnlyAttributes(),
			StripShorthands(attributes, tags),
		).TrimSpace(), nil
	}
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
func IsPrimitive(value any) bool {
	return slices.Contains(primitiveDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsComposite returns if a variable is a composite type.
func IsComposite(value any) bool {
	return slices.Contains(compositeDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsArray returns if a variable is a JSON array.
func IsArray(value any) bool {
	return slices.Contains(arrayDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsObject returns if a variable is a JSON map.
func IsObject(value any) bool {
	return slices.Contains(objectDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsNumber returns if a variable is a JSON number.
func IsNumber(value any) bool {
	return slices.Contains(numberDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsInteger returns if a variable is a JSON number of integer type.
func IsInteger(value any) bool {
	return slices.Contains(integerDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsFloat returns if a variable is a JSON number of float type.
func IsFloat(value any) bool {
	return slices.Contains(floatDataTypeKinds, reflect.TypeOf(value).Kind())
}

// IsBool returns if a variable is a JSON boolean.
func IsBool(value any) bool {
	return reflect.Bool == reflect.TypeOf(value).Kind()
}

// IsString returns if a variable is a JSON string.
func IsString(value any) bool {
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
