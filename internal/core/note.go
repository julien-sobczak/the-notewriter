package core

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const Indent int = 2

type NoteKind int64

const (
	KindReference  NoteKind = 0
	KindNote       NoteKind = 1
	KindFlashcard  NoteKind = 2
	KindCheatsheet NoteKind = 3
	KindJournal    NoteKind = 4
	KindTodo       NoteKind = 5
)

type File struct {
	// A unique identifier among all notes
	ID string
	// The kind of note
	Kind NoteKind
	// A relative path to the collection directory
	RelativePath string
	// The FrontMatter for the note file
	Attributes AttributeList

	// TODO split Content into a list of notes???? Or create a method GetSubNotes()???
	Content string
}

type Attribute struct {
	Name          string
	Value         interface{}
	OriginalValue interface{}
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
		fieldName := attribute.Name
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
