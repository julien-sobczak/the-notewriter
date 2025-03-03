package main

import (
	"strings"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
)

// Lite version of internal/core/parser.go

/* ParsedFile */

type ParsedFile struct {
	Markdown *markdown.File

	// The paths to the file
	AbsolutePath string
	RelativePath string

	// Notes inside the file
	Notes []*ParsedNote
}

// ParseFile contains the main logic to parse a raw note file.
func ParseFile(relativePath string, md *markdown.File) (*ParsedFile, error) {
	result := &ParsedFile{
		Markdown:     md,
		AbsolutePath: md.AbsolutePath,
		RelativePath: relativePath,
	}

	// Extract objects
	notes, err := result.extractNotes()
	if err != nil {
		return nil, err
	}
	result.Notes = notes
	return result, nil
}

/* ParsedNote */

// ParsedNote represents a single raw note inside a file.
type ParsedNote struct {
	Title   string
	Content string
}

// ParseNotes extracts the notes from a file body.
func ParseNotes(fileBody string) []*ParsedNote {
	var noteTitle string
	var noteContent strings.Builder

	var results []*ParsedNote

	for _, line := range strings.Split(fileBody, "\n") {
		// Minimalist implementation. Only search for ## headings
		if strings.HasPrefix(line, "## ") {
			if noteTitle != "" {
				results = append(results, &ParsedNote{
					Title:   noteTitle,
					Content: strings.TrimSpace(noteContent.String()),
				})
			}
			noteTitle = strings.TrimPrefix(line, "## ")
			noteContent.Reset()
			continue
		}

		if noteTitle != "" {
			noteContent.WriteString(line)
			noteContent.WriteRune('\n')
		}
	}
	if noteTitle != "" {
		results = append(results, &ParsedNote{
			Title:   noteTitle,
			Content: strings.TrimSpace(noteContent.String()),
		})
	}

	return results
}

func (p *ParsedFile) extractNotes() ([]*ParsedNote, error) {
	// All notes collected until now
	var notes []*ParsedNote

	sections, err := p.Markdown.GetSections()
	if err != nil {
		return nil, nil
	}

	for _, section := range sections {
		// Minimalist implementation. Only search for ## headings
		if section.HeadingLevel != 2 {
			continue
		}

		title := section.HeadingText
		body := section.ContentText

		notes = append(notes, &ParsedNote{
			Title:   title.String(),
			Content: strings.TrimSpace(body.String()),
		})
	}

	return notes, nil
}
