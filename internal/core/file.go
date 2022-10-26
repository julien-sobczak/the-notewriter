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
	ID int64

	// A relative path to the collection directory
	RelativePath string

	// The FrontMatter for the note file
	frontMatter *yaml.Node

	// TODO create a method GetNotes()
	Content string
	notes   []*Note

	CreatedAt *time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

func (f *File) FrontMatterString() (string, error) {
	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(Indent)
	err := bufEncoder.Encode(f.frontMatter)
	if err != nil {
		return "", err
	}
	return CompactYAML(buf.String()), nil
}

func (f *File) GetAttribute(key string) interface{} {
	if f.frontMatter == nil {
		return nil
	}
	for i := 0; i < len(f.frontMatter.Content); i++ {
		keyNode := f.frontMatter.Content[i*2]
		valueNode := f.frontMatter.Content[i*2+1]
		if keyNode.Value != key {
			continue
		}
		return toSafeYAMLValue(valueNode)
	}

	// Not found
	return nil
}

func (f *File) SetAttribute(key string, value interface{}) {
	if f.frontMatter == nil {
		var frontMatterContent []*yaml.Node
		f.frontMatter = &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: frontMatterContent,
		}
	}

	found := false
	for i := 0; i < len(f.frontMatter.Content)/2; i++ {
		keyNode := f.frontMatter.Content[i*2]
		valueNode := f.frontMatter.Content[i*2+1]
		if keyNode.Value != key {
			continue
		}

		found = true

		newValueNode := toSafeYAMLNode(value)
		if newValueNode.Kind == yaml.ScalarNode {
			valueNode.Value = newValueNode.Value
		} else if newValueNode.Kind == yaml.DocumentNode {
			valueNode.Content = newValueNode.Content[0].Content
		} else {
			valueNode.Content = newValueNode.Content
		}
	}

	if !found {
		f.frontMatter.Content = append(f.frontMatter.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
		})
		newValueNode := toSafeYAMLNode(value)
		switch newValueNode.Kind {
		case yaml.DocumentNode:
			f.frontMatter.Content = append(f.frontMatter.Content, newValueNode.Content[0])
		case yaml.ScalarNode:
			f.frontMatter.Content = append(f.frontMatter.Content, newValueNode)
		default:
			fmt.Printf("Unexcepted type %v\n", newValueNode.Kind)
			os.Exit(1)
		}
	}
}

func (f *File) GetAttributes() map[string]interface{} {
	if f.frontMatter == nil {
		return nil
	}

	result := make(map[string]interface{})
	for i := 0; i < len(f.frontMatter.Content); i++ {
		keyNode := f.frontMatter.Content[i*2]
		valueNode := f.frontMatter.Content[i*2+1]
		result[keyNode.Value] = toSafeYAMLValue(valueNode)
	}

	return result
}

func (f *File) GetNotes() []*Note {
	if f.notes != nil {
		return f.notes
	}

	var notes []*Note

	var currentNote bytes.Buffer
	var currentNoteTitle string
	var currentLevel int
	var lineNumber int
	var linesCountInCurrentNote int

	for _, line := range strings.Split(strings.TrimSuffix(f.Content, "\n"), "\n") {
		lineNumber++
		if ok, text, level := isHeading(line); ok { // New section = new potential note?
			ok, kind := isSupportedNote(text)
			if !ok {
				// Just a subsection, not a new note
				continue
			}

			// Close previous note
			if linesCountInCurrentNote > 0 {
				note := NewNote(f, kind, currentNoteTitle, currentNote.String(), lineNumber)
				notes = append(notes, note)
				// Reset
				currentNote.Reset()
				currentNoteTitle = ""
				currentLevel = 0
				lineNumber = 0
				linesCountInCurrentNote = 0
			}

			// Start new note
			if kind != KindFree && level > currentLevel {
				currentNote.WriteString(line)
				linesCountInCurrentNote++
			}
		} else {
			// Do not bother to append the line if no note is started
			if linesCountInCurrentNote > 0 {
				currentNote.WriteString(line)
				linesCountInCurrentNote++
			}
		}

	}

	if len(notes) > 0 {
		f.notes = notes
	}
	return f.notes
}

func NewEmptyFile() *File {
	return &File{}
}

func NewFileFromAttributes(attributes []Attribute) *File {
	file := &File{}
	for _, attribute := range attributes {
		file.SetAttribute(attribute.Key, attribute.Value)
	}
	return file
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

	var frontMatter yaml.Node
	err = yaml.Unmarshal(rawFrontMatter.Bytes(), &frontMatter)
	if err != nil {
		return nil, err
	}

	file := &File{
		// We ignore if the file already exists in database
		ID:        0,
		CreatedAt: nil,
		UpdatedAt: nil,
		DeletedAt: nil,
		// Reread the file
		RelativePath: filepath,
		Content:      strings.TrimSpace(rawContent.String()),
		frontMatter:  frontMatter.Content[0],
	}

	return file, nil
}

func (f *File) Save() error {
	// TODO Persist to disk
	frontMatter, err := f.FrontMatterString()
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

// isHeading returns if a givne line is a Markdown heading and its level.
func isHeading(line string) (bool, string, int) {
	if !strings.HasPrefix(line, "#") {
		return false, "", 0
	}
	if strings.HasPrefix(line, "###### ") {
		return true, strings.TrimPrefix(line, "###### "), 6
	} else if strings.HasPrefix(line, "##### ") {
		return true, strings.TrimPrefix(line, "##### "), 5
	} else if strings.HasPrefix(line, "#### ") {
		return true, strings.TrimPrefix(line, "#### "), 4
	} else if strings.HasPrefix(line, "### ") {
		return true, strings.TrimPrefix(line, "### "), 3
	} else if strings.HasPrefix(line, "## ") {
		return true, strings.TrimPrefix(line, "## "), 2
	} else if strings.HasPrefix(line, "# ") {
		return true, strings.TrimPrefix(line, "# "), 1
	}

	return false, "", 0
}

func isSupportedNote(text string) (bool, NoteKind) {
	if regexNote.MatchString(text) {
		return true, KindNote
	}
	if regexCheatsheet.MatchString(text) {
		return true, KindCheatsheet
	}
	if regexFlashcard.MatchString(text) {
		return true, KindFlashcard
	}
	if regexQuote.MatchString(text) {
		return true, KindQuote
	}
	return false, KindFree
}
