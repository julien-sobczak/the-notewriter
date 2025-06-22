package core

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/julien-sobczak/the-notewriter/internal/helpers"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

type Slug string // TODO see if useful in practice (mainly to build, validate or concatenate)

type Tag string // TODO see if useful in practice (mainly when working with reminder tags)

// Preprocessor processes a parsed note and returns a new list of parsed notes.
type Preprocessor func(*ParsedFile, *ParsedNote) ([]*ParsedNote, error)

// Preprocessors is a map of preprocessor functions.
var preprocessors = make(map[string]Preprocessor)

// RegisterPreprocessor registers the preprocessor matching a given name.
func RegisterPreprocessor(name string, preprocessor Preprocessor) {
	if _, ok := preprocessors[name]; ok {
		panic(fmt.Sprintf("Preprocessor %q already registered", name))
	}
	preprocessors[name] = preprocessor
}

type ParsedFile struct {
	Markdown *markdown.File

	// The root repository path
	RepositoryPath string
	// The absolute path of the file
	AbsolutePath string
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
	Type  string

	// The absolute path of the file
	AbsolutePath string
	// The relative path inside the repository
	RelativePath string

	// Heading
	Slug       string
	Title      markdown.Document
	ShortTitle markdown.Document
	LongTitle  markdown.Document

	Line           int
	Content        markdown.Document
	Body           markdown.Document
	Comment        markdown.Document
	NoteTags       TagSet       // Tags defined explicitely in the note
	NoteAttributes AttributeSet // Attributes defined explicitely in the note
	Attributes     AttributeSet // Attributes defined in the note and inherited from the parent notes/files

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

func MustParseOrphanFile(md *markdown.File) *ParsedFile {
	parsedFile, err := ParseOrphanFile(md)
	if err != nil {
		panic(err)
	}
	return parsedFile
}

func ParseOrphanFile(md *markdown.File) (*ParsedFile, error) {
	return ParseFile(md, markdown.EmptyFile)
}

func ParseFile(md *markdown.File, mdParent *markdown.File) (*ParsedFile, error) {
	if mdParent == nil {
		mdParent = markdown.EmptyFile
	}

	// Extract attributes
	parentAttributes, err := NewAttributeSetFromMarkdown(mdParent)
	if err != nil {
		return nil, err
	}
	parentAttributes = parentAttributes.CastOrIgnore(CurrentConfigFile().Attributes)
	fileAttributes, err := NewAttributeSetFromMarkdown(md)
	if err != nil {
		return nil, err
	}
	fileAttributes = fileAttributes.CastOrIgnore(CurrentConfigFile().Attributes)
	fileAttributes = parentAttributes.Merge(fileAttributes)

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
	_, shortTitle, _ := CurrentConfigFile().IsSupportedType(string(title))

	// Extract/Generate slug
	relativePath := CurrentRepository().GetFileRelativePath(md.AbsolutePath)
	slug := DetermineFileSlug(relativePath)
	// Slug is explicitely defined?
	if value, ok := fileAttributes["slug"]; ok {
		if v, ok := value.(string); ok {
			slug = v
		}
	}

	result := &ParsedFile{
		Markdown: md,

		RepositoryPath: CurrentConfig().RootDirectory,
		AbsolutePath:   md.AbsolutePath,
		RelativePath:   relativePath,

		// Main Heading
		Slug:       slug,
		Title:      title,
		ShortTitle: shortTitle,

		// File attributes extracted from the Front Matter
		FileAttributes: fileAttributes,
	}

	// Extract objects
	medias, err := result.extractMedias() // Start with medias as they are used in notes
	if err != nil {
		return nil, err
	}
	result.Medias = medias
	notes, err := result.extractNotes()
	if err != nil {
		return nil, err
	}
	result.Notes = notes
	result.Wikilinks = result.extractWikilinks()

	return result, nil
}

func (p *ParsedFile) extractNotes() ([]*ParsedNote, error) {

	// All notes collected until now
	var notes []*ParsedNote
	var noteSections []*markdown.Section // Sections matching the notes

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
		noteTags, noteAttributes := ExtractBlockTagsAndAttributes(noteBody)

		// Determine the titles
		title := section.HeadingText
		noteType, shortTitle, supported := CurrentConfigFile().IsSupportedType(string(title))

		if !supported {
			// Ex: top-level heading, subsections inside a "Note:" already included in the containing note, ...
			continue
		}

		// Ignore ignorabled notes
		if noteTags.Includes("ignore") {
			continue
		}

		// Determine slug from attribute or define a default one otherwise
		slug := markdown.Slug(p.Slug, noteType.Name, shortTitle)
		if attributeSlug, ok := noteAttributes.Slug(); ok {
			slug = attributeSlug
		}

		// Apply post-processing on note body
		postProcessedNoteBody, err := noteBody.Transform(
			StripBlockTagsAndAttributes(),
			markdown.StripHTMLComments(),
			markdown.StripMarkdownUnofficialComments(),
			markdown.AlignHeadings(),
			ReplaceMedias(p.Medias),
			markdown.ReplaceCharacters(markdown.AsciidocCharacterSubstitutions))
		if err != nil {
			return nil, err
		}

		// Find a possible parent note
		i := len(notes) - 1
		var previousNote *ParsedNote
		var parentNote *ParsedNote
		for i >= 0 {
			previousNote = notes[i]
			previousSection := noteSections[i]
			if previousNote.Level < section.HeadingLevel && previousSection.Includes(*section) {
				// A previous note can have a higher level but there may exist a Markdown heading between them.
				// Ex:
				//      # Note: A
				//      # Parent
				//      ## Note: B
				// "B" has no parent note. The section "Parent" must be used for the long name instead.
				parentNote = previousNote
				break
			}
			i--
		}
		// Find possible parent sections
		var parentTitles []markdown.Document
		for _, otherSection := range sections {
			if otherSection.Includes(*section) {
				sectionTitle := otherSection.HeadingText.String()
				if _, shortTitle, supported := CurrentConfigFile().IsSupportedType(sectionTitle); supported {
					// Ex: "## Note: Parent Note"
					parentTitles = append(parentTitles, shortTitle)
				} else {
					// Ex: "## Not a note"
					parentTitles = append(parentTitles, otherSection.HeadingText)
				}
			}
		}

		body, comment := postProcessedNoteBody.ExtractComment()

		// Determine attributes
		attributes := FilterNonInheritableAttributes(p.FileAttributes)
		if parentNote != nil {
			parentAttributes := FilterNonInheritableAttributes(parentNote.Attributes)
			attributes = attributes.Merge(parentAttributes)
		}
		attributes = attributes.Merge(noteAttributes)
		// Append hooks defined on the note type
		if noteType.Hooks != nil {
			attributes.AddHook(noteType.Hooks...)
		}

		// Determine the long title
		var titles []markdown.Document
		if parentTitles != nil {
			titles = append(titles, parentTitles...)
		} else if p.ShortTitle != "" {
			titles = append(titles, p.ShortTitle)
		}
		titles = append(titles, shortTitle)
		longTitle := FormatLongTitle(titles...)

		parsedNote := &ParsedNote{
			Parent:         parentNote,
			Level:          section.HeadingLevel,
			Type:           noteType.Name,
			AbsolutePath:   p.AbsolutePath,
			RelativePath:   p.RelativePath,
			Slug:           slug,
			Title:          title,
			ShortTitle:     shortTitle,
			LongTitle:      longTitle,
			Line:           section.FileLineStart,
			NoteTags:       noteTags,
			NoteAttributes: noteAttributes,
			Attributes:     attributes,
			Content:        noteContent,
			Body:           body,
			Comment:        comment,
		}

		parsedNotes := []*ParsedNote{parsedNote}

		// Apply successive preprocessors on the note.
		// As a processor can generate several notes, we need to loop
		// until all processors have been applied.
		for _, preprocessorName := range noteType.Preprocessors {
			preprocessor, ok := preprocessors[preprocessorName]
			if !ok {
				return nil, fmt.Errorf("unknown preprocessor %q", preprocessorName)
			}

			// The notes generated by the current preprocessor
			newParsedNotes := []*ParsedNote{}

			for _, parsedNote := range parsedNotes {
				generatedNotes, err := preprocessor(p, parsedNote)
				if err != nil {
					return nil, fmt.Errorf("failing preprocessor %q on note %q (%s:%d): %v", preprocessorName, parsedNote.ShortTitle, parsedNote.RelativePath, parsedNote.Line, err)
				}
				if len(generatedNotes) > 0 {
					newParsedNotes = append(newParsedNotes, generatedNotes...)
				}
			}

			parsedNotes = newParsedNotes
		}

		notes = append(notes, parsedNote)
		noteSections = append(noteSections, section)
	}

	// Extract objects
	for _, note := range notes {
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

// ReplaceMedias replaces the media links in a Markdown document by <media> tags easier to work with.
func ReplaceMedias(medias []*ParsedMedia) markdown.Transformer {
	return func(doc markdown.Document) (markdown.Document, error) {
		rawDoc := doc.String()
		// Replace medias by <media> tags
		for _, media := range medias {
			if media.Dangling {
				continue // Do not replace dangling medias
			}
			// Replace the media link by a <media> tag
			escapedRawPath := regexp.QuoteMeta(media.RawPath)
			pattern := `!\[(.*?)\]\(` + escapedRawPath + `\s*(?:"(.*?)")?.*?\)`
			re := regexp.MustCompile(pattern)
			rawDoc = re.ReplaceAllStringFunc(rawDoc, func(match string) string {
				groups := re.FindStringSubmatch(match)
				if len(groups) < 3 {
					// Not enough groups, return the original match
					return match
				}
				alt := groups[1]
				relativePath := media.RelativePath
				title := groups[2]
				// Extract the media link and replace it by a <media> tag
				// Ex: ![image](../medias/pic.png)
				// becomes <media relative-path="../medias/pic.png" />
				var sb strings.Builder
				sb.WriteString("<media ")
				if relativePath != "" {
					sb.WriteString(fmt.Sprintf("relative-path=\"%s\" ", relativePath))
				}
				if alt != "" {
					sb.WriteString(fmt.Sprintf("alt=\"%s\" ", alt))
				}
				if title != "" {
					sb.WriteString(fmt.Sprintf("title=\"%s\" ", title))
				}
				sb.WriteString("/>")
				return sb.String()
			})
		}
		return markdown.Document(rawDoc), nil
	}
}

// FilterNonInheritableAttributes filters the attributes to keep only the inheritable ones
func FilterNonInheritableAttributes(attributeSet AttributeSet) AttributeSet {
	// Filter the attributes to keep only the inheritable ones
	// (ex: tags are not inheritable)
	filtered := make(AttributeSet)
	for key, value := range attributeSet {
		attributeConfig, ok := CurrentConfig().ConfigFile.GetAttribute(key)
		if !ok || *attributeConfig.Inherit {
			// Undefined attribute are inherited by default
			filtered[key] = value
		}
	}
	return filtered
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

	for _, mdMedia := range p.Markdown.Body.ExtractImages() {
		if mdMedia.External() {
			// External medias aren't processed and we simply rendered using the URL
			continue
		}
		mdMediaAbsolute, err := mdMedia.Transform(markdown.ResolveAbsoluteURL(p.AbsolutePath))
		if err != nil {
			return nil, err
		}

		// Check for medias referenced multiple times
		if _, ok := filepaths[mdMediaAbsolute.URL]; ok {
			continue
		}

		newMedia := ParseMedia(p.RepositoryPath, mdMediaAbsolute.URL)
		newMedia.RawPath = mdMedia.URL
		newMedia.Line = mdMedia.Line

		medias = append(medias, newMedia)
		filepaths[mdMediaAbsolute.URL] = true // Memorize duplicates
	}

	return medias, nil
}

// ParseMedia parses a media file from its path.
func ParseMedia(repositoryPath, absolutePath string) *ParsedMedia {
	parsedMedia := &ParsedMedia{
		AbsolutePath: absolutePath,
		RelativePath: RelativePath(repositoryPath, absolutePath),
		MediaKind:    DetectMediaKind(absolutePath),
		Extension:    filepath.Ext(absolutePath),
	}
	stat, err := filesystem.Stat(parsedMedia.AbsolutePath)
	if errors.Is(err, os.ErrNotExist) {
		parsedMedia.Dangling = true
	} else {
		parsedMedia.Dangling = false
		parsedMedia.Size = stat.Size()
		parsedMedia.MTime = stat.ModTime()
	}
	return parsedMedia
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

	insideCodeBlock := false
	// We ignore headings inside code blocks

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

		if markdown.IsCodeBlock(line.Text) {
			insideCodeBlock = !insideCodeBlock
			continue
		}
		if insideCodeBlock {
			continue
		}

		ok, headingText, _ := markdown.IsHeading(line.Text)
		if ok {
			_, _, supported := CurrentConfigFile().IsSupportedType(headingText)
			if supported {
				// Found the first sub-note
				return document.ExtractLines(0, line.Number-1).TrimSpace(), nil
			}
		}
	}

	// No sub-note found, simply returns the original document
	return document, nil
}

// FormatLongTitle formats the long title of a note.
func FormatLongTitle(titles ...markdown.Document) markdown.Document {

	// Some titles may already have been formated (ex: "Go / Goroutines")
	// Explode everything and start from scratch.
	var explodedTitles []string
	for _, title := range titles {
		explodedTitles = append(explodedTitles, strings.Split(string(title), NoteLongTitleSeparator)...)
	}

	// Implementation: We concatenate the titles but we must avoid duplication.
	//
	// Ex:
	//     # Subject
	//     ## Note: Technique A
	//     ### Flashcard: Technique A
	//
	// The long title must be "Subject / Technique A", not "Subject / Technique A / Technique A".
	//
	// Ex:
	//     # Go
	//     ## Note: Goroutines
	//     ## Note: Go History
	//
	// The long titles must be "Go / Goroutines" & "Go History".

	prevTitle := ""
	longTitle := ""

	for i := len(explodedTitles) - 1; i >= 0; i-- {
		title := explodedTitles[i]

		if text.IsBlank(title) { // Empty
			continue
		}

		if prevTitle == title { // Duplicate
			continue
		}

		if strings.HasPrefix(longTitle, title) { // Common prefix
			// Beware "false" common prefixes. Ex: "Go" and "Goroutines" must result in "Go / Goroutines"
			nextCharacter, _ := utf8.DecodeRuneInString(strings.TrimPrefix(longTitle, title))
			if !syntax.IsWordChar(nextCharacter) {
				continue
			}
		}

		if longTitle == "" {
			longTitle = title
		} else {
			longTitle = title + NoteLongTitleSeparator + longTitle
		}
		prevTitle = title
	}

	return markdown.Document(longTitle)
}

// Matches checks if the note matches the given query.
func (n *ParsedNote) Matches(query *Query) bool {
	if query == nil {
		// No query, no filter
		return true
	}

	if query.Slug != "" {
		if !strings.EqualFold(n.Slug, query.Slug) {
			return false
		}
	}
	if query.Path != "" {
		if !strings.HasPrefix(n.RelativePath, query.Path) {
			return false
		}
	}
	if len(query.Types) > 0 {
		if !slices.Contains(query.Types, n.Type) {
			return false
		}
	}
	if len(query.Tags) > 0 {
		if !n.NoteTags.IncludesAll(query.Tags) {
			return false
		}
	}
	if len(query.Attributes) > 0 {
		for key, expectedValue := range query.Attributes {
			noteValue, ok := n.Attributes[key]
			if !ok {
				return false
			}
			if expectedValue == noteValue {
				return false
			}
		}
	}

	// query.Terms is not supported as parsed notes are still not indexed

	return true
}
