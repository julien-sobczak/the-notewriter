package core

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const Indent int = 2

type File struct {
	// A unique identifier among all files
	ID int

	// A relative path to the collection directory
	RelativePath string

	// The FrontMatter for the note file
	frontMatter *yaml.Node
	Attributes  AttributeList

	// TODO create a method GetNotes()
	Content string

	CreatedAt *time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

type Attribute struct {
	Key       string
	Value     interface{}
	KeyNode   *yaml.Node
	ValueNode *yaml.Node
}

type AttributeList []*Attribute

func (l AttributeList) FrontMatterString() (string, error) {
	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(Indent)
	node, err := l.FrontMatterYAML()
	if err != nil {
		return "", err
	}
	bufEncoder.Encode(node)
	return CompactYAML(buf.String()), nil
}

func (l AttributeList) FrontMatterYAML() (*yaml.Node, error) {
	var frontMatterContent []*yaml.Node
	for _, attribute := range l {
		fieldName := attribute.Key
		value := attribute.Value

		var fieldNode yaml.Node
		fieldDoc, err := yaml.Marshal(value)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(fieldDoc, &fieldNode)
		if err != nil {
			return nil, err
		}

		frontMatterContent = append(frontMatterContent, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fieldName,
		})
		switch fieldNode.Kind {
		case yaml.DocumentNode:
			frontMatterContent = append(frontMatterContent, fieldNode.Content[0])
		case yaml.ScalarNode:
			frontMatterContent = append(frontMatterContent, &fieldNode)
		default:
			fmt.Printf("Unexcepted type %v\n", fieldNode.Kind)
			os.Exit(1)
		}
	}
	frontMatter := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: frontMatterContent,
	}
	return frontMatter, nil
}

func NewAttributeListFromString(rawYAML string) (AttributeList, error) {
	var node yaml.Node
	err := yaml.Unmarshal([]byte(rawYAML), &node)
	if err != nil {
		return nil, err
	}

	if node.Kind != yaml.DocumentNode {
		return nil, fmt.Errorf("unexcepted YAML structure %v", node.Kind)
	}

	mappingNode := node.Content[0]

	if mappingNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("unexcepted YAML structure %v", mappingNode.Kind)
	}

	for i := 0; i < len(mappingNode.Content)/2; i++ {
		// if attributeNode.K
		keyNode := mappingNode.Content[i*2]
		valueNode := mappingNode.Content[i*2+1]
		fmt.Printf("%v: %v\n", keyNode.Value, valueNode.Value)
	}

	var result AttributeList
	return result, nil
}

func NewFileFromPath(filepath string) (*File, error) {
	contentBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var rawFrontMatter bytes.Buffer
	var rawContent bytes.Buffer
	frontMatterStarted := false
	frontMatterEnded := false
	for _, line := range strings.Split(strings.TrimSuffix(string(contentBytes), "\n"), "\n") {
		if strings.HasPrefix(line, "---") {
			if !frontMatterStarted {
				frontMatterStarted = true
				continue
			} else {
				frontMatterEnded = true
				continue
			}
		}

		if frontMatterStarted && !frontMatterEnded {
			rawFrontMatter.WriteString(line)
			rawFrontMatter.WriteString("\n")
		} else {
			rawContent.WriteString(line)
			rawContent.WriteString("\n")
		}
	}

	attributes, err := NewAttributeListFromString(rawFrontMatter.String())
	if err != nil {
		return nil, err
	}

	return &File{
		// We ignore if the file already exists in database
		ID:        0,
		CreatedAt: nil,
		UpdatedAt: nil,
		DeletedAt: nil,
		// Reread the file
		RelativePath: filepath,
		Content:      strings.TrimSpace(rawContent.String()),
		Attributes:   attributes,
	}, nil
}

func (f *File) GetNotes() []*Note {
	// TODO implement
	return nil
}

func (f *File) Save() error {
	// TODO Persist to disk
	frontMatter, err := f.Attributes.FrontMatterString()
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(frontMatter)
	sb.WriteString("---\n")
	sb.WriteString(f.Content)
	fmt.Println(sb.String())
	return nil
}
