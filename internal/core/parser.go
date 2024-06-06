package core

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

type ParsedFileNew struct {
	Markdown *MarkdownFile

	// Main Heading
	Slug       string
	Title      string
	ShortTitle string

	// File attributes extracted from the Front Matter
	FileAttributes map[string]interface{}

	// Extracted objects
	Notes  []*ParsedNoteNew
	Medias []*ParsedMediaNew
}

// ParsedNote represents a single raw note inside a file.
type ParsedNoteNew struct {
	Level int
	Kind  NoteKind

	// Heading
	Slug       string
	Title      string
	ShortTitle string

	Line           int
	Body           string
	NoteAttributes map[string]interface{}
	NoteTags       []string

	// Extracted objects
	Flashcard *ParsedFlashcardNew
	Links     []*ParsedLinkNew
	Reminders []*ParsedReminderNew
}

type ParsedFlashcardNew struct {
	// Short title of the note
	ShortTitle string

	// Fields in Markdown
	Front string
	Back  string
}

type ParsedLinkNew struct {
	// The link text
	Text string

	// The link destination
	URL string

	// The optional link title
	Title string

	// The optional GO name
	GoName string
}

type ParsedReminderNew struct {
	// Description in Markdown of the reminder (ex: the line)
	Description string

	// Tag value containig the formula to determine the next occurence
	Tag string `yaml:"tag"`
}

type ParsedMediaNew struct {
	// The path as specified in the file. (Ex: "../medias/pic.png")
	RawPath string
	// The absolute path
	AbsolutePath string
	// The file extension
	Extension string

	// Type of media
	MediaKind MediaKind

	// Media exists on disk
	Dangling bool
	// Content last modification date
	MTime time.Time
	// Size of the file
	Size int64
	// Permission of the file
	Mode fs.FileMode

	// Line number where the link present.
	Line int
}

func ParseFileFromMarkdownFile(md *MarkdownFile) (*ParsedFileNew, error) {
	// Extract file attributes
	frontMatter, err := md.FrontMatterAsMap()
	if err != nil {
		return nil, err
	}
	fileAttributes := CastAttributes(frontMatter, GetSchemaAttributeTypes())

	// Extract titles
	topSection, err := md.GetTopSection()
	if err != nil {
		return nil, err
	}
	title := ""
	if topSection != nil {
		title = topSection.HeadingText
	}
	_, _, shortTitle := isSupportedNote(title)

	// Extract/Generate slug
	slug := DetermineFileSlug(md.AbsolutePath)
	// Slug is explicitely defined?
	if value, ok := fileAttributes["slug"]; ok {
		if v, ok := value.(string); ok {
			slug = v
		}
	}

	result := &ParsedFileNew{
		Markdown: md,

		// Main Heading
		Slug:       slug,
		Title:      title,
		ShortTitle: shortTitle,

		// File attributes extracted from the Front Matter
		FileAttributes: fileAttributes,
	}

	// Extract objects
	notes, err := result.extractNotes()
	if err != nil {
		return nil, err
	}
	medias, err := result.extractMedias()
	if err != nil {
		return nil, err
	}

	result.Notes = notes
	result.Medias = medias

	return nil, nil
}

func (p *ParsedFileNew) extractNotes() ([]*ParsedNoteNew, error) {
	// All notes collected until now
	var notes []*ParsedNoteNew

	sections, err := p.Markdown.GetSections()
	if err != nil {
		return nil, nil
	}

	for _, section := range sections {

		noteContent := text.ExtractLines(section.ContentText, 1, -1)

		if text.IsBlank(noteContent) {
			// skip sections without text (= category to organize notes, not really free notes)
			continue
		}

		// Determine the attributes
		tags, attributes := ExtractBlockTagsAndAttributes(noteContent)

		// Determine the titles
		title := section.HeadingText
		_, kind, shortTitle := isSupportedNote(title)

		// Determine slug from attribute or define a default one otherwise
		slug := markdown.Slug(p.Slug, string(kind), shortTitle)
		if value, ok := attributes["slug"]; ok {
			if v, ok := value.(string); ok {
				slug = v
			}
		}

		parsedNote := &ParsedNoteNew{
			Level:          section.HeadingLevel,
			Kind:           kind,
			Slug:           slug,
			Title:          title,
			ShortTitle:     shortTitle,
			Line:           section.FileLineStart,
			NoteAttributes: CastAttributes(attributes, GetSchemaAttributeTypes()),
			NoteTags:       tags,
			Body:           noteContent,
		}
		notes = append(notes, parsedNote)
	}

	// Extract objects
	for _, note := range notes {
		note.Flashcard, err = note.extractFlashcard()
		if err != nil {
			return nil, err
		}
		note.Links, err = note.extractLinks()
		if err != nil {
			return nil, err
		}
		note.Reminders, err = note.extractReminders()
		if err != nil {
			return nil, err
		}
	}

	return notes, nil
}

func (p *ParsedFileNew) extractMedias() ([]*ParsedMediaNew, error) {
	// All medias collected until now
	var medias []*ParsedMediaNew

	// Avoid returning duplicates if a media is included twice
	filepaths := make(map[string]bool)

	// Ignore medias inside code blocks (ex: a sample Markdown code block)
	fileBody := markdown.CleanCodeBlocks(p.Markdown.Body)

	regexMedia := regexp.MustCompile(`!\[(.*?)\]\((\S*?)(?:\s+"(.*?)")?\)`)
	matches := regexMedia.FindAllStringSubmatch(fileBody, -1)
	for _, match := range matches {
		txt := match[0]
		line := text.LineNumber(fileBody, txt)

		rawPath := match[2]

		// Check for medias referenced multiple times
		if _, ok := filepaths[rawPath]; ok {
			continue
		}

		// Ex: /some/path/to/markdown.md + ../index.md => /some/path/to/../markdown.md
		absolutePath, err := filepath.Abs(filepath.Join(filepath.Base(p.Markdown.AbsolutePath), rawPath))
		if err != nil {
			return nil, err
		}

		medias = append(medias, &ParsedMediaNew{
			RawPath:      rawPath,
			AbsolutePath: absolutePath,
			Line:         line,
			MediaKind:    DetectMediaKind(rawPath),
			Extension:    filepath.Ext(rawPath),
		})
		filepaths[rawPath] = true // Memorize duplicates
	}

	// Read files on disk after having caught "easy" errors
	for _, media := range medias {
		stat, err := os.Stat(media.AbsolutePath)
		if errors.Is(err, os.ErrNotExist) {
			media.Dangling = true
		} else {
			media.Dangling = false
			media.Size = stat.Size()
			media.MTime = stat.ModTime()
			media.Mode = stat.Mode()
		}
	}

	return medias, nil
}

func (p *ParsedNoteNew) extractFlashcard() (*ParsedFlashcardNew, error) {
	if p.Kind != KindFlashcard {
		return nil, nil
	}

	// Only front/back to parse
	front, back, ok := splitFrontBack(p.Body)
	if !ok {
		return nil, errors.New("missing flashcard separator")
	}

	return &ParsedFlashcardNew{
		ShortTitle: p.ShortTitle,
		Front:      front,
		Back:       back,
	}, nil
}

func (p *ParsedNoteNew) extractLinks() ([]*ParsedLinkNew, error) {
	var links []*ParsedLinkNew

	reLink := regexp.MustCompile(`(?:^|[^!])\[(.*?)\]\("?(http[^\s"]*)"?(?:\s+["'](.*?)["'])?\)`)
	// Note: Markdown images uses the same syntax as links but precedes the link by !
	reTitle := regexp.MustCompile(`(?:(.*)\s+)?#go\/(\S+).*`)

	matches := reLink.FindAllStringSubmatch(p.Body, -1)
	for _, match := range matches {
		text := match[1]
		url := match[2]
		title := match[3]
		submatch := reTitle.FindStringSubmatch(title)
		if submatch == nil {
			continue
		}
		shortTitle := submatch[1]
		goName := submatch[2]

		link := &ParsedLinkNew{
			Text:   text,
			URL:    url,
			Title:  shortTitle,
			GoName: goName,
		}
		links = append(links, link)
	}

	return links, nil
}

func (p *ParsedNoteNew) extractReminders() ([]*ParsedReminderNew, error) {
	var reminders []*ParsedReminderNew

	reReminders := regexp.MustCompile("`(#reminder-(\\S+))`")
	reList := regexp.MustCompile(`^\s*(?:[-+*]|\d+[.])\s+(?:\[.\]\s+)?(.*)\s*$`)

	for _, line := range strings.Split(p.Body, "\n") {
		matches := reReminders.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			tag := match[1]
			_ = match[2] // expression

			description := strings.TrimSpace(p.ShortTitle)

			submatch := reList.FindStringSubmatch(line)
			if submatch != nil {
				// Reminder for a list element
				description = RemoveTagsAndAttributes(submatch[1]) // Remove tags
			}

			reminder := &ParsedReminderNew{
				Description: description,
				Tag:         tag,
			}
			reminders = append(reminders, reminder)
		}
	}

	return reminders, nil
}

// TODO uncomment after refactoring
// func (p *ParsedFileNew) ToFile() (*File, error) {
// 	return NewFileFromParsedFile(p)
// }
