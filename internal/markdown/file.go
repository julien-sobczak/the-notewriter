package markdown

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

type File struct {
	AbsolutePath string
	Content      []byte
	LStat        fs.FileInfo
	Stat         fs.FileInfo
	FrontMatter  FrontMatter
	Body         Document
	BodyLine     int
}

func (m File) String() string {
	return fmt.Sprintf("Markdown file %q", m.AbsolutePath)
}

type Section struct {
	Parent        *Section
	HeadingText   Document
	HeadingLevel  int
	ContentText   Document
	FileLineStart int // 1-based index based on Markdown file
	FileLineEnd   int
	BodyLineStart int // 1-based index based on body (ignored the Front Matter)
	BodyLineEnd   int
}

func (m Section) String() string {
	return fmt.Sprintf("%s %s", strings.Repeat("#", m.HeadingLevel), m.HeadingText)
}

// ParseFile parses a Markdown file.
func ParseFile(path string) (*File, error) {
	lstat, err := filesystem.Lstat(path)
	if err != nil {
		return nil, err
	}

	stat, err := filesystem.Stat(path)
	if err != nil {
		return nil, err
	}

	contentAsBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawFrontMatter bytes.Buffer
	var rawBody bytes.Buffer
	frontMatterStarted := false
	frontMatterEnded := false
	bodyStarted := false
	bodyLine := 0
	for i, line := range strings.Split(strings.TrimSuffix(string(contentAsBytes), "\n"), "\n") {
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
				bodyLine = i + 1
			}
			if bodyStarted {
				rawBody.WriteString(line)
				rawBody.WriteString("\n")
			}
		}
	}

	return &File{
		AbsolutePath: path,
		Content:      contentAsBytes,
		LStat:        lstat,
		Stat:         stat,
		FrontMatter:  FrontMatter(rawFrontMatter.String()),
		Body:         Document(rawBody.String()),
		BodyLine:     bodyLine,
	}, nil
}

func (m *File) LastUpdateDate() time.Time {
	return m.LStat.ModTime()
}

func (m *File) WalkSections(walkFn func(parent *Section, current *Section, children []*Section) error) error {
	sections, err := m.GetSections()
	if err != nil {
		return err
	}
	for _, current := range sections {
		var children []*Section
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

func (m *File) GetSections() ([]*Section, error) {
	var sections []*Section
	var lastSectionAtLevel [10]*Section

	lines := m.Body.Lines()

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

		if ok, headingText, headingLevel := IsHeading(line); ok {
			// Previous section to close?
			lastLevel := -1
			var lastSection *Section
			if len(sections) > 0 {
				lastSection = sections[len(sections)-1]
				lastLevel = lastSection.HeadingLevel
			}
			if lastLevel >= headingLevel {
				// Close previous section(s)
				for _, section := range sections {
					if section.HeadingLevel >= headingLevel && section.BodyLineEnd == 0 {
						section.FileLineEnd = m.BodyLine - 1 + lineNumber - 1
						section.BodyLineEnd = lineNumber - 1
						section.ContentText = m.Body.ExtractLines(section.BodyLineStart, lineNumber-1)
					}
				}
			}

			// Start new section
			newSection := &Section{
				HeadingText:   Document(headingText),
				HeadingLevel:  headingLevel,
				FileLineStart: m.BodyLine - 1 + lineNumber,
				BodyLineStart: lineNumber,
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
		if section.BodyLineEnd == 0 {
			section.FileLineEnd = m.BodyLine - 1 + lineNumber
			section.BodyLineEnd = lineNumber
			section.ContentText = m.Body.ExtractLines(section.BodyLineStart, lineNumber)
		}
	}

	// Trim content
	for _, section := range sections {
		// Remove blank lines at the end of each section
		trimmedContentText, _, nbLinesRemovedAtEnd := section.ContentText.TrimBlankLines()
		// No need to update the "Start" index as it corresponds to the heading line number
		section.ContentText = trimmedContentText
		section.FileLineEnd -= nbLinesRemovedAtEnd
		section.BodyLineEnd -= nbLinesRemovedAtEnd
	}

	return sections, nil
}

func (m *File) GetTopSection() (*Section, error) {
	sections, err := m.GetSections()
	if err != nil {
		return nil, err
	}
	if len(sections) == 0 {
		return nil, nil
	}
	return sections[0], nil
}
