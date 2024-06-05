package core

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"gopkg.in/yaml.v3"
)

type MarkdownFile struct {
	absolutePath   string
	content        []byte
	lstat          fs.FileInfo
	stat           fs.FileInfo
	frontMatter    string
	body           string
	bodyLineNumber int
}

type MarkdownSection struct {
	Parent              *MarkdownSection
	HeadingText         string
	HeadingLevel        int
	ContentText         string
	FileLineNumberStart int // 1-based index based on Markdown file
	FileLineNumberEnd   int
	BodyLineNumberStart int // 1-based index based on body (ignored the Front Matter)
	BodyLineNumberEnd   int
}

func (m MarkdownSection) String() string {
	return fmt.Sprintf("%s %s", strings.Repeat("#", m.HeadingLevel), m.HeadingText)
}

// ParseMarkdownFile parses a Markdown file.
func ParseMarkdownFile(path string) (*MarkdownFile, error) {
	relativePath, err := CurrentRepository().GetFileRelativePath(path)
	if err != nil {
		return nil, err
	}
	absolutePath := CurrentRepository().GetAbsolutePath(relativePath)

	lstat, err := os.Lstat(absolutePath)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(absolutePath)
	if err != nil {
		return nil, err
	}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawFrontMatter bytes.Buffer
	var rawBody bytes.Buffer
	frontMatterStarted := false
	frontMatterEnded := false
	bodyStarted := false
	bodyLineNumber := 0
	for i, line := range strings.Split(strings.TrimSuffix(string(contentBytes), "\n"), "\n") {
		if strings.HasPrefix(line, "---") {
			if bodyStarted {
				// Flashcard Front/Back line separator
				rawBody.WriteString(line)
				rawBody.WriteString("\n")
			} else if !frontMatterStarted {
				frontMatterStarted = true
			} else if !frontMatterEnded {
				frontMatterEnded = true
			}
			continue
		}

		if frontMatterStarted && !frontMatterEnded {
			rawFrontMatter.WriteString(line)
			rawFrontMatter.WriteString("\n")
		} else {
			if !text.IsBlank(line) && !bodyStarted {
				bodyStarted = true
				bodyLineNumber = i + 1
			}
			if bodyStarted {
				rawBody.WriteString(line)
				rawBody.WriteString("\n")
			}
		}
	}

	return &MarkdownFile{
		absolutePath:   absolutePath,
		content:        contentBytes,
		lstat:          lstat,
		stat:           stat,
		frontMatter:    rawFrontMatter.String(),
		body:           rawBody.String(),
		bodyLineNumber: bodyLineNumber,
	}, nil
}

func (f *MarkdownFile) AbsolutePath() string {
	return f.absolutePath
}

func (f *MarkdownFile) FrontMatterAsNode() (*yaml.Node, error) {
	var frontMatter = new(yaml.Node)
	if err := yaml.Unmarshal([]byte(f.frontMatter), frontMatter); err != nil {
		return nil, err
	}
	if frontMatter.Kind > 0 { // Happen when no Front Matter is present
		frontMatter = frontMatter.Content[0]
	}
	return frontMatter, nil
}

func (f *MarkdownFile) FrontMatterAsMap() (map[string]interface{}, error) {
	var attributes = make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(f.frontMatter), attributes); err != nil {
		return nil, err
	}
	return attributes, nil
}

func (m *MarkdownFile) FrontMatter() string {
	return m.frontMatter
}

func (m *MarkdownFile) Body() string {
	return m.body
}

func (m *MarkdownFile) BodyLineNumber() int {
	return m.bodyLineNumber
}

func (m *MarkdownFile) LastUpdateDate() time.Time {
	return m.lstat.ModTime()
}

func (m *MarkdownFile) WalkSections(walkFn func(parent *MarkdownSection, current *MarkdownSection, children []*MarkdownSection) error) error {
	sections, err := m.GetSections()
	if err != nil {
		return err
	}
	for _, current := range sections {
		var children []*MarkdownSection
		for _, otherSection := range sections {
			if otherSection.Parent != nil && otherSection.Parent == current {
				children = append(children, otherSection)
			}
		}
		if err := walkFn(current.Parent, current, children); err != nil {
			return err
		}
	}
	return nil
}

func (m *MarkdownFile) GetSections() ([]*MarkdownSection, error) {
	var sections []*MarkdownSection
	var lastSectionAtLevel [10]*MarkdownSection

	lines := strings.Split(m.body, "\n")

	// Current line number during the parsing
	var lineNumber int

	// Beware to ignore '#' in code blocks
	insideCodeBlock := false

	for i, line := range lines {
		lineNumber = i + 1 // lines are 1-based
		if strings.HasPrefix(line, "```") {
			insideCodeBlock = !insideCodeBlock
		}
		if insideCodeBlock {
			// Ignore possible Markdown heading in code blocks
			continue
		}

		if ok, headingText, headingLevel := markdown.IsHeading(line); ok {
			// Previous section to close?
			lastLevel := -1
			var lastSection *MarkdownSection
			if len(sections) > 0 {
				lastSection = sections[len(sections)-1]
				lastLevel = lastSection.HeadingLevel
			}
			if lastLevel >= headingLevel {
				// Close previous section(s)
				for _, section := range sections {
					if section.HeadingLevel >= headingLevel && section.BodyLineNumberEnd == 0 {
						section.FileLineNumberEnd = m.bodyLineNumber - 1 + lineNumber - 1
						section.BodyLineNumberEnd = lineNumber - 1
						section.ContentText = text.ExtractLines(m.body, section.BodyLineNumberStart, lineNumber-1)
					}
				}
			}

			// Start new section
			newSection := &MarkdownSection{
				HeadingText:         headingText,
				HeadingLevel:        headingLevel,
				FileLineNumberStart: m.bodyLineNumber - 1 + lineNumber,
				BodyLineNumberStart: lineNumber,
			}
			lastSectionAtLevel[headingLevel] = newSection

			// Top-section?
			if lastSectionAtLevel[headingLevel-1] != nil {
				// Parent found
				newSection.Parent = lastSectionAtLevel[headingLevel-1]
			}

			sections = append(sections, newSection)
		}
	}

	// Iterate over sections and use line numbers to split the raw content into notes
	if len(sections) == 0 {
		return nil, nil
	}

	// Complete unfinished section(s)
	for _, section := range sections {
		if section.BodyLineNumberEnd == 0 {
			section.FileLineNumberEnd = m.bodyLineNumber - 1 + lineNumber
			section.BodyLineNumberEnd = lineNumber
			section.ContentText = text.ExtractLines(m.body, section.BodyLineNumberStart, lineNumber)
		}
	}

	// Trim content
	for _, section := range sections {
		// Remove blank lines at the end of each section
		for {
			if strings.HasSuffix(section.ContentText, "\n") {
				section.ContentText = strings.TrimSuffix(section.ContentText, "\n")
				section.FileLineNumberEnd -= 1
				section.BodyLineNumberEnd -= 1
			} else {
				break
			}
		}
		// Remove spaces also at the space of each section
		section.ContentText = strings.TrimSpace(section.ContentText)
		// No need to update the "Start" index as it corresponds to the heading line number
	}

	return sections, nil
}

// TODO create parser.go with
// type ParsedFile {
//    ParsedNotes []*ParsedNote
//    ParsedMedias []*ParsedMedias
//    ParsedLinks []*ParsedLinks
// }

// TODO on ParsedFile() ParsedMedia(), add SHA1()

/*

File.GetObjects()

All objects (file, note, reminder, link) are present in a given file
Media objects can be referenced from multiple files
Notes are present inside a file (and inside another note)
Flashcard are indissociable from their note.
Reminder references a specific note (or a specific item in a note)
Go Links are present in a note but are independant.

NewParsedFileFromMarkdownFile(mdFile) *ParsedFile


MarkdownFile ------> ParsedFile ---------> File -------------> PackFile

I understand        I extract              Core logic          I bundle
Markdown syntax     _NoteWriter_ objects                       _NoteWriter_ objects


Option 1: Parsed when needed (ex: `GetLinks` on `Note`)
* Advantage(s):
  * `ParsedXXX` object transparent

Option 2: Parsed everything immediately (ex: `ParseFile` calls `ParseNote`. `ParseMedia`, etc.)
* Advantage(s):
  * Clear separation of logic (parsing <> database interaction)
  * Unique place to test parsing logic (without interaction with DB) (`parser_test.go`)
  * Easier interface for lint rules
* Drawback(s):
  * Useful parsing? (ex: in Linter => in practice, we can expect a rule to validate almost anything)

Decision: Option 2 wins

MarkdownFile / ParsedFile => Stateless
File/PackFile => Stateful

file := parsedFile.ToFile()
file := NewFileFromParsedFile(parsedFile)
file.Save()

packFile := file.ToFile()
packFile := NewPackFileFromFile(file)
packFile.Save() // Write to .nt/objects

packFile := NewPackFileFromPath(path)
file := packFile.ToFile()
file.Save() // refresh the DB


$ nt add
-> packFile.Save()
-> index.StagingArea

$ nt commit
-> index.StagingArea -> index // update object to packfile OID
-> gc()

$ nt reset
-> read index.StagingArea
  -> updated/deleted objects => reread last packfile based on index + .Save()
  -> added objects => read from DB + .Delete()

PackObject is an entry inside a PackFile

> The packfile is a single file containing the contents of all the objects that were removed from your filesystem. The index is a file that contains offsets into that packfile so you can quickly seek to a specific object.
=> Don't use binary files for debuggablity purposes. Use YAML file instead (even if performance are decreased)
*/
