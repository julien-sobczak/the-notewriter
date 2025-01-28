package core

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/helpers"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

type Slug string // TODO see if useful in practice (mainly to build, validate or concatenate)

type Tag string // TODO see if useful in practice (mainly when working with reminder tags)

type ParsedFile struct {
	Markdown *markdown.File

	RepositoryPath string // FIXME remove
	AbsolutePath   string // FIXME remove
	// The relative path inside the repository
	RelativePath string

	// Main Heading
	Slug       string
	Title      markdown.Document
	ShortTitle markdown.Document

	// File attributes extracted from the Front Matter
	FileAttributes AttributeSet

	// Extracted objects
	Notes     []*ParsedNote
	Medias    []*ParsedMedia
	Wikilinks []markdown.Wikilink // TODO still useful now that the method is exposed on markdown.Document?
}

// ParsedNote represents a single raw note inside a file.
type ParsedNote struct {
	Parent *ParsedNote

	Level int
	Kind  NoteKind

	// The absolute path of the file
	AbsolutePath string
	// The relative path inside the repository
	RelativePath string

	// Heading
	Slug       string
	Title      markdown.Document
	ShortTitle markdown.Document

	Line       int
	Content    markdown.Document
	Body       markdown.Document
	Comment    markdown.Document
	Attributes AttributeSet

	// Extracted objects
	Flashcard *ParsedFlashcard
	GoLinks   []*ParsedGoLink
	Reminders []*ParsedReminder
}

type ParsedFlashcard struct {
	// Short title of the note
	ShortTitle markdown.Document

	// Slug of the note
	Slug string

	// Fields in Markdown
	Front markdown.Document
	Back  markdown.Document
}

type ParsedGoLink struct {
	// The link text
	Text markdown.Document

	// The link destination
	URL string

	// The optional link title
	Title string

	// The optional GO name
	GoName string
}

type ParsedReminder struct {
	// Description in Markdown of the reminder (ex: the line)
	Description markdown.Document

	// Tag value containig the formula to determine the next occurence
	Tag string `yaml:"tag"`
}

type ParsedMedia struct {
	// The path as specified in the file. (Ex: "../medias/pic.png")
	RawPath string
	// The absolute path
	AbsolutePath string
	// The relative path inside the repository
	RelativePath string
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

	// Line number where the link present.
	Line int
}

func (p *ParsedMedia) FileMTime() time.Time {
	return p.MTime
}
func (p *ParsedMedia) FileSize() int64 {
	return p.Size
}
func (p *ParsedMedia) FileHash() string {
	// Implementation: We do not store the hash to avoid calculating
	// the hash if not needed as medias can be large.
	hash, _ := helpers.HashFromFile(p.AbsolutePath)
	// TODO handle error
	return hash
}

// FIXME delete this method => Move as test helper instead
func ParseFileFromRelativePath(repositoryAbsolutePath, fileRelativePath string) (*ParsedFile, error) {
	fileAbsolutePath := filepath.Join(repositoryAbsolutePath, fileRelativePath)
	markdownFile, err := markdown.ParseFile(fileAbsolutePath)
	if err != nil {
		return nil, err
	}
	return ParseFile(repositoryAbsolutePath, markdownFile, nil)
}

func ParseFile(repositoryAbsolutePath string, md *markdown.File, mdParent *markdown.File) (*ParsedFile, error) {
	if mdParent == nil {
		mdParent = markdown.EmptyFile
	}

	// Extract attributes
	parentFrontMatter, err := mdParent.FrontMatter.AsMap()
	if err != nil {
		return nil, err
	}
	fileFrontMatter, err := md.FrontMatter.AsMap()
	if err != nil {
		return nil, err
	}
	parentAttributes := AttributeSet(parentFrontMatter).CastOrIgnore(GetSchemaAttributeTypes())
	fileAttributes := parentAttributes.Merge(AttributeSet(fileFrontMatter))

	// Check if file must be ignored
	if fileAttributes.Tags().Includes("ignore") {
		return nil, nil
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
	relativePath := RelativePath(repositoryAbsolutePath, md.AbsolutePath)
	slug := DetermineFileSlug(relativePath)
	// Slug is explicitely defined?
	if value, ok := fileAttributes["slug"]; ok {
		if v, ok := value.(string); ok {
			slug = v
		}
	}

	result := &ParsedFile{
		Markdown: md,

		RepositoryPath: repositoryAbsolutePath,
		AbsolutePath:   md.AbsolutePath,
		RelativePath:   RelativePath(repositoryAbsolutePath, md.AbsolutePath),

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
	wikilinks := result.extractWikilinks()

	result.Notes = notes
	result.Medias = medias
	result.Wikilinks = wikilinks

	return result, nil
}

func (p *ParsedFile) extractNotes() ([]*ParsedNote, error) {

	// All notes collected until now
	var notes []*ParsedNote

	sections, err := p.Markdown.GetSections()
	if err != nil {
		return nil, nil
	}

	for _, section := range sections {

		// Trim content to remove sub-notes (= typed notes)
		noteContent := section.ContentText.MustTransform(StripSubNotesTransformer)

		noteBody := noteContent.ExtractLines(2, -1) // Trim heading

		if noteBody.IsBlank() {
			// skip sections without text (= category to organize notes, not really free notes)
			continue
		}

		// Determine the attributes
		noteTags, noteAttributes := ExtractBlockTagsAndAttributes(noteBody, GetSchemaAttributeTypes())

		// Determine the titles
		title := section.HeadingText
		supported, kind, shortTitle := isSupportedNote(string(title))

		if !supported {
			// Ex: top-level heading, subsections inside a "Note:" already included in the containing note, ...
			continue
		}

		// Ignore ignorabled notes
		if noteTags.Includes("ignore") {
			continue
		}

		// Determine slug from attribute or define a default one otherwise
		slug := markdown.Slug(p.Slug, string(kind), shortTitle)
		if attributeSlug, ok := noteAttributes.Slug(); ok {
			slug = attributeSlug
		}

		// Apply post-processing on note body
		postProcessedNoteBody, err := noteBody.Transform(
			markdown.StripHTMLComments(),
			markdown.StripMarkdownUnofficialComments(),
			// TODO inject <Media> tags? => wait in File to be able to replace link with custom format "blob:<oid>" instead
			markdown.ReplaceCharacters(markdown.AsciidocCharacterSubstitutions))
		if err != nil {
			return nil, err
		}
		// TODO convert quotes

		// Find a possible parent note
		i := len(notes) - 1
		var previousNote *ParsedNote
		var parentNote *ParsedNote
		for i > 0 {
			previousNote = notes[i]
			if previousNote.Level < section.HeadingLevel {
				parentNote = previousNote
				break
			}
			i--
		}

		body, comment := postProcessedNoteBody.ExtractComment()

		attributes := FilterNonInheritableAttributes(p.FileAttributes, p.RelativePath, kind)
		if parentNote != nil {
			parentAttributes := FilterNonInheritableAttributes(parentNote.Attributes, p.RelativePath, kind)
			attributes = attributes.Merge(parentAttributes)
		}
		attributes = attributes.Merge(noteAttributes)

		parsedNote := &ParsedNote{
			Parent:       parentNote,
			Level:        section.HeadingLevel,
			Kind:         kind,
			AbsolutePath: p.AbsolutePath,
			RelativePath: p.RelativePath,
			Slug:         slug,
			Title:        title,
			ShortTitle:   markdown.Document(shortTitle),
			Line:         section.FileLineStart,
			Attributes:   attributes,
			Content:      noteContent,
			Body:         body,
			Comment:      comment,
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
		note.GoLinks, err = note.extractGoLinks()
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

func (p *ParsedFile) GenerateNotes(generator *ParsedNote) ([]*ParsedNote, []*ParsedMedia, error) {
	// Inline or external?
	filename := generator.Attributes.CastValueAsString("file")
	interpreter := generator.Attributes.CastValueAsString("interpreter")

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
	generatedFile, err := ParseFile(p.RepositoryPath, mdFile, nil)
	if err != nil {
		return nil, nil, err
	}

	var resultsNotes []*ParsedNote
	var resultsMedias []*ParsedMedia
	// Use original line number to make easy to jump to the generator note
	for _, generatedNote := range generatedFile.Notes {
		generatedNote.Line = generator.Line
	}
	for _, generatedMedia := range generatedFile.Medias {
		generatedMedia.Line = generator.Line
	}

	return resultsNotes, resultsMedias, nil
}

func (p *ParsedFile) extractWikilinks() []markdown.Wikilink {
	return p.Markdown.Body.Wikilinks()
}

// Hash returns a hash based on the full file content.
func (p *ParsedFile) Hash() string {
	return helpers.Hash([]byte(p.Markdown.Content))
}

// Filename returns the filename of the Markdown file.
func (p *ParsedFile) Filename() string {
	return filepath.Base(p.AbsolutePath)
}

// AbsoluteDir returns the dirname of the Markdown file.
func (p *ParsedFile) AbsoluteDir() string {
	return filepath.Dir(p.AbsolutePath)
}

// RelativeDir returns the dirname of the Markdown file.
func (p *ParsedFile) RelativeDir() string {
	return filepath.Dir(p.RelativePath)
}

func (p *ParsedFile) FileLineNumber(bodyLineNumber int) int {
	return p.Markdown.BodyLine + bodyLineNumber - 1
}

func (p *ParsedFile) extractMedias() ([]*ParsedMedia, error) {
	// All medias collected until now
	var medias []*ParsedMedia

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
		absolutePath, err := filepath.Abs(filepath.Join(filepath.Dir(p.AbsolutePath), rawPath))
		if err != nil {
			return nil, err
		}

		medias = append(medias, &ParsedMedia{
			RawPath:      rawPath,
			AbsolutePath: absolutePath,
			RelativePath: RelativePath(p.RepositoryPath, absolutePath),
			Line:         p.FileLineNumber(line),
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
		}
	}

	return medias, nil
}

func (p *ParsedNote) extractFlashcard() (*ParsedFlashcard, error) {
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

	return &ParsedFlashcard{
		ShortTitle: p.ShortTitle,
		Slug:       p.Slug,
		Front:      front,
		Back:       back,
	}, nil
}

func (p *ParsedNote) extractGoLinks() ([]*ParsedGoLink, error) {
	var links []*ParsedGoLink

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

		link := &ParsedGoLink{
			Text:   markdown.Document(text),
			URL:    url,
			Title:  shortTitle,
			GoName: goName,
		}
		links = append(links, link)
	}

	return links, nil
}

func (p *ParsedNote) extractReminders() ([]*ParsedReminder, error) {
	var reminders []*ParsedReminder

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

			reminder := &ParsedReminder{
				Description: description,
				Tag:         tag,
			}
			reminders = append(reminders, reminder)
		}
	}

	return reminders, nil
}

// Hash returns a hash based on the Markdown content.
func (p *ParsedNote) Hash() string {
	return p.Content.Hash()
}

// FindMediaByFilename searches for a media based on the filename.
// The code uses `strings.HasSuffix` and therefore, (partial) paths can be passed too.
func (f *ParsedFile) FindMediaByFilename(filename string) (*ParsedMedia, bool) {
	for _, media := range f.Medias {
		if strings.HasSuffix(media.AbsolutePath, filename) {
			return media, true
		}
	}
	return nil, false
}

// FindNoteByTitle searches for a note based on its title.
// The code does a strict comparison and the exact title must be passed (without the leading '#' characters).
func (f *ParsedFile) FindNoteByTitle(title string) (*ParsedNote, bool) {
	for _, note := range f.Notes {
		if note.Title == markdown.Document(title) {
			return note, true
		}
	}
	return nil, false
}

// FindNoteByShortTitle searches for a note based on its short title.
// The code does a strict comparison and the exact short title must be passed.
func (f *ParsedFile) FindNoteByShortTitle(shortTitle string) (*ParsedNote, bool) {
	for _, note := range f.Notes {
		if note.ShortTitle == markdown.Document(shortTitle) {
			return note, true
		}
	}
	return nil, false
}

// FindGoLinkByGoName searches for a go link from its go name.
func (p *ParsedNote) FindGoLinkByGoName(name string) (*ParsedGoLink, bool) {
	for _, goLink := range p.GoLinks {
		if goLink.GoName == name {
			return goLink, true
		}
	}
	return nil, false
}

// FindReminderByTag searches for a go link from its go name.
func (p *ParsedNote) FindReminderByTag(tag string) (*ParsedReminder, bool) {
	for _, reminder := range p.Reminders {
		if reminder.Tag == tag {
			return reminder, true
		}
	}
	return nil, false
}

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

// DetermineFileSlug generates a slug from a file path.
func DetermineFileSlug(relativePath string) string {
	var slugsParts []any

	// Include the dirname
	dirname := filepath.Base(filepath.Dir(relativePath))
	if dirname != "" {
		// Do not prefix by the dirname when file are present at the root
		slugsParts = append(slugsParts, dirname)
	}

	// Include the filename (without the extension) except for index.md (as no additional meaning)
	// and except when the file is named after the directory.
	filenameWithoutExtension := text.TrimExtension(filepath.Base(relativePath))
	if filenameWithoutExtension != "index" && filenameWithoutExtension != dirname {
		slugsParts = append(slugsParts, filenameWithoutExtension)
	}

	return markdown.Slug(slugsParts...)
}

// RelativePath returns the relative from a given file.
// Ex:
//
//	absolutePath = /home/julien/repository/dir/note.md
//	rootPath     = /home/julien/repository/
//	relativePath =                         dir/note.md
func RelativePath(rootPath, absolutePath string) string {
	relativePath, err := filepath.Rel(rootPath, absolutePath)
	if err != nil {
		// Must not happen (fail abruptly)
		log.Fatalf("Unable to determine relative path for %q from root %q: %v", absolutePath, rootPath, err)
	}
	return relativePath
}

// StripSubNotesTransformer removes sub-notes from a document
func StripSubNotesTransformer(document markdown.Document) (markdown.Document, error) {
	// The current implementation traverses the lines until finding the first sub-note
	it := document.Iterator()

	// Skip top note heading
	for it.HasNext() {
		line := it.Next()
		ok, _, _ := markdown.IsHeading(line.Text)
		if ok {
			break
		}
	}

	// Move to next note-specific heading
	for it.HasNext() {
		line := it.Next()
		ok, headingText, _ := markdown.IsHeading(line.Text)
		if ok {
			supported, _, _ := isSupportedNote(headingText)
			if supported {
				// Found the first sub-note
				return document.ExtractLines(0, line.Number-1).TrimSpace(), nil
			}
		}
	}

	// No sub-note found, simply returns the original document
	return document, nil
}
