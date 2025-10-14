package core

import (
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

// NoteProcessor processes a parsed note and returns a new list of parsed notes.
type NoteProcessor func(*ParsedFile, *ParsedNote) ([]*ParsedNote, error)

// noteProcessors is a map of note processor functions.
var noteProcessors = make(map[string]NoteProcessor)

// RegisterNoteProcessor registers the note processor matching a given name.
func RegisterNoteProcessor(name string, processor NoteProcessor) {
	if _, ok := noteProcessors[name]; ok {
		panic(fmt.Sprintf("NoteProcessor %q already registered", name))
	}
	noteProcessors[name] = processor
}

// FileProcessor processes a parsed file and returns a modified file.
type FileProcessor func(*ParsedFile) (*ParsedFile, error)

// fileProcessors is a map of file processor functions.
var fileProcessors = make(map[string]FileProcessor)

// RegisterFileProcessor registers the file processor matching a given name.
func RegisterFileProcessor(name string, processor FileProcessor) {
	if _, ok := fileProcessors[name]; ok {
		panic(fmt.Sprintf("FileProcessor %q already registered", name))
	}
	fileProcessors[name] = processor
}

type ParsedFile struct {
	Markdown *markdown.File

	// The root repository path
	RepositoryPath string
	// The absolute path of the file
	AbsolutePath string
	// The relative path inside the repository
	RelativePath string

	// Optional type (unlike notes, the file type is optional)
	Type string

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

func (p ParsedFile) String() string {
	return fmt.Sprintf("ParsedFile %q", p.RelativePath)
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
	Flashcards []*ParsedFlashcard
	GoLinks    []*ParsedGoto
	Reminders  []*ParsedReminder
	Memories   []*ParsedMemory

	// List items extracted from Markdown lists
	Items *Items
}

func (p ParsedNote) String() string {
	return fmt.Sprintf("ParsedNote %q in file %q at line %d", p.Title, p.RelativePath, p.Line)
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

func (p ParsedFlashcard) String() string {
	return fmt.Sprintf("ParsedFlashcard %q", p.ShortTitle)
}

type ParsedGoto struct {
	// The link text
	Text markdown.Document

	// The link destination
	URL string

	// The optional link title
	Title string

	// The optional GO name
	Name string
}

func (p ParsedGoto) String() string {
	return fmt.Sprintf("ParsedGoto %q", p.Name)
}

type ParsedReminder struct {
	// Description in Markdown of the reminder (ex: the line)
	Description markdown.Document

	// Tag value containig the formula to determine the next occurence
	Tag string `yaml:"tag"`
}

func (p ParsedReminder) String() string {
	return fmt.Sprintf("ParsedReminder `#%s`", p.Tag)
}

type ParsedMemory struct {
	// The text content of the memory
	Text markdown.Document

	// When the memory occurred
	OccurredAt time.Time
}

func (p ParsedMemory) String() string {
	return fmt.Sprintf("ParsedMemory %q occurred at %s", p.Text, p.OccurredAt.Format("2006-01-02"))
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

func (p ParsedMedia) String() string {
	return fmt.Sprintf("ParsedMedia %q", p.RawPath)
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

	// A file title must represent a untyped heading (ex: "# Golang"), a typed note (ex: "# Journal: 2025-10-09"), or a typed file (ex: "# Reading: K&R")
	fileType, shortTitle, isFile := CurrentConfigFile().MatchFileType(string(title))
	if !isFile {
		// Try matching a note type
		_, shortTitle, _ = CurrentConfigFile().MatchNoteType(string(title))
	}

	// Extract tags and attributes from shortTitle
	titleAttributes := ExtractAttributes(shortTitle, CurrentConfigFile().Attributes)

	// Merge title attributes into fileAttributes
	fileAttributes = fileAttributes.Merge(titleAttributes)

	// Strip tags and attributes from titles
	title = title.MustTransform(StripTagsAndAttributes(CurrentConfigFile().Attributes))
	shortTitle = shortTitle.MustTransform(StripTagsAndAttributes(CurrentConfigFile().Attributes))

	// Extract/Generate slug
	relativePath := CurrentRepository().GetFileRelativePath(md.AbsolutePath)
	slug := DetermineFileSlug(relativePath)
	// Slug is explicitely defined?
	if value, ok := fileAttributes["slug"]; ok {
		if v, ok := value.(string); ok {
			slug = v
		}
	}

	// File type is optional
	fileTypeName := ""
	if fileType != nil {
		fileTypeName = fileType.Name
	}

	result := &ParsedFile{
		Markdown: md,

		RepositoryPath: CurrentConfig().RootDirectory,
		AbsolutePath:   md.AbsolutePath,
		RelativePath:   relativePath,

		// Main Heading
		Type:       fileTypeName,
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

	// Apply file processors based on file tags
	result, err = applyFileProcessors(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// applyFileProcessors applies file processors based on file tags, file types, and note types
func applyFileProcessors(file *ParsedFile) (*ParsedFile, error) {
	fileTags := file.FileAttributes.Tags()

	if file.Type != "" {
		fileType := CurrentConfigFile().MustGetFileType(file.Type)
		for _, processorName := range fileType.Processors {
			processor, ok := fileProcessors[processorName]
			if !ok {
				return nil, fmt.Errorf("unknown file processor %q for file type %q", processorName, fileType.Name)
			}
			var err error
			file, err = processor(file)
			if err != nil {
				return nil, fmt.Errorf("failing file processor %q on file %q: %v", processorName, file.RelativePath, err)
			}
		}
	}

	// Check if file contains Master notes and apply master processor first
	hasMasterNotes := false
	for _, note := range file.Notes {
		if note.Type == "Master" {
			hasMasterNotes = true
			break
		}
	}

	if hasMasterNotes {
		if processor, ok := fileProcessors["master"]; ok {
			var err error
			file, err = processor(file)
			if err != nil {
				return nil, fmt.Errorf("failing file processor 'master' on file %q: %v", file.RelativePath, err)
			}
		}
	}

	// Apply processors based on file tags
	// IMPROVEMENT remove and force using file types instead?
	if fileTags.Includes("toc") {
		if processor, ok := fileProcessors["toc"]; ok {
			var err error
			file, err = processor(file)
			if err != nil {
				return nil, fmt.Errorf("failing file processor 'toc' on file %q: %v", file.RelativePath, err)
			}
		}
	}

	return file, nil
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

		// Determine the titles
		title := section.HeadingText
		noteType, shortTitle, supported := CurrentConfigFile().MatchNoteType(string(title))

		if !supported {
			// Ex: top-level heading, subsections inside a "Note:" already included in the containing note, ...
			continue
		}

		// Trim content to remove sub-notes (= typed notes)
		noteContent := section.ContentText.MustTransform(StripSubNotesTransformer)

		noteBody := noteContent.ExtractLines(2, -1) // Trim heading
		if noteBody.IsBlank() {
			// skip sections without text (= category to organize notes, not really free notes)
			continue
		}

		configAttributes := CurrentConfigFile().Attributes

		// Determine the attributes
		noteTags, noteAttributes := ExtractBlockTagsAndAttributes(noteBody, configAttributes)

		// Extract tags and attributes from shortTitle
		titleAttributes := ExtractAttributes(shortTitle, configAttributes)

		// Add title tags to noteTags
		noteTags = noteTags.Merge(titleAttributes.Tags())

		// Merge title attributes into noteAttributes
		noteAttributes = noteAttributes.Merge(titleAttributes)

		// Update titles by removing shorthands, tags, and attributes
		title = title.MustTransform(StripTagsAndAttributes(configAttributes))
		shortTitle = shortTitle.MustTransform(StripTagsAndAttributes(configAttributes))

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
				if _, shortTitle, supported := CurrentConfigFile().MatchNoteType(sectionTitle); supported {
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

		// Apply default values for missing required attributes
		defaultAttributes := CurrentConfigFile().GetAttributeDefaults(noteType.Name)
		attributes = defaultAttributes.Merge(attributes)

		// Enrich with title attribute if not already defined
		if _, ok := attributes["title"]; !ok && !title.IsBlank() {
			attributes["title"] = shortTitle.String()
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

		// Apply successive processors on the note.
		// As a processor can generate several notes, we need to loop
		// until all processors have been applied.
		for _, processorName := range noteType.Processors {
			processor, ok := noteProcessors[processorName]
			if !ok {
				return nil, fmt.Errorf("unknown note processor %q", processorName)
			}

			// The notes generated by the current processor
			newParsedNotes := []*ParsedNote{}

			for _, parsedNote := range parsedNotes {
				generatedNotes, err := processor(p, parsedNote)
				if err != nil {
					return nil, fmt.Errorf("failing processor %q on note %q (%s:%d): %v", processorName, parsedNote.ShortTitle, parsedNote.RelativePath, parsedNote.Line, err)
				}
				if len(generatedNotes) > 0 {
					newParsedNotes = append(newParsedNotes, generatedNotes...)
				}
			}

			parsedNotes = newParsedNotes
		}

		notes = append(notes, parsedNotes...)
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
		note.Memories, err = note.extractMemories()
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
		attributeConfig, ok := CurrentConfigFile().GetAttribute(key)
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

func (p *ParsedNote) extractGoLinks() ([]*ParsedGoto, error) {
	var links []*ParsedGoto

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

		link := &ParsedGoto{
			Text:  markdown.Document(text),
			URL:   url,
			Title: shortTitle,
			Name:  goName,
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
				textTitle := p.ShortTitle.MustTransform(StripTagsAndAttributes(CurrentConfigFile().Attributes))
				textItem := markdown.Document(submatch[1])
				textItem = textItem.MustTransform(StripTagsAndAttributes(CurrentConfigFile().Attributes))
				description = textTitle + " / " + textItem
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

func (p *ParsedNote) extractMemories() ([]*ParsedMemory, error) {
	var memories []*ParsedMemory

	// Get configuration to find attributes marked with memory: true
	config := CurrentConfigFile()

	// Check note-level attributes (block attributes)
	for attrName, attrValue := range p.Attributes {
		configAttr, found := config.Attributes.Find(attrName)
		if !found {
			continue
		}

		// Check if this attribute is marked as memory and is date format
		if configAttr.Memory != nil && *configAttr.Memory {
			// Parse the date value
			occurredAt, err := p.parseAttributeDate(configAttr, attrValue)
			if err != nil {
				return nil, fmt.Errorf("failed to parse date for memory attribute %q: %w", attrName, err)
			}

			if !occurredAt.IsZero() {
				cleanedText := p.ShortTitle.MustTransform(StripTagsAndAttributes(CurrentConfigFile().Attributes))
				memory := &ParsedMemory{
					Text:       cleanedText,
					OccurredAt: occurredAt,
				}
				memories = append(memories, memory)
			}
		}
	}

	// Check item-level attributes (inline attributes in list items)
	if p.Items != nil {
		for _, item := range p.Items.Children {
			memories = append(memories, p.extractMemoriesFromListItem(item)...)
		}
	}

	return memories, nil
}

func (p *ParsedNote) extractMemoriesFromListItem(item *ListItem) []*ParsedMemory {
	var memories []*ParsedMemory

	// Get configuration to find attributes marked with memory: true
	config := CurrentConfigFile()

	for attrName, attrValue := range item.Attributes {
		configAttr, found := config.Attributes.Find(attrName)
		if !found {
			continue
		}

		// Check if this attribute is marked as memory and is date format
		if configAttr.Memory != nil && *configAttr.Memory {
			// Parse the date value
			occurredAt, err := p.parseAttributeDate(configAttr, attrValue)
			if err != nil {
				// Log error but don't fail the entire extraction
				CurrentLogger().Warnf("Failed to parse date for memory attribute %q in list item: %v", attrName, err)
				continue
			}

			if !occurredAt.IsZero() {
				textTitle := p.ShortTitle.MustTransform(StripTagsAndAttributes(CurrentConfigFile().Attributes))
				textItem := item.Text.MustTransform(StripTagsAndAttributes(CurrentConfigFile().Attributes))
				text := textTitle + " / " + textItem
				memory := &ParsedMemory{
					Text:       text,
					OccurredAt: occurredAt,
				}
				memories = append(memories, memory)
			}
		}
	}

	// Recursively check children
	for _, child := range item.Children {
		memories = append(memories, p.extractMemoriesFromListItem(child)...)
	}

	return memories
}

func (p *ParsedNote) parseAttributeDate(configAttr *ConfigAttribute, attrValue any) (time.Time, error) {
	// Convert attribute value to string
	valueStr, ok := CastStringFn(attrValue)
	if !ok {
		return time.Time{}, fmt.Errorf("attribute value is not a string: %v", attrValue)
	}

	// Parse the date based on the format
	format := configAttr.Format
	if format == "" {
		return time.Time{}, fmt.Errorf("no format specified for date attribute")
	}

	// Convert custom format to Go time format
	goFormat := convertDateFormatToGo(format)

	parsedTime, err := time.Parse(goFormat, valueStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date %q with format %q: %w", valueStr, goFormat, err)
	}

	return parsedTime, nil
}

func convertDateFormatToGo(format string) string {
	// Convert common date formats to Go's reference time format
	// This is a simple conversion; can be extended as needed
	switch format {
	case "yyyy-mm-dd":
		return "2006-01-02"
	case "mm/dd/yyyy":
		return "01/02/2006"
	case "dd/mm/yyyy":
		return "02/01/2006"
	case "yyyy/mm/dd":
		return "2006/01/02"
	default:
		// Try to convert format by replacing common patterns
		goFormat := strings.ReplaceAll(format, "yyyy", "2006")
		goFormat = strings.ReplaceAll(goFormat, "mm", "01")
		goFormat = strings.ReplaceAll(goFormat, "dd", "02")
		return goFormat
	}
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

// FindGotoByName searches for a goto from its name.
func (p *ParsedNote) FindGotoByName(name string) (*ParsedGoto, bool) {
	for _, gotoLink := range p.GoLinks {
		if gotoLink.Name == name {
			return gotoLink, true
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

// FindMemoryByText searches for a memory from its text.
func (p *ParsedNote) FindMemoryByText(text markdown.Document) (*ParsedMemory, bool) {
	for _, memory := range p.Memories {
		if memory.Text == text {
			return memory, true
		}
	}
	return nil, false
}

// StripTagsAndAttributes removes all tags and attributes from a NoteWriter note.
func StripTagsAndAttributes(attributes ConfigAttributes) markdown.Transformer {
	return func(doc markdown.Document) (markdown.Document, error) {
		return doc.MustTransform(
			StripTags(),
			StripAttributes(attributes)).
			TrimSpace(), nil
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
			_, _, supported := CurrentConfigFile().MatchNoteType(headingText)
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

// SimplifyMarkdown simplifies a Markdown document by removing all tags, attributes and emphasis.
func SimplifyMarkdown(configAttributes ConfigAttributes) markdown.Transformer {
	return func(doc markdown.Document) (markdown.Document, error) {
		return doc.MustTransform(
			StripTagsAndAttributes(configAttributes),
			markdown.StripEmphasis(),
		), nil
	}
}
