package markdown

import (
	"gopkg.in/yaml.v3"
)

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

func (f FrontMatter) AsMap() (map[string]interface{}, error) {
	var attributes = make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(f), attributes); err != nil {
		return nil, err
	}
	return attributes, nil
}

func (f FrontMatter) Cast( /* schema */ ) (FrontMatter, error) {
	// TODO to implement
	return f, nil
}
