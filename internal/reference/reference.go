package reference

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/itchyny/gojq"
	"github.com/julien-sobczak/the-notewriter/pkg/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"gopkg.in/yaml.v3"
)

type Result interface {
	Description() string
	Attributes() map[string]any
}

// Manager retrieves the metadata for references.
type Manager interface {
	Ready() (bool, error)

	// Search returns the best matching reference.
	Search(query string) ([]Result, error)
}

// ParseTemplate parses a reference template, supporting additional custom functions.
func ParseTemplate(templateText string) (*template.Template, error) {
	// Add additional functions in complement to standard functions
	// See https://pkg.go.dev/text/template#hdr-Functions
	//
	// See also Consul Template for inspiration
	// https://github.com/hashicorp/consul-template/blob/main/template/funcs.go
	// https://github.com/hashicorp/consul-template/blob/main/docs/templating-language.md#join
	functions := template.FuncMap{
		"json": func(data any) (string, error) {
			jsonData, err := json.Marshal(data)
			if err != nil {
				return "", err
			}
			return string(jsonData), nil
		},
		"jsonPretty": func(data any) (string, error) {
			jsonData, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return "", err
			}
			return string(jsonData), nil
		},
		"yaml": func(data any) (string, error) {
			yamlData, err := yaml.Marshal(data)
			if err != nil {
				return "", err
			}
			return string(yamlData), nil
		},
		"slug": func(data any) string {
			txt := fmt.Sprintf("%s", data)
			return markdown.Slug(txt)
		},
		"jq": func(expr string, data any) (any, error) {
			query, err := gojq.Parse(expr)
			if err != nil {
				return nil, err
			}
			iter := query.Run(data)
			var values []any
			for {
				v, ok := iter.Next()
				if !ok {
					break
				}
				if err, ok := v.(error); ok {
					return nil, err
				}
				values = append(values, v)
			}
			if len(values) == 1 {
				return values[0], nil
			}
			return values, nil
		},
		"title": func(data any) string {
			txt := fmt.Sprintf("%s", data)
			return text.ToBookTitle(txt)
		},
		// join is a templating version of strings.Join
		"join": func(sep string, data any) (string, error) {
			if v, ok := data.(string); ok {
				return v, nil
			}
			if v, ok := data.([]string); ok {
				return strings.Join(v, sep), nil
			}
			if rawValues, ok := data.([]interface{}); ok {
				var v []string
				for _, rawValue := range rawValues {
					if typedValue, ok := rawValue.(string); ok {
						v = append(v, typedValue)
					}
				}
				return strings.Join(v, sep), nil
			}
			return "", errors.New("unsupported type for join")
		},
	}
	tmpl, err := template.New("").Funcs(functions).Parse(templateText)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

// EvaluateTemplate evaluate a reference template, supporting additional custom functions.
func EvaluateTemplate(templateText string, result Result) (string, error) {
	tmpl, err := ParseTemplate(templateText)
	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, result.Attributes())
	if err != nil {
		return "", err
	}
	return tpl.String(), nil
}
