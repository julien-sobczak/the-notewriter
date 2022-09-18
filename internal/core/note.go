package core

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
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

type Note struct {
	// A unique identifier among all notes
	ID string
	// The kind of note
	Kind NoteKind
	// A relative path to the collection directory
	RelativePath string
	// The FrontMatter for the note file
	FrontMatter map[string]interface{}
	// The order of field when marshalling the FrontMatter
	FrontMatterOrder []string

	// TODO split Content into a list of notes???? Or create a method GetSubNotes()???
	Content string
}

func (n *Note) Save() error {
	// TODO Persist to disk
	var sb strings.Builder
	sb.WriteString("---\n")

	var frontMatterContent []*yaml.Node
	for _, fieldName := range n.FrontMatterOrder {
		value, ok := n.FrontMatter[fieldName]
		if !ok {
			continue
		}
		var fieldNode yaml.Node
		fieldDoc, err := yaml.Marshal(value)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(fieldDoc, &fieldNode)
		if err != nil {
			return err
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
	frontMatter := yaml.Node{
		Kind:    yaml.MappingNode,
		Content: frontMatterContent,
	}

	// d, err := yaml.Marshal(frontMatter) // FIXME sort fields in a well-defined order: title, author, year, ..., details.
	// if err != nil {
	// 	return err
	// }

	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(Indent)
	bufEncoder.Encode(frontMatter)
	sb.WriteString(CompactYAML(buf.String()))
	sb.WriteString("---\n")
	sb.WriteString(n.Content)
	fmt.Println(sb.String())
	return nil
}

// CompactYAML removes leading spaces in front of sequences.
//
// Ex:
//
//   doc:
//     - toto: tata
//
// Becomes
//
//   doc:
//   - toto: tata
func CompactYAML(doc string) string {
	// Identing sequences using zero-space (compact form) is not supported:
	// https://github.com/go-yaml/yaml/issues/661
	var buf bytes.Buffer
	r, _ := regexp.Compile(`^(\s*)  (- .*)$`)
	insideSequence := false
	var leadingSpaces string // the spaces prefix for successive lines in the sequence
	for _, line := range strings.Split(strings.TrimSuffix(doc, "\n"), "\n") {
		if r.MatchString(line) {
			rs := r.FindStringSubmatch(line)
			buf.WriteString(rs[1] + rs[2])
			buf.WriteString("\n")
			insideSequence = true
			leadingSpaces = rs[1] + "    "
		} else if insideSequence && strings.HasPrefix(line, leadingSpaces) {
			buf.WriteString(line[Indent:])
			buf.WriteString("\n")
		} else {
			buf.WriteString(line)
			buf.WriteString("\n")
			insideSequence = false
			leadingSpaces = ""
		}
	}
	return buf.String()
}
