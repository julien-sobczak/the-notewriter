package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"regexp"
	"regexp/syntax"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/julien-sobczak/the-notewriter/internal/helpers"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// NoteLongTitleSeparator represents the separator when determine the long title of a note.
const NoteLongTitleSeparator string = " / "

const missingMediaOID string = "4044044044044044044044044044044044044040"

type NoteKind string

const (
	KindFree       NoteKind = "free"
	KindReference  NoteKind = "reference"
	KindNote       NoteKind = "note"
	KindFlashcard  NoteKind = "flashcard"
	KindCheatsheet NoteKind = "cheatsheet"
	KindQuote      NoteKind = "quote"
	KindJournal    NoteKind = "journal"
	KindTodo       NoteKind = "todo"
	KindArtwork    NoteKind = "artwork"
	KindSnippet    NoteKind = "snippet"
	// Edit website/docs/guides/notes.md when adding new kinds
)

// Regex to validate and/or extract information from notes
var (
	// Kinds
	regexReference  = regexp.MustCompile(`^(?i)Reference:\s*(.*)$`)  // Ex: `# Reference: Go History`
	regexNote       = regexp.MustCompile(`^(?i)Note:\s*(.*)$`)       // Ex: `# Note: On Go Logo`
	regexFlashcard  = regexp.MustCompile(`^(?i)Flashcard:\s*(.*)$`)  // Ex: `# Flashcard: Goroutines Syntax`
	regexCheatsheet = regexp.MustCompile(`^(?i)Cheatsheet:\s*(.*)$`) // Ex: `# Cheatsheet: How to start a goroutine`
	regexQuote      = regexp.MustCompile(`^(?i)Quote:\s*(.*)$`)      // Ex: `# Quote: Marcus Aurelius on Doing`
	regexTodo       = regexp.MustCompile(`^(?i)Todo:\s*(.*)$`)       // Ex: `# Todo: Backlog`
	regexArtwork    = regexp.MustCompile(`^(?i)Artwork:\s*(.*)$`)    // Ex: `# Artwork: Vincent van Gogh`
	regexSnippet    = regexp.MustCompile(`^(?i)Snippet:\s*(.*)$`)    // Ex: `# Snippet: Ideas for post title`
	regexChecklist  = regexp.MustCompile(`^(?i)Checklist:\s*(.*)$`)  // Ex: `# Checklist: Travel`
	regexJournal    = regexp.MustCompile(`^(?i)Journal:\s*(.*)$`)    // Ex: `# Journal: 2023-01-01`
)

type Note struct {
	OID string `yaml:"oid"`

	// File containing the note
	FileOID string `yaml:"file_oid"`
	File    *File  `yaml:"-"` // Lazy-loaded

	// Parent Note surrounding the note
	ParentNoteOID string `yaml:"parent_note_oid"`
	ParentNote    *Note  `yaml:"-"` // Lazy-loaded

	// Type of note
	NoteKind NoteKind `yaml:"kind"`

	// Original title of the note without leading # characters
	Title string `yaml:"title"`
	// Long title of the note without the kind prefix but prefixed by parent note's short titles
	LongTitle string `yaml:"long_title"`
	// Short title of the note without the kind prefix
	ShortTitle string `yaml:"short_title"`

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path"`
	// The full wikilink to this note (without the extension)
	Wikilink string `yaml:"wikilink"`

	// Merged attributes
	Attributes map[string]interface{} `yaml:"attributes,omitempty"`

	// Merged tags
	Tags []string `yaml:"tags,omitempty"`

	// Line number (1-based index) of the note section title
	Line int `yaml:"line"`

	// Content in various formats (best for editing, rendering, writing, etc.)
	ContentRaw      string `yaml:"content_raw"`
	Hash            string `yaml:"content_hash"`
	TitleMarkdown   string `yaml:"title_markdown"`
	TitleHTML       string `yaml:"title_html"`
	TitleText       string `yaml:"title_text"`
	ContentMarkdown string `yaml:"content_markdown"`
	ContentHTML     string `yaml:"content_html"`
	ContentText     string `yaml:"content_text"`
	CommentMarkdown string `yaml:"comment_markdown,omitempty"`
	CommentHTML     string `yaml:"comment_html,omitempty"`
	CommentText     string `yaml:"comment_text,omitempty"`

	// Timestamps to track changes
	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	DeletedAt     time.Time `yaml:"deleted_at,omitempty"`
	LastCheckedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}

// NewOrExistingNote loads and updates an existing note or creates a new one if new.
func NewOrExistingNote(f *File, parent *Note, title string, content string, lineNumber int) *Note {
	content = strings.TrimSpace(content)

	note, _ := CurrentCollection().FindNoteByWikilink(f.RelativePath + "#" + title)
	if note != nil {
		note.update(f, title, content, lineNumber)
		return note
	}

	hash := helpers.Hash([]byte(content))
	note, _ = CurrentCollection().FindMatchingNotes(f.RelativePath, title, hash)
	if note != nil {
		note.update(f, title, content, lineNumber)
		return note
	}

	return NewNote(f, parent, title, content, lineNumber)
}

// NewNote creates a new note from given attributes.
func NewNote(f *File, parent *Note, title string, content string, lineNumber int) *Note {
	rawContent := strings.TrimSpace(content)

	_, kind, shortTitle := isSupportedNote(title)

	var titles []string
	if f.ShortTitle != "" {
		titles = append(titles, f.ShortTitle)
	}
	if parent != nil {
		titles = append(titles, parent.ShortTitle)
	}
	titles = append(titles, shortTitle)
	longTitle := FormatLongTitle(titles...)

	n := &Note{
		OID:          NewOID(),
		FileOID:      f.OID,
		File:         f,
		NoteKind:     kind,
		Title:        title,
		LongTitle:    longTitle,
		ShortTitle:   shortTitle,
		RelativePath: f.RelativePath,
		Wikilink:     f.Wikilink + "#" + strings.TrimSpace(title),
		Line:         lineNumber,
		CreatedAt:    clock.Now(),
		UpdatedAt:    clock.Now(),
		new:          true,
		stale:        true,
	}
	if parent != nil {
		n.ParentNote = parent
		n.ParentNoteOID = parent.OID
		n.LongTitle = parent.LongTitle + " / " + n.LongTitle
	}

	n.updateContent(rawContent)

	CurrentDB().WIP().Register(n)

	return n
}

/* Object */

func (n *Note) Kind() string {
	return "note"
}

func (n *Note) UniqueOID() string {
	return n.OID
}

func (n *Note) ModificationTime() time.Time {
	return n.UpdatedAt
}

func (n *Note) Refresh() (bool, error) {
	// Simply force the content to be reevaluated to force inluded notes to be reread
	n.updateContent(n.ContentRaw)
	return n.stale, nil
}

func (n *Note) State() State {
	if !n.DeletedAt.IsZero() {
		return Deleted
	}
	if n.new {
		return Added
	}
	if n.stale {
		return Modified
	}
	return None
}

func (n *Note) ForceState(state State) {
	switch state {
	case Added:
		n.new = true
	case Deleted:
		n.DeletedAt = clock.Now()
	}
	n.stale = true
}

func (n *Note) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(n)
	if err != nil {
		return err
	}
	return nil
}

func (n *Note) Write(w io.Writer) error {
	data, err := yaml.Marshal(n)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (n *Note) SubObjects() []StatefulObject {
	var objs []StatefulObject
	for _, object := range n.GetLinks() {
		objs = append(objs, object)
		objs = append(objs, object.SubObjects()...)
	}
	for _, object := range n.GetMedias() {
		objs = append(objs, object)
		objs = append(objs, object.SubObjects()...)
	}
	for _, object := range n.GetReminders() {
		objs = append(objs, object)
		objs = append(objs, object.SubObjects()...)
	}
	return objs
}

func (n *Note) Blobs() []*BlobRef {
	// Use Media.Blobs() instead
	return nil
}

func (n *Note) Relations() []*Relation {
	var relations []*Relation

	// Utility function to append wikilink to the returned relations
	addWikilink := func(wikilinkTxt string, relationType string) {
		wikilink, err := NewWikilink(wikilinkTxt)
		if err != nil {
			// Ignore malformed links
			return
		}

		if wikilink.Section() != "" {
			note, _ := CurrentCollection().FindNoteByWikilink(wikilink.Link)
			if note != nil {
				relations = append(relations, &Relation{
					SourceOID:  n.OID,
					SourceKind: "note",
					TargetOID:  note.OID,
					TargetKind: "note",
					Type:       relationType,
				})
			}
		} else {
			file, _ := CurrentCollection().FindFileByWikilink(wikilink.Link)
			if file != nil {
				relations = append(relations, &Relation{
					SourceOID:  n.OID,
					SourceKind: "note",
					TargetOID:  file.OID,
					TargetKind: "file",
					Type:       relationType,
				})
			}
		}
	}

	// Search for embedded notes
	reEmbeddedNote := regexp.MustCompile(`^!\[\[(.*)(?:\|.*)?\]\]\s*`)
	matches := reEmbeddedNote.FindAllStringSubmatch(n.ContentRaw, -1)
	for _, match := range matches {
		wikilink := match[1]
		addWikilink(wikilink, "includes")
	}

	// Check attribute "source"
	if n.HasAttribute("source") {
		source := n.GetAttribute("source").(string) // Enforced by linter
		if MatchWikilink(source) {
			addWikilink(source, "references")
		}
	}

	// Check attribute "references"
	if n.HasAttribute("references") {
		references := n.GetAttribute("references").([]interface{}) // Enforced by linter
		for _, referenceRaw := range references {
			if reference, ok := referenceRaw.(string); ok {
				if MatchWikilink(reference) {
					addWikilink(reference, "referenced_by")
				}
			}
		}
	}

	// Check attribute "inspirations"
	if n.HasAttribute("inspirations") {
		inspirations := n.GetAttribute("inspirations").([]interface{}) // Enforced by linter
		for _, inspirationRaw := range inspirations {
			if inspiration, ok := inspirationRaw.(string); ok {
				if MatchWikilink(inspiration) {
					addWikilink(inspiration, "inspired_by")
				}
			}
		}
	}

	return relations
}

func (n Note) String() string {
	return fmt.Sprintf("note %q [%s]", n.Title, n.OID)
}

/* Update */

func (n *Note) update(f *File, title string, content string, lineNumber int) {
	rawContent := strings.TrimSpace(content)

	if f.OID != n.FileOID {
		n.File = f
		n.FileOID = f.OID
		n.stale = true
	}

	if rawContent != n.ContentRaw {
		n.updateContent(rawContent)
		n.stale = true
	}
	if lineNumber != n.Line {
		n.Line = lineNumber
		n.stale = true
	}
}

/* State Management */

func (n *Note) New() bool {
	return n.new
}

func (n *Note) Updated() bool {
	return n.stale
}

/* Parsing */

func (n *Note) parseContentRaw() (mdTitle, htmlTitle, txtTitle string, mdContent, htmlContent, txtContent string, mdComment, htmlComment, txtComment string) {
	// Always remove block tags and attributes in all formats
	content := StripBlockTagsAndAttributes(n.ContentRaw)

	// Always remove HTML comments
	content = text.StripHTMLComments(content)

	// Always replace Asciidoc special characters
	content = markdown.ReplaceAsciidocCharacterSubstitutions(content)
	// Extract optional personal comment
	content, comment := markdown.StripComment(content)

	// Replace local-specific links by generic OID links
	content = n.ReplaceMediasByOIDLinks(content)

	if comment != "" {
		mdComment = strings.TrimSpace(comment)
		txtComment = strings.TrimSpace(comment)
		htmlComment = markdown.ToInlineHTML(strings.ReplaceAll(comment, "\n", " "))
	}

	mdTitle = "# " + n.LongTitle
	htmlTitle = "<h1>" + markdown.ToInlineHTML(n.LongTitle) + "</h1>"
	txtTitle = markdown.ToText(n.LongTitle) + "\n" + text.Repeat("=", len(n.LongTitle))

	// Quotes are processed differently
	if n.NoteKind == KindQuote {
		quote, attribution := markdown.ExtractQuote(content)

		// Turn every text line into a quote
		// Add the attribute name or author in suffix
		// Ex:
		//   `@name: Walt Disney`
		//
		//   The way to get started is to quit
		//   talking and begin doing.
		//
		// Becomes:
		//
		//   > The way to get started is to quit
		//   > talking and begin doing.
		//   > — Walt Disney

		if attribution == "" {
			attribution = n.GetAttributeString("name", n.GetAttributeString("author", ""))
		}
		source := n.GetAttributeString("source", "")
		if strings.Contains(source, "[[") {
			// Ignore source containing wikilink.
			// Ideally, we would retrieve the correspond note to retrieve its title.
			source = ""
		}

		// Markdown
		mdContent += text.PrefixLines(quote, "> ")
		txtContent += text.PrefixLines(quote, "> ")
		if attribution != "" {
			mdContent += "> — " + attribution + "\n"
			txtContent += "> — " + attribution + "\n"
		}
		// HTML
		if attribution == "" {
			htmlContent += fmt.Sprintf(`<figure>
	<blockquote>
		%s
	</blockquote>
</figure>`, markdown.ToHTML(quote))
		} else if source == "" {
			htmlContent += fmt.Sprintf(`<figure>
	<blockquote>
		%s
	</blockquote>
	<figcaption>— %s</figcaption>
</figure>`, markdown.ToHTML(quote), markdown.ToInlineHTML(attribution))
		} else {
			htmlContent += fmt.Sprintf(`<figure>
	<blockquote>
		%s
	</blockquote>
	<figcaption>— %s <cite>%s</cite></figcaption>
</figure>`, markdown.ToHTML(quote), markdown.ToInlineHTML(attribution), markdown.ToInlineHTML(source))
		}

		mdContent = strings.TrimSpace(mdContent)
		htmlContent = strings.TrimSpace(htmlContent)
		txtContent = strings.TrimSpace(txtContent)

		return
	}

	// Manage embedded notes
	// We process as usual the other lines but inject the embedded note content.
	lines := strings.Split(content, "\n")
	reEmbeddedNote := regexp.MustCompile(`^!\[\[(.*)(?:\|.*)?\]\]\s*`)
	var currentBlock strings.Builder
	for _, line := range lines {
		matches := reEmbeddedNote.FindStringSubmatch(line)
		if matches != nil {
			if currentBlock.Len() > 0 {
				blockContent := currentBlock.String()
				mdContent += markdown.ToMarkdown(blockContent) + "\n\n"
				htmlContent += markdown.ToHTML(blockContent) + "\n\n"
				txtContent += markdown.ToText(blockContent) + "\n\n"
				currentBlock.Reset()
			}
			wikilink := matches[1]
			note, _ := CurrentCollection().FindNoteByWikilink(wikilink)
			if note == nil {
				note = CurrentDB().WIP().FindNoteByWikilink(wikilink)
			}
			// Ignore missing notes, this one will be reprocessed later
			if note != nil {
				mdContent += note.ContentMarkdown + "\n\n"
				htmlContent += note.ContentHTML + "\n\n"
				txtContent += note.ContentText + "\n\n"
			} else {
				// Print the missing link, otherwise the note content may be weird
				mdContent += line + "\n\n"
				htmlContent += "<del>" + wikilink + "</del>\n\n"
				txtContent += markdown.ToText(line) + "\n\n"
			}
		} else {
			currentBlock.WriteString(line)
			currentBlock.WriteRune('\n')
		}
	}
	if currentBlock.Len() > 0 {
		blockContent := currentBlock.String()
		mdContent += markdown.ToMarkdown(blockContent)
		htmlContent += markdown.ToHTML(blockContent)
		txtContent += markdown.ToText(blockContent)
	}

	mdContent = strings.TrimSpace(mdContent)
	htmlContent = strings.TrimSpace(htmlContent)
	txtContent = strings.TrimSpace(txtContent)

	return
}

// ReplaceMediasByOIDLinks replaces all non-dangling links by a OID fake link.
func (n *Note) ReplaceMediasByOIDLinks(md string) string {
	regexMedias := regexp.MustCompile(`!\[.*?\]\((\S*?)(?:\s+"(.*?)")?\)`)

	var result strings.Builder
	prevIndex := 0
	matches := regexMedias.FindAllStringSubmatchIndex(md, -1)
	for _, match := range matches {
		result.WriteString(md[prevIndex:match[2]])

		link := md[match[2]:match[3]]
		relativePath, err := CurrentCollection().GetNoteRelativePath(n.GetFile().RelativePath, link)
		if err != nil {
			// Use a 404 image
			result.WriteString("oid:" + missingMediaOID)
			prevIndex = match[3]
			continue
		}

		media, err := CurrentCollection().FindMediaByRelativePath(relativePath)
		if err != nil || media == nil {
			// Use a 404 image
			result.WriteString("oid:" + missingMediaOID)
			prevIndex = match[3]
			continue
		}

		if media.Dangling {
			// Use a 404 image
			result.WriteString("oid:" + missingMediaOID)
			prevIndex = match[3]
			continue
		}

		result.WriteString("oid:" + media.OID)
		prevIndex = match[3]
	}
	// Add remaining text
	result.WriteString(md[prevIndex:])

	return result.String()
}

func (n *Note) updateContent(rawContent string) {

	prevContentMarkdown := n.ContentMarkdown
	prevAttributes := n.Attributes

	n.ContentRaw = strings.TrimSpace(rawContent)
	n.Hash = helpers.Hash([]byte(n.ContentRaw))

	tags, attributes := ExtractBlockTagsAndAttributes(n.ContentRaw)

	// Append note title in a attribute title if not already present
	if _, ok := attributes["title"]; !ok {
		attributes["title"] = n.ShortTitle
	}

	// Merge with parent attributes
	if n.ParentNoteOID == "" {
		attributes = n.mergeAttributes(n.GetFile().GetAttributes(), nil, attributes)
	} else {
		attributes = n.mergeAttributes(n.GetFile().GetAttributes(), n.GetParentNote().GetNoteAttributes(), attributes)
	}

	// Merge with parent tags
	if n.ParentNoteOID == "" {
		tags = mergeTags(n.GetFile().GetTags(), tags)
	} else {
		tags = mergeTags(n.GetParentNote().GetTags(), tags)
	}

	n.Tags = tags
	n.Attributes = attributes

	// Reread content as tags and attributes previously defined on the note can influence the output.
	mdTitle, htmlTitle, txtTitle, mdContent, htmlContent, txtContent, mdComment, htmlComment, txtComment := n.parseContentRaw()
	n.TitleMarkdown = mdTitle
	n.TitleHTML = htmlTitle
	n.TitleText = txtTitle
	n.ContentMarkdown = mdContent
	n.ContentHTML = htmlContent
	n.ContentText = txtContent
	n.CommentMarkdown = mdComment
	n.CommentHTML = htmlComment
	n.CommentText = txtComment

	if prevContentMarkdown != n.ContentMarkdown || !reflect.DeepEqual(prevAttributes, n.Attributes) {
		n.stale = true
	}
}

// mergeAttributes is similar to generic mergeAttributes function but filter to exclude non-inheritable attributes.
func (n *Note) mergeAttributes(fileAttributes, parentNoteAttributes, noteAttributes map[string]interface{}) map[string]interface{} {
	inheritableFileAttributes := fileAttributes
	inheritableParentNoteAttributes := FilterNonInheritableAttributes(parentNoteAttributes, n.RelativePath, n.NoteKind)
	ownAttributes := noteAttributes
	return MergeAttributes(inheritableFileAttributes, inheritableParentNoteAttributes, ownAttributes)
}

// GetNoteAttributes returns the attributes specifically present on the note.
func (n *Note) GetNoteAttributes() map[string]interface{} {
	_, attributes := ExtractBlockTagsAndAttributes(n.ContentRaw)
	return CastAttributes(attributes, GetSchemaAttributeTypes())
}

// GetNoteTags returns the tags specifically present on the note.
func (n *Note) GetNoteTags() []string {
	tags, _ := ExtractBlockTagsAndAttributes(n.ContentRaw)
	return tags
}

// GetFile returns the containing file, loading it from database if necessary.
func (n *Note) GetFile() *File {
	if n.FileOID == "" {
		return nil
	}
	if n.File == nil {
		file, err := CurrentCollection().LoadFileByOID(n.FileOID)
		if err != nil {
			log.Fatalf("Unable to find file %q: %v", n.FileOID, err)
		}
		n.File = file
	}
	return n.File
}

// GetParentNote returns the parent note, loading it from database if necessary.
func (n *Note) GetParentNote() *Note {
	if n.ParentNoteOID == "" {
		return nil
	}
	if n.ParentNote == nil {
		note, err := CurrentCollection().LoadNoteByOID(n.ParentNoteOID)
		if err != nil {
			log.Fatalf("Unable to note file %q: %v", n.ParentNoteOID, err)
		}
		n.ParentNote = note
	}
	return n.ParentNote
}

func (n *Note) GetAttributes() map[string]interface{} {
	// Present to be consistent with File.GetAttributes()
	return n.Attributes
}

func (n *Note) HasAttribute(name string) bool {
	_, ok := n.Attributes[name]
	return ok
}

func (n *Note) SetAttribute(name string, value interface{}) {
	n.Attributes[name] = value
}

func (n *Note) GetAttribute(name string) interface{} {
	if value, ok := n.Attributes[name]; ok {
		return value
	}
	return nil
}

func (n *Note) GetAttributeString(name, defaultValue string) string {
	value := n.GetAttribute(name)
	if value == nil {
		return defaultValue
	}
	return fmt.Sprintf("%v", value)
}

func (n *Note) GetTags() []string {
	// Present to be consistent with File.GetTags()
	return n.Tags
}

// HasTag returns if a file has a given tag.
func (n *Note) HasTag(name string) bool {
	return slices.Contains(n.GetTags(), name)
}

func isSupportedNote(text string) (bool, NoteKind, string) {
	if m := regexReference.FindStringSubmatch(text); m != nil {
		return true, KindReference, m[1]
	}
	if m := regexNote.FindStringSubmatch(text); m != nil {
		return true, KindNote, m[1]
	}
	if m := regexCheatsheet.FindStringSubmatch(text); m != nil {
		return true, KindCheatsheet, m[1]
	}
	if m := regexFlashcard.FindStringSubmatch(text); m != nil {
		return true, KindFlashcard, m[1]
	}
	if m := regexQuote.FindStringSubmatch(text); m != nil {
		return true, KindQuote, m[1]
	}
	if m := regexTodo.FindStringSubmatch(text); m != nil {
		return true, KindTodo, m[1]
	}
	if m := regexArtwork.FindStringSubmatch(text); m != nil {
		return true, KindArtwork, m[1]
	}
	if m := regexSnippet.FindStringSubmatch(text); m != nil {
		return true, KindArtwork, m[1]
	}
	if m := regexChecklist.FindStringSubmatch(text); m != nil {
		return true, KindArtwork, m[1]
	}
	if m := regexJournal.FindStringSubmatch(text); m != nil {
		return true, KindJournal, m[1]
	}
	return false, KindFree, text
}

/* Sub Objects */

// GetMedias extracts medias from the note.
func (n *Note) GetMedias() []*Media {
	return extractMediasFromMarkdown(n.GetFile().RelativePath, n.ContentRaw)
}

// GetLinks extracts special links from a note.
func (n *Note) GetLinks() []*Link {
	var links []*Link

	reLink := regexp.MustCompile(`(?:^|[^!])\[(.*?)\]\("?(http[^\s"]*)"?(?:\s+["'](.*?)["'])?\)`)
	// Note: Markdown images uses the same syntax as links but precedes the link by !
	reTitle := regexp.MustCompile(`(?:(.*)\s+)?#go\/(\S+).*`)

	matches := reLink.FindAllStringSubmatch(n.ContentRaw, -1)
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

		link := NewOrExistingLink(n, text, url, shortTitle, goName)
		links = append(links, link)
	}

	return links
}

// GetReminders extracts reminders from the note.
func (n *Note) GetReminders() []*Reminder {
	var reminders []*Reminder

	reReminders := regexp.MustCompile("`(#reminder-(\\S+))`")
	reList := regexp.MustCompile(`^\s*(?:[-+*]|\d+[.])\s+(?:\[.\]\s+)?(.*)\s*$`)

	for _, line := range strings.Split(n.ContentRaw, "\n") {
		matches := reReminders.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			tag := match[1]
			_ = match[2] // expression

			description := n.ShortTitle

			submatch := reList.FindStringSubmatch(line)
			if submatch != nil {
				// Reminder for a list element
				description = RemoveTagsAndAttributes(submatch[1]) // Remove tags
			}

			reminder, err := NewOrExistingReminder(n, description, tag)
			if err != nil {
				log.Fatal(err)
			}
			reminders = append(reminders, reminder)
		}
	}

	return reminders
}

/* State Management */

func (n *Note) Check() error {
	CurrentLogger().Debugf("Checking note %s...", n.Wikilink)
	n.LastCheckedAt = clock.Now()
	query := `
		UPDATE note
		SET last_checked_at = ?
		WHERE oid = ?;`
	if _, err := CurrentDB().Client().Exec(query, timeToSQL(n.LastCheckedAt), n.OID); err != nil {
		return err
	}
	query = `
		UPDATE flashcard
		SET last_checked_at = ?
		WHERE note_oid = ?;`
	if _, err := CurrentDB().Client().Exec(query, timeToSQL(n.LastCheckedAt), n.OID); err != nil {
		return err
	}
	query = `
		UPDATE link
		SET last_checked_at = ?
		WHERE note_oid = ?;`
	if _, err := CurrentDB().Client().Exec(query, timeToSQL(n.LastCheckedAt), n.OID); err != nil {
		return err
	}
	query = `
		UPDATE reminder
		SET last_checked_at = ?
		WHERE note_oid = ?;`
	if _, err := CurrentDB().Client().Exec(query, timeToSQL(n.LastCheckedAt), n.OID); err != nil {
		return err
	}

	return nil
}

func (n *Note) Save() error {
	var err error
	n.UpdatedAt = clock.Now()
	n.LastCheckedAt = clock.Now()
	switch n.State() {
	case Added:
		err = n.Insert()
	case Modified:
		err = n.Update()
	case Deleted:
		err = n.Delete()
	default:
		err = n.Check()
	}
	if err != nil {
		return err
	}
	n.new = false
	n.stale = false
	return nil
}

func (n *Note) Insert() error {
	CurrentLogger().Debugf("Inserting note %s...", n.Wikilink)
	query := `
		INSERT INTO note(
			oid,
			file_oid,
			note_oid,
			kind,
			relative_path,
			wikilink,
			title,
			long_title,
			short_title,
			attributes,
			tags,
			"line",
			content_raw,
			hashsum,
			title_markdown,
			title_html,
			title_text,
			content_markdown,
			content_html,
			content_text,
			comment_markdown,
			comment_html,
			comment_text,
			created_at,
			updated_at,
			last_checked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	attributesJSON, err := AttributesJSON(n.Attributes)
	if err != nil {
		return err
	}

	_, err = CurrentDB().Client().Exec(query,
		n.OID,
		n.FileOID,
		n.ParentNoteOID,
		n.NoteKind,
		n.RelativePath,
		n.Wikilink,
		n.Title,
		n.LongTitle,
		n.ShortTitle,
		attributesJSON,
		strings.Join(n.Tags, ","),
		n.Line,
		n.ContentRaw,
		n.Hash,
		n.TitleMarkdown,
		n.TitleHTML,
		n.TitleText,
		n.ContentMarkdown,
		n.ContentHTML,
		n.ContentText,
		n.CommentMarkdown,
		n.CommentHTML,
		n.CommentText,
		timeToSQL(n.CreatedAt),
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *Note) Update() error {
	CurrentLogger().Debugf("Updating note %s...", n.Wikilink)
	query := `
		UPDATE note
		SET
			file_oid = ?,
			note_oid = ?,
			kind = ?,
			relative_path = ?,
			wikilink = ?,
			title = ?,
			long_title = ?,
			short_title = ?,
			attributes = ?,
			tags = ?,
			"line" = ?,
			content_raw = ?,
			hashsum = ?,
			title_markdown = ?,
			title_html = ?,
			title_text = ?,
			content_markdown = ?,
			content_html = ?,
			content_text = ?,
			comment_markdown = ?,
			comment_html = ?,
			comment_text = ?,
			updated_at = ?,
			last_checked_at = ?
		WHERE oid = ?;
	`

	attributesJSON, err := AttributesJSON(n.Attributes)
	if err != nil {
		return err
	}

	_, err = CurrentDB().Client().Exec(query,
		n.FileOID,
		n.ParentNoteOID,
		n.NoteKind,
		n.RelativePath,
		n.Wikilink,
		n.Title,
		n.LongTitle,
		n.ShortTitle,
		attributesJSON,
		strings.Join(n.Tags, ","),
		n.Line,
		n.ContentRaw,
		n.Hash,
		n.TitleMarkdown,
		n.TitleHTML,
		n.TitleText,
		n.ContentMarkdown,
		n.ContentHTML,
		n.ContentText,
		n.CommentMarkdown,
		n.CommentHTML,
		n.CommentText,
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.LastCheckedAt),
		n.OID,
	)

	return err
}

func (n *Note) Delete() error {
	CurrentLogger().Debugf("Deleting note %s...", n.Wikilink)
	query := `DELETE FROM note WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query, n.OID)
	return err
}

// CountNotes returns the total number of notes.
func (c *Collection) CountNotes() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// CountNotesByKind returns the total number of notes for every kind.
func (c *Collection) CountNotesByKind() (map[NoteKind]int, error) {
	res := map[NoteKind]int{
		KindFree:       0,
		KindReference:  0,
		KindNote:       0,
		KindFlashcard:  0,
		KindCheatsheet: 0,
		KindQuote:      0,
		KindJournal:    0,
		KindTodo:       0,
		KindArtwork:    0,
		KindSnippet:    0,
	}

	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note where kind = ?`, KindFree).Scan(&count); err == nil {
		res[KindFree] = count
	}
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note where kind = ?`, KindReference).Scan(&count); err == nil {
		res[KindReference] = count
	}
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note where kind = ?`, KindNote).Scan(&count); err == nil {
		res[KindNote] = count
	}
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note where kind = ?`, KindFlashcard).Scan(&count); err == nil {
		res[KindFlashcard] = count
	}
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note where kind = ?`, KindCheatsheet).Scan(&count); err == nil {
		res[KindCheatsheet] = count
	}
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note where kind = ?`, KindQuote).Scan(&count); err == nil {
		res[KindQuote] = count
	}
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note where kind = ?`, KindJournal).Scan(&count); err == nil {
		res[KindJournal] = count
	}
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note where kind = ?`, KindTodo).Scan(&count); err == nil {
		res[KindTodo] = count
	}
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note where kind = ?`, KindArtwork).Scan(&count); err == nil {
		res[KindArtwork] = count
	}
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note where kind = ?`, KindSnippet).Scan(&count); err == nil {
		res[KindSnippet] = count
	}

	return res, nil
}

// CountTags returns the tags with their associated count.
func (c *Collection) CountTags() (map[string]int, error) {
	result := make(map[string]int)

	// See https://www.vivekkalyan.com/splitting-comma-seperated-fields-sqlite
	rows, err := CurrentDB().Client().Query(`
		WITH RECURSIVE split(tag, str) AS (
			SELECT '', tags||',' FROM note
			UNION ALL SELECT
			substr(str, 0, instr(str, ',')),
			substr(str, instr(str, ',')+1)
			FROM split WHERE str!=''
		)
		SELECT distinct tag, count(*)
		FROM split
		WHERE tag!=''
		group by tag;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tag string
		var count int

		err = rows.Scan(
			&tag,
			&count,
		)
		if err != nil {
			return nil, err
		}
		result[tag] = count
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CountAttributes returns the attributes with their associated count.
func (c *Collection) CountAttributes() (map[string]int, error) {
	result := make(map[string]int)

	// See https://database.guide/sqlite-json_each/
	rows, err := CurrentDB().Client().Query(`
		SELECT tt.attribute, count(*) FROM (
			SELECT j.key as attribute, j.value
			from note t, json_each(t.attributes) j
		) AS tt
		GROUP BY tt.attribute;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var attribute string
		var count int

		err = rows.Scan(
			&attribute,
			&count,
		)
		if err != nil {
			return nil, err
		}
		result[attribute] = count
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Collection) DumpNotes() error {
	notes, err := QueryNotes(CurrentDB().Client(), "")
	if err != nil {
		return err
	}
	for _, note := range notes {
		CurrentLogger().Infof("Note %s [%s] [[%s]]\n", note.LongTitle, note.OID, note.Wikilink)
	}
	return nil
}

func (c *Collection) LoadNoteByOID(oid string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE oid = ?`, oid)
}

func (c *Collection) FindNotesByFileOID(oid string) ([]*Note, error) {
	return QueryNotes(CurrentDB().Client(), `WHERE file_oid = ?`, oid)
}

func (c *Collection) FindNoteByTitle(title string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE title = ?`, title)
}

func (c *Collection) FindNoteByHash(hash string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE hashsum = ?`, hash)
}

func (c *Collection) FindMatchingNotes(relativePath, title, hash string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE relative_path = ? AND (title = ? OR hashsum = ?)`, relativePath, title, hash)
}

func (c *Collection) FindNoteByWikilink(wikilink string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE wikilink LIKE ?`, "%"+wikilink)
}

func (c *Collection) FindNotesByWikilink(wikilink string) ([]*Note, error) {
	return QueryNotes(CurrentDB().Client(), `WHERE wikilink LIKE ?`, "%"+wikilink)
}

func (c *Collection) FindNotesLastCheckedBefore(point time.Time, path string) ([]*Note, error) {
	if path == "." {
		path = ""
	}
	return QueryNotes(CurrentDB().Client(), `WHERE last_checked_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

// SearchNotes query notes to find the ones matching a list of criteria.
//
// Examples:
//
//	tag:favorite kind:reference kind:note path:projects/
func (c *Collection) SearchNotes(q string) ([]*Note, error) {
	query, err := ParseQuery(q)
	if err != nil {
		return nil, err
	}

	// Prepare SQL values
	var querySQL strings.Builder
	querySQL.WriteString("SELECT note_fts.rowid ")
	querySQL.WriteString("FROM note_fts ")
	querySQL.WriteString("JOIN note on note.oid = note_fts.oid ")
	querySQL.WriteString("WHERE note.oid IS NOT NULL ") // useless but simplify the query building
	if len(query.Kinds) > 0 {
		var kindsSQL []string
		for _, kind := range query.Kinds {
			kindsSQL = append(kindsSQL, fmt.Sprintf(`"%s"`, kind))
		}
		querySQL.WriteString(fmt.Sprintf("AND note.kind IN (%s) ", strings.Join(kindsSQL, ",")))
	}
	if len(query.Tags) > 0 {
		querySQL.WriteString("AND ( ")
		for _, tag := range query.Tags {
			querySQL.WriteString(fmt.Sprintf("  note.tags LIKE '%%%s%%' ", tag))
		}
		querySQL.WriteString(") ")
	}
	if len(query.Attributes) > 0 {
		querySQL.WriteString("AND ( ")
		for name, value := range query.Attributes {
			querySQL.WriteString(fmt.Sprintf(`  json_extract(note.attributes, "$.%s") = "%s" `, name, value))
		}
		querySQL.WriteString(") ")
	}
	if query.Path != "" {
		querySQL.WriteString(fmt.Sprintf("AND note.relative_path LIKE '%s' ", query.Path+"%"))
	}
	if len(query.Terms) > 0 {
		querySQL.WriteString(fmt.Sprintf("AND note_fts MATCH '%s' ", strings.Join(query.Terms, " AND ")))
	}

	querySQL.WriteString("ORDER BY rank LIMIT 10;")
	CurrentLogger().Debug(querySQL.String())
	queryFTS, err := CurrentDB().Client().Prepare(querySQL.String())
	if err != nil {
		return nil, err
	}
	res, err := queryFTS.Query()
	if err != nil {
		log.Fatal(err)
	}
	defer res.Close()
	var ids []string
	for res.Next() {
		var id int
		res.Scan(&id)
		ids = append(ids, fmt.Sprint(id))
	}
	if len(ids) == 0 {
		return nil, nil
	}

	return QueryNotes(CurrentDB().Client(), "WHERE rowid IN ("+strings.Join(ids, ",")+")")
}

/* SQL Helpers */

func QueryNote(db SQLClient, whereClause string, args ...any) (*Note, error) {
	var n Note
	var createdAt string
	var updatedAt string
	var lastCheckedAt string
	var tagsRaw string
	var attributesRaw string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			note_oid,
			kind,
			relative_path,
			wikilink,
			title,
			long_title,
			short_title,
			attributes,
			tags,
			"line",
			content_raw,
			hashsum,
			title_markdown,
			title_html,
			title_text,
			content_markdown,
			content_html,
			content_text,
			comment_markdown,
			comment_html,
			comment_text,
			created_at,
			updated_at,
			last_checked_at
		FROM note
		%s;`, whereClause), args...).
		Scan(
			&n.OID,
			&n.FileOID,
			&n.ParentNoteOID,
			&n.NoteKind,
			&n.RelativePath,
			&n.Wikilink,
			&n.Title,
			&n.LongTitle,
			&n.ShortTitle,
			&attributesRaw,
			&tagsRaw,
			&n.Line,
			&n.ContentRaw,
			&n.Hash,
			&n.TitleMarkdown,
			&n.TitleHTML,
			&n.TitleText,
			&n.ContentMarkdown,
			&n.ContentHTML,
			&n.ContentText,
			&n.CommentMarkdown,
			&n.CommentHTML,
			&n.CommentText,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	attributes, err := UnmarshalAttributes(attributesRaw)
	if err != nil {
		return nil, err
	}

	n.Attributes = attributes
	n.Tags = strings.Split(tagsRaw, ",")
	n.CreatedAt = timeFromSQL(createdAt)
	n.UpdatedAt = timeFromSQL(updatedAt)
	n.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &n, nil
}

func QueryNotes(db SQLClient, whereClause string, args ...any) ([]*Note, error) {
	var notes []*Note

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			note_oid,
			kind,
			relative_path,
			wikilink,
			title,
			long_title,
			short_title,
			attributes,
			tags,
			"line",
			content_raw,
			hashsum,
			title_markdown,
			title_html,
			title_text,
			content_markdown,
			content_html,
			content_text,
			comment_markdown,
			comment_html,
			comment_text,
			created_at,
			updated_at,
			last_checked_at
		FROM note
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var n Note
		var createdAt string
		var updatedAt string
		var lastCheckedAt string
		var tagsRaw string
		var attributesRaw string

		err = rows.Scan(
			&n.OID,
			&n.FileOID,
			&n.ParentNoteOID,
			&n.NoteKind,
			&n.RelativePath,
			&n.Wikilink,
			&n.Title,
			&n.LongTitle,
			&n.ShortTitle,
			&attributesRaw,
			&tagsRaw,
			&n.Line,
			&n.ContentRaw,
			&n.Hash,
			&n.TitleMarkdown,
			&n.TitleHTML,
			&n.TitleText,
			&n.ContentMarkdown,
			&n.ContentHTML,
			&n.ContentText,
			&n.CommentMarkdown,
			&n.CommentHTML,
			&n.CommentText,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		)
		if err != nil {
			return nil, err
		}

		attributes, err := UnmarshalAttributes(attributesRaw)
		if err != nil {
			return nil, err
		}

		n.Attributes = attributes
		n.Tags = strings.Split(tagsRaw, ",")
		n.CreatedAt = timeFromSQL(createdAt)
		n.UpdatedAt = timeFromSQL(updatedAt)
		n.LastCheckedAt = timeFromSQL(lastCheckedAt)
		notes = append(notes, &n)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return notes, err
}

/* Format */

func (n *Note) FormatToJSON() string {
	type NoteRepresentation struct {
		OID                string                 `json:"oid"`
		RelativePath       string                 `json:"relativePath"`
		Wikilink           string                 `json:"wikilink"`
		Attributes         map[string]interface{} `json:"attributes"`
		Tags               []string               `json:"tags"`
		ShortTitleRaw      string                 `json:"shortTitleRaw"`
		ShortTitleMarkdown string                 `json:"shortTitleMarkdown"`
		ShortTitleHTML     string                 `json:"shortTitleHTML"`
		ShortTitleText     string                 `json:"shortTitleText"`
		ContentRaw         string                 `json:"contentRaw"`
		ContentMarkdown    string                 `json:"contentMarkdown"`
		ContentHTML        string                 `json:"contentHTML"`
		ContentText        string                 `json:"contentText"`
		CreatedAt          time.Time              `json:"createdAt"`
		UpdatedAt          time.Time              `json:"updatedAt"`
		DeletedAt          *time.Time             `json:"deletedAt"`
	}
	repr := NoteRepresentation{
		OID:                n.OID,
		RelativePath:       n.RelativePath,
		Wikilink:           n.Wikilink,
		ShortTitleRaw:      n.ShortTitle,
		ShortTitleMarkdown: markdown.ToMarkdown(n.ShortTitle),
		ShortTitleHTML:     markdown.ToHTML(n.ShortTitle),
		ShortTitleText:     markdown.ToText(n.ShortTitle),
		Attributes:         n.GetAttributes(),
		Tags:               n.GetTags(),
		ContentRaw:         n.ContentRaw,
		ContentMarkdown:    n.ContentMarkdown,
		ContentHTML:        n.ContentHTML,
		ContentText:        n.ContentText,
		CreatedAt:          n.CreatedAt,
		UpdatedAt:          n.UpdatedAt,
	}
	if !n.DeletedAt.IsZero() {
		repr.DeletedAt = &n.DeletedAt
	}
	output, _ := json.MarshalIndent(repr, "", " ")
	return string(output)
}

func (n *Note) FormatToYAML() string {
	b := new(strings.Builder)
	n.Write(b)
	return b.String()
}

func (n *Note) FormatToMarkdown() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", n.Title))
	sb.WriteRune('\n')
	sb.WriteString(n.ContentMarkdown)
	return sb.String()
}

func (n *Note) FormatToHTML() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<h1>%s</h1>\n", markdown.ToHTML(n.Title)))
	sb.WriteRune('\n')
	sb.WriteString(n.ContentHTML)
	return sb.String()
}

func (n *Note) FormatToText() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s\n", markdown.ToText(n.Title)))
	sb.WriteRune('\n')
	sb.WriteString(n.ContentText)
	return sb.String()
}

// FormatLongTitle formats the long title of a note.
func FormatLongTitle(titles ...string) string {
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

	for i := len(titles) - 1; i >= 0; i-- {
		title := titles[i]

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

	return longTitle
}
