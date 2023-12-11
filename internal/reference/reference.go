package reference

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// EvaluateTemplate evaluate a reference template, supporting additional custom functions.
func EvaluateTemplate(templateText string, result Result) (string, error) {
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
		"jq": func(data any, expr string) (any, error) {
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
	}
	tmpl, err := template.New("").Funcs(functions).Parse(templateText)
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
