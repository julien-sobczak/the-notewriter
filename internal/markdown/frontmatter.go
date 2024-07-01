package markdown

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// Default indentation in front matter
const Indent int = 2

// FrontMatter represents the Front Matter
type FrontMatter string

func (f FrontMatter) AsNode() (*yaml.Node, error) {
	var frontMatter = new(yaml.Node)
	if err := yaml.Unmarshal([]byte(f), frontMatter); err != nil {
		return nil, err
	}
	if frontMatter.Kind > 0 { // Happen when no Front Matter is present
		frontMatter = frontMatter.Content[0]
	}
	return frontMatter, nil
}

func (f FrontMatter) AsMap() (map[string]any, error) {
	var attributes = make(map[string]any)
	if err := yaml.Unmarshal([]byte(f), attributes); err != nil {
		return nil, err
	}
	return attributes, nil
}

// AsBeautifulYAML formats the current attributes to the YAML front matter format.
func (f FrontMatter) AsBeautifulYAML() (string, error) {
	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(Indent)
	m, err := f.AsMap()
	if err != nil {
		return "", err
	}
	if err = bufEncoder.Encode(m); err != nil {
		return "", err
	}
	return CompactYAML(buf.String()), nil
}

func (f FrontMatter) Cast( /* schema */ ) (FrontMatter, error) {
	// TODO to implement
	return f, nil
}
