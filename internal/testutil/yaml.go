package testutil

import (
	"fmt"
	"io"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// AssertYAMLMatches compares a YAML document with a map template.
// It fails if an attribute is present in the YAML but not in the map, or vice versa.
// If a value in the map is a regexp.Regexp, the YAML value must be a string and match the regex.
func AssertYAMLMatches(t *testing.T, template map[string]any, r io.Reader) {
	t.Helper()

	err := CompareYAMLReader(template, r)
	assert.NoError(t, err, "YAML does not match template")
}

// RequireYAMLMatches is similar to AssertYAMLMatches but uses require.


func CompareYAMLReader(template map[string]any, r io.Reader) error {
	var doc map[string]any
	dec := yaml.NewDecoder(r)
	err := dec.Decode(&doc)
	if err != nil {
		return fmt.Errorf("failed to decode YAML: %v", err)
	}

	return compareYAMLMap(template, doc, "")
}

func compareYAMLMap(expected, actual map[string]any, path string) error {
	// Check for missing or extra keys
	for k, v := range expected {
		fullPath := k
		if path != "" {
			fullPath = path + "." + k
		}
		actualVal, ok := actual[k]
		if !ok {
			return fmt.Errorf("missing key in YAML: %s", fullPath)
		}
		if err := compareYAMLValue(v, actualVal, fullPath); err != nil {
			return err
		}
	}
	for k := range actual {
		fullPath := k
		if path != "" {
			fullPath = path + "." + k
		}
		_, ok := expected[k]
		if !ok {
			return fmt.Errorf("unexpected key in YAML: %s", fullPath)
		}
	}
	return nil
}

func compareYAMLValue(expected, actual any, path string) error {
	switch exp := expected.(type) {
	case *regexp.Regexp:
		str, ok := actual.(string)
		if !ok {
			return fmt.Errorf("expected string at %s for regexp match", path)
		}
		if !exp.MatchString(str) {
			return fmt.Errorf("value at %s does not match regexp: %q", path, str)
		}
	case map[string]any:
		actMap, ok := actual.(map[string]any)
		if !ok {
			return fmt.Errorf("expected map at %s", path)
		}
		if err := compareYAMLMap(exp, actMap, path); err != nil {
			return err
		}
	case []any:
		actSlice, ok := actual.([]any)
		if !ok {
			return fmt.Errorf("expected slice at %s", path)
		}
		if len(exp) != len(actSlice) {
			return fmt.Errorf("slice length mismatch at %s: expected %d, got %d", path, len(exp), len(actSlice))
		}
		for i := range exp {
			if err := compareYAMLValue(exp[i], actSlice[i], fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	default:
		// YAML v3 auto-converts ISO 8601 timestamps to time.Time.
		// First convert the time.Time to string before comparison.
		if t, ok := actual.(time.Time); ok {
			actual = t.Format(time.RFC3339)
		}

		if exp != actual {
			return fmt.Errorf("value mismatch at %s: expected %v, got %v", path, exp, actual)
		}
	}
	return nil
}
