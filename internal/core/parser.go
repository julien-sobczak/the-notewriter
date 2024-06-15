package core

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

type Slug string // TODO see if useful in practice (mainly to build, validate or concatenate)

type Tag string // TODO see if useful in practice (mainly when working with reminder tags)

type GoName string // TODO see if useful in practice (=> no method = useless)

type ParsedFileNew struct {
	Markdown *markdown.File

	// Main Heading
	Slug       string
	Title      markdown.Document
	ShortTitle markdown.Document

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
	Title      markdown.Document
	ShortTitle markdown.Document

	Line           int
	Content        markdown.Document
	Body           markdown.Document
	NoteAttributes map[string]interface{}
	NoteTags       []string

	// Extracted objects
	Flashcard *ParsedFlashcardNew
	Links     []*ParsedLinkNew
	Reminders []*ParsedReminderNew
}

type ParsedFlashcardNew struct {
	// Short title of the note
	ShortTitle markdown.Document

	// Fields in Markdown
	Front markdown.Document
	Back  markdown.Document
}

type ParsedLinkNew struct {
	// The link text
	Text markdown.Document

	// The link destination
	URL string

	// The optional link title
	Title string

	// The optional GO name
	GoName GoName
}

type ParsedReminderNew struct {
	// Description in Markdown of the reminder (ex: the line)
	Description markdown.Document

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
	mtime time.Time
	// Size of the file
	size int64
	// Permission of the file
	mode fs.FileMode

	// Line number where the link present.
	Line int
}

func (p *ParsedNoteNew) GetAttributeAsString(name string) string {
	if v, ok := p.NoteAttributes[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	// Conversion errors are ignored (we consider the requested attribute doesn't exist)
	return ""
}

func (p *ParsedMediaNew) MTime() time.Time {
	return p.mtime
}
func (p *ParsedMediaNew) Size() int64 {
	return p.size
}
func (p *ParsedMediaNew) Mode() fs.FileMode {
	return p.mode
}

func ParseFileFromMarkdownFile(md *markdown.File) (*ParsedFileNew, error) {
	// Extract file attributes
	frontMatter, err := md.FrontMatter.AsMap()
	if err != nil {
		return nil, err
	}
	// fileAttributes := CastAttributes(frontMatter, GetSchemaAttributeTypes()) // FIXME move in file
	fileAttributes := frontMatter

	// Check if file must be ignored
	if value, ok := fileAttributes["tags"]; ok {
		if v, ok := value.(string); ok && v == "ignore" {
			return nil, nil
		}
		if v, ok := value.([]string); ok && slices.Contains[[]string](v, "ignore") {
			return nil, nil
		}
	}

	// Extract titles
	topSection, err := md.GetTopSection()
	if err != nil {
		return nil, err
	}
	title := markdown.Document("")
	if topSection != nil {
		title = topSection.HeadingText
	}
	_, _, shortTitle := isSupportedNote(string(title)) // TODO change signature to avoid casts

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
		ShortTitle: markdown.Document(shortTitle),

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

	return result, nil
}

func (p *ParsedFileNew) extractNotes() ([]*ParsedNoteNew, error) {
	// All notes collected until now
	var notes []*ParsedNoteNew

	sections, err := p.Markdown.GetSections()
	if err != nil {
		return nil, nil
	}

	for _, section := range sections {

		noteContent := section.ContentText
		noteBody := noteContent.ExtractLines(2, -1) // Trim heading

		if noteBody.IsBlank() {
			// skip sections without text (= category to organize notes, not really free notes)
			continue
		}

		// Determine the attributes
		tags, attributes := ExtractBlockTagsAndAttributes(string(noteBody)) // TODO change signature

		// Determine the titles
		title := section.HeadingText
		supported, kind, shortTitle := isSupportedNote(string(title))

		if !supported {
			// Ex: top-level heading, subsections inside a "Note:" already included in the containing note, ...
			continue
		}

		// Determine slug from attribute or define a default one otherwise
		slug := markdown.Slug(p.Slug, string(kind), shortTitle)
		if value, ok := attributes["slug"]; ok {
			if v, ok := value.(string); ok {
				slug = v
			}
		}

		// Apply post-processing on note body
		postProcessNoteBody, err := noteBody.Transform(
			markdown.StripHTMLComments(),
			markdown.StripMarkdownUnofficialComments(),
			// TODO inject <Media> tags? => wait in File to be able to replace link with custom format "blob:<oid>" instead
			markdown.ReplaceCharacters(markdown.AsciidocCharacterSubstitutions))
		if err != nil {
			return nil, err
		}

		parsedNote := &ParsedNoteNew{
			Level:      section.HeadingLevel,
			Kind:       kind,
			Slug:       slug,
			Title:      title,
			ShortTitle: markdown.Document(shortTitle),
			Line:       section.FileLineStart,
			// NoteAttributes: CastAttributes(attributes, GetSchemaAttributeTypes()), FIXME
			NoteAttributes: attributes,
			NoteTags:       tags,
			Content:        noteContent,
			Body:           postProcessNoteBody.TrimSpace(),
		}

		if parsedNote.Kind == KindGenerator {
			// Generator notes are not saved in database
			// They are parsed, evaluated and the results is injected as if
			// the generated notes had been edited manually.
			generatedNotes, generatedMedias, err := p.GenerateNotes(parsedNote)
			if err != nil {
				return nil, err
			}
			if len(generatedNotes) > 0 {
				notes = append(notes, generatedNotes...)
			}
			if len(generatedMedias) > 0 {
				p.Medias = append(p.Medias, generatedMedias...)
			}
		} else {
			notes = append(notes, parsedNote)
		}
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

func (p *ParsedFileNew) GenerateNotes(generator *ParsedNoteNew) ([]*ParsedNoteNew, []*ParsedMediaNew, error) {
	// Inline or external?
	filename := generator.GetAttributeAsString("file")
	interpreter := generator.GetAttributeAsString("interpreter")

	var cmdArgs []string

	if interpreter != "" {
		// Check binary exists...
		interpreterStat, err := os.Stat(interpreter)
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("interpreter %q doesn't exist in generator %q", interpreter, generator.ShortTitle)
		}
		// ... and is executable
		if !IsExec(interpreterStat.Mode()) {
			return nil, nil, fmt.Errorf("interpreter %q is not executable in generator %q", interpreter, generator.ShortTitle)
		}

		cmdArgs = append(cmdArgs, interpreter)
	}

	if filename != "" { // External
		scriptPath := filepath.Join(filepath.Dir(p.Markdown.AbsolutePath), interpreter)

		// Check file exists
		scriptStat, err := os.Stat(scriptPath)
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("script %q doesn't exist in generator %q", filename, generator.ShortTitle)
		}

		// check file is executable
		if interpreter == "" && !IsExec(scriptStat.Mode()) {
			return nil, nil, fmt.Errorf("script %q is not executable in generator %q", filename, generator.ShortTitle)
		}

		cmdArgs = append(cmdArgs, scriptPath)
	} else { // Internal

		// Search for the first code block in note
		codeBlocks := generator.Body.ExtractCodeBlocks()
		if len(codeBlocks) == 0 {
			return nil, nil, fmt.Errorf("missing 'file' attribute or code block inside generator %q", p.ShortTitle)
		}

		script := codeBlocks[0]

		scriptLanguage := script.Language
		scriptContent := script.Source

		if scriptLanguage == "" {
			return nil, nil, fmt.Errorf("missing language in code block inside generator %q", p.ShortTitle)
		}

		// Expect the Markdown language
		cmdArgs = append(cmdArgs, scriptLanguage)

		scriptPath, err := os.CreateTemp("nt", "script")
		if err != nil {
			return nil, nil, fmt.Errorf("unable to create temporary script for generator %q: %w", p.ShortTitle, err)
		}
		defer os.Remove(scriptPath.Name())
		os.WriteFile(scriptPath.Name(), []byte(scriptContent), 0755)

		cmdArgs = append(cmdArgs, scriptPath.Name())
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", stderr.Bytes())
		return nil, nil, fmt.Errorf("failed to run generator command %q: %w", strings.Join(cmdArgs, " "), err)
	}

	mdPath, err := os.CreateTemp("nt", "md")
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create temporary Markdown file for generator %q: %w", p.ShortTitle, err)
	}
	defer os.Remove(mdPath.Name())
	if err := os.WriteFile(mdPath.Name(), stdout.Bytes(), 0644); err != nil {
		return nil, nil, fmt.Errorf("unable to write temporary Markdown file for generator %q: %w", p.ShortTitle, err)
	}

	mdFile, err := markdown.ParseFile(mdPath.Name())
	if err != nil {
		return nil, nil, err
	}
	generatedFile, err := ParseFileFromMarkdownFile(mdFile)
	if err != nil {
		return nil, nil, err
	}

	var resultsNotes []*ParsedNoteNew
	var resultsMedias []*ParsedMediaNew
	// Use original line number to make easy to jump to the generator note
	for _, generatedNote := range generatedFile.Notes {
		generatedNote.Line = generator.Line
	}
	for _, generatedMedia := range generatedFile.Medias {
		generatedMedia.Line = generator.Line
	}

	return resultsNotes, resultsMedias, nil
}

func (p *ParsedFileNew) FileLineNumber(bodyLineNumber int) int {
	return p.Markdown.BodyLine + bodyLineNumber - 1
}

func (p *ParsedFileNew) extractMedias() ([]*ParsedMediaNew, error) {
	// All medias collected until now
	var medias []*ParsedMediaNew

	// Avoid returning duplicates if a media is included twice
	filepaths := make(map[string]bool)

	// Ignore medias inside code blocks (ex: a sample Markdown code block)
	fileBody := p.Markdown.Body.MustTransform(markdown.StripCodeBlocks())

	regexMedia := regexp.MustCompile(`!\[(.*?)\]\((\S*?)(?:\s+"(.*?)")?\)`)
	matches := regexMedia.FindAllStringSubmatch(string(fileBody), -1)
	for _, match := range matches {
		txt := match[0]
		line := text.LineNumber(string(fileBody), txt)

		rawPath := match[2]

		// Check for medias referenced multiple times
		if _, ok := filepaths[rawPath]; ok {
			continue
		}

		// Ex: /some/path/to/markdown.md + ../index.md => /some/path/to/../markdown.md
		absolutePath, err := filepath.Abs(filepath.Join(filepath.Dir(p.Markdown.AbsolutePath), rawPath))
		if err != nil {
			return nil, err
		}

		medias = append(medias, &ParsedMediaNew{
			RawPath:      rawPath,
			AbsolutePath: absolutePath,
			Line:         p.FileLineNumber(line),
			MediaKind:    DetectMediaKind(rawPath),
			Extension:    filepath.Ext(rawPath),
		})
		filepaths[rawPath] = true // Memorize duplicates
	}

	// Read files on disk after having caught "easy" errors
	for _, media := range medias {
		stat, err := os.Stat(media.AbsolutePath)
		fmt.Println("Testing TOTO", media.AbsolutePath)
		if errors.Is(err, os.ErrNotExist) {
			media.Dangling = true
		} else {
			media.Dangling = false
			media.size = stat.Size()
			media.mtime = stat.ModTime()
			media.mode = stat.Mode()
		}
	}

	return medias, nil
}

func (p *ParsedNoteNew) extractFlashcard() (*ParsedFlashcardNew, error) {
	if p.Kind != KindFlashcard {
		return nil, nil
	}

	// Only front/back to parse
	parts := p.Body.SplitByHorizontalRules()
	if len(parts) < 2 {
		return nil, errors.New("missing flashcard separator")
	}
	if len(parts) > 2 {
		return nil, errors.New("too many flashcard separator")
	}
	front := parts[0]
	back := parts[1]

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

	matches := reLink.FindAllStringSubmatch(string(p.Body), -1)
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
			Text:   markdown.Document(text),
			URL:    url,
			Title:  shortTitle,
			GoName: GoName(goName),
		}
		links = append(links, link)
	}

	return links, nil
}

func (p *ParsedNoteNew) extractReminders() ([]*ParsedReminderNew, error) {
	var reminders []*ParsedReminderNew

	reReminders := regexp.MustCompile("`(#reminder-(\\S+))`")
	reList := regexp.MustCompile(`^\s*(?:[-+*]|\d+[.])\s+(?:\[.\]\s+)?(.*)\s*$`)

	lines := p.Body.Lines()
	for _, line := range lines {
		matches := reReminders.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			tag := match[1]
			_ = match[2] // expression

			description := p.ShortTitle.TrimSpace()

			submatch := reList.FindStringSubmatch(line)
			if submatch != nil {
				// Reminder for a list element
				descriptionText := markdown.Document(submatch[1])
				descriptionCleaned, err := descriptionText.Transform(StripTagsAndAttributes())
				if err != nil {
					return nil, err
				}
				description = descriptionCleaned
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

// FindMediaByFilename searches for a media based on the filename.
// The code uses `strings.HasSuffix` and therefore, (partial) paths can be passed too.
func (f *ParsedFileNew) FindMediaByFilename(filename string) (*ParsedMediaNew, bool) {
	for _, media := range f.Medias {
		if strings.HasSuffix(media.AbsolutePath, filename) {
			return media, true
		}
	}
	return nil, false
}

// FindNoteByShortTitle searches for a note based on its short title.
// The code does a strict comparison and the exact short title must be passed.
func (f *ParsedFileNew) FindNoteByShortTitle(shortTitle string) (*ParsedNoteNew, bool) {
	for _, note := range f.Notes {
		if note.ShortTitle == markdown.Document(shortTitle) {
			return note, true
		}
	}
	return nil, false
}

// TODO uncomment after refactoring
// func (p *ParsedFileNew) ToFile() (*File, error) {
// 	return NewFileFromParsedFile(p)
// }

// StripTagsAndAttributes removes all tags and attributes from a NoteWriter note.
func StripTagsAndAttributes() markdown.Transformer {
	return func(doc markdown.Document) (markdown.Document, error) {
		var res bytes.Buffer
		for _, line := range doc.Lines() {
			newLine := regexTags.ReplaceAllLiteralString(line, "")
			newLine = regexAttributes.ReplaceAllLiteralString(newLine, "")
			if !text.IsBlank(newLine) {
				res.WriteString(newLine + "\n")
			}
		}
		return markdown.Document(text.SquashBlankLines(res.String())).TrimSpace(), nil
	}
}
