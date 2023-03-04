package core

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
	"github.com/julien-sobczak/the-notetaker/pkg/text"
	"gopkg.in/yaml.v3"
)

type NoteKind int

const (
	KindFree       NoteKind = 0
	KindReference  NoteKind = 1
	KindNote       NoteKind = 2
	KindFlashcard  NoteKind = 3
	KindCheatsheet NoteKind = 4
	KindQuote      NoteKind = 5
	KindJournal    NoteKind = 6
	KindTodo       NoteKind = 7
	KindArtwork    NoteKind = 8
	KindSnippet    NoteKind = 9
)

var regexReference = regexp.MustCompile(`(?i)^Reference[-:_ ]\s*(.*)$`)
var regexNote = regexp.MustCompile(`^(?i)Note[-:_ ]\s*(.*)$`)
var regexFlashcard = regexp.MustCompile(`^(?i)Flashcard[-:_ ]\s*(.*)$`)
var regexCheatsheet = regexp.MustCompile(`^(?i)Cheatsheet[-:_ ]\s*(.*)$`)
var regexQuote = regexp.MustCompile(`^(?i)Quote[-:_ ]\s*(.*)$`)
var regexTodo = regexp.MustCompile(`^(?i)Todo[-:_ ]\s*(.*)$`)
var regexArtwork = regexp.MustCompile(`^(?i)Artwork[-:_ ]\s*(.*)$`)
var regexSnippet = regexp.MustCompile(`^(?i)Snippet[-:_ ]\s*(.*)$`)

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
	// Short title of the note without the kind prefix
	ShortTitle string `yaml:"short_title"`

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path"`
	// The full wikilink to this note (without the extension)
	Wikilink string `yaml:"wikilink"`

	// Note-specific attributes. Use GetAttributes() to get all merged attributes
	Attributes map[string]interface{} `yaml:"attributes"`

	// Note-specific tags. Use GetTags() to get all merged tags
	Tags []string `yaml:"tags,omitempty"`

	// Line number (1-based index) of the note section title
	Line int `yaml:"line"`

	// Content in various formats (best for editing, rendering, writing, etc.)
	ContentRaw      string `yaml:"content_raw"`
	Hash            string `yaml:"content_hash"`
	ContentMarkdown string `yaml:"content_markdown"`
	ContentHTML     string `yaml:"content_html"`
	ContentText     string `yaml:"content_text"`

	// Timestamps to track changes
	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	DeletedAt     time.Time `yaml:"-"`
	LastCheckedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}

func NewOrExistingNote(f *File, title string, content string, lineNumber int) *Note {
	content = strings.TrimSpace(content)

	note, _ := FindNoteByWikilink(f.RelativePath + "#" + title)
	if note != nil {
		note.Update(f, title, content, lineNumber)
		return note
	}

	hash := hash([]byte(content))
	note, _ = FindMatchingNotes(title, hash)
	if note != nil {
		note.Update(f, title, content, lineNumber)
		return note
	}

	return NewNote(f, title, content, lineNumber)
}

func NewNote(f *File, title string, content string, lineNumber int) *Note {
	rawContent := strings.TrimSpace(content)

	_, kind, shortTitle := isSupportedNote(title)

	n := &Note{
		OID:           NewOID(),
		FileOID:       f.OID,
		File:          f,
		ParentNoteOID: "",
		ParentNote:    nil,
		NoteKind:      kind,
		Title:         title,
		ShortTitle:    shortTitle,
		RelativePath:  f.RelativePath,
		Wikilink:      f.Wikilink + "#" + strings.TrimSpace(title),
		Line:          lineNumber,
		CreatedAt:     clock.Now(),
		UpdatedAt:     clock.Now(),
		new:           true,
		stale:         true,
	}

	n.updateContent(rawContent)

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
	}
	for _, object := range n.GetMedias() {
		objs = append(objs, object)
	}
	for _, object := range n.GetReminders() {
		objs = append(objs, object)
	}
	return objs
}

func (n *Note) Blobs() []BlobRef {
	// Use Media.Blobs() instead
	return nil
}

func (n Note) String() string {
	return fmt.Sprintf("note %q [%s]", n.Title, n.OID)
}

/* Update */

func (n *Note) Update(f *File, title string, content string, lineNumber int) {
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

func (n *Note) parseContentRaw() (string, []string, map[string]interface{}) {
	var tags []string
	var attributes map[string]interface{}
	var content string

	content, tags = extractTags(n.ContentRaw)
	content, attributes = extractAttributes(content)
	content = strings.TrimSpace(content)
	content = n.expandSyntaxSugar(content)

	return content, tags, attributes
}

func (n *Note) updateContent(rawContent string) {
	n.ContentRaw = strings.TrimSpace(rawContent)
	n.Hash = hash([]byte(n.ContentRaw))

	content, tags, attributes := n.parseContentRaw()

	n.Tags = tags
	n.Attributes = attributes
	n.ContentMarkdown = markdown.ToMarkdown(content)
	n.ContentHTML = markdown.ToHTML(n.ContentMarkdown) // Use processed md to use <h2>, <h3>, ... whatever the note level
	n.ContentText = markdown.ToText(n.ContentMarkdown)
}

func (n *Note) SetParent(parent *Note) {
	if parent == nil {
		return
	}
	n.ParentNote = parent
	n.ParentNoteOID = parent.OID
}

// GetFile returns the containing file, loading it from database if necessary.
func (n *Note) GetFile() *File {
	if n.FileOID == "" {
		return nil
	}
	if n.File == nil {
		file, err := LoadFileByOID(n.FileOID)
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
		note, err := LoadNoteByOID(n.ParentNoteOID)
		if err != nil {
			log.Fatalf("Unable to note file %q: %v", n.ParentNoteOID, err)
		}
		n.ParentNote = note
	}
	return n.ParentNote
}

func (n *Note) GetAttributes() map[string]interface{} {
	if n.ParentNoteOID == "" {
		return mergeAttributes(n.GetFile().GetAttributes(), n.Attributes)

	}
	return mergeAttributes(n.GetParentNote().GetAttributes(), n.Attributes)
}

func (n *Note) GetAttribute(name string) interface{} {
	if value, ok := n.GetAttributes()[name]; ok {
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
	if n.ParentNoteOID == "" {
		return mergeTags(n.GetFile().GetTags(), n.Tags)
	}
	return mergeTags(n.GetParentNote().GetTags(), n.Tags)
}

func mergeTags(tags ...[]string) []string {
	var result []string
	for _, items := range tags {
		for _, item := range items {
			found := false
			for _, existingItem := range result {
				if existingItem == item {
					found = true
					break
				}
			}
			if !found {
				result = append(result, item)
			}
		}
	}
	return result
}

func mergeAttributes(attributes ...map[string]interface{}) map[string]interface{} {
	// Implementation: THe code is obscure due to untyped elements.
	// We don't want to always replace old values when the old value is a slice
	// that can accept these new values too.
	//
	// Examples:
	//   ---
	//   tags: [a]
	//   references: []
	//   ---
	//
	//   `#b`
	//   <!-- references: https://example.org -->
	//
	// Should be the same as:
	//   ---
	//   tags: [a, b]
	//   references: [https://example.org]
	//   ---
	//
	// Most of the code tries to manage this use case.

	result := make(map[string]interface{})
	empty := true
	for _, m := range attributes {
		for newKey, newValue := range m {
			// Check if the attribute was already defined
			if currentValue, ok := result[newKey]; ok {

				// If the tyoe is a slice, append the new value instead of overriding
				switch x := currentValue.(type) {
				case []interface{}:
					switch y := newValue.(type) {
					case []interface{}:
						result[newKey] = append(x, y...)
					case []string:
						for _, item := range y {
							result[newKey] = append(x, fmt.Sprintf("%v", item))
						}
					default:
						result[newKey] = append(x, newValue)
					}
				case []string:
					switch y := newValue.(type) {
					case []interface{}:
						for _, item := range y {
							result[newKey] = append(x, fmt.Sprintf("%v", item))
						}
					case []string:
						result[newKey] = append(x, y...)
					default:
						result[newKey] = append(x, fmt.Sprintf("%v", newValue))
					}

				default:
					// override
					result[newKey] = newValue
				}
			} else {
				result[newKey] = newValue
			}
			empty = false
		}
	}
	if empty {
		return nil
	}
	return result
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
	// FIXME what about Journal notes?
	return false, KindFree, ""
}

func (n *Note) expandSyntaxSugar(rawContent string) string {
	if n.NoteKind == KindQuote {
		// Turn every text line into a quote
		// Add the attribute name or author in suffix
		// Ex:
		//   ---
		//   name: Walt Disney
		//   ---
		//   "The way to get started is to quit"
		//   "talking and begin doing."
		//
		// Becomes:
		//
		//   > The way to get started is to quit
		//   > talking and begin doing.
		//   > — Walt Disney

		var res bytes.Buffer
		previousLineWasQuotation := false
		for _, line := range strings.Split(rawContent, "\n") {
			if text.IsBlank(line) {
				if previousLineWasQuotation {
					name := n.GetAttributeString("name", n.GetAttributeString("author", ""))
					if !text.IsBlank(name) {
						res.WriteString("> -- " + name + "\n")
					}
					res.WriteString(line + "\n")
					previousLineWasQuotation = false
				}
				res.WriteString(line + "\n")
			} else {
				res.WriteString("> " + strings.TrimSpace(line) + "\n")
				previousLineWasQuotation = true
			}
		}

		if previousLineWasQuotation {
			name := n.GetAttributeString("name", n.GetAttributeString("author", ""))
			if !text.IsBlank(name) {
				res.WriteString("> -- " + name + "\n")
			}
		}

		return res.String()
	}

	return rawContent
}

// extractTags searches for all tags and remove lines containing only tags.
func extractTags(content string) (string, []string) {
	var tags []string

	reTags := regexp.MustCompile("`#(\\S+)`")
	reOnlyTags := regexp.MustCompile("^(`#\\S+`\\s*)+$")
	var res bytes.Buffer
	for _, line := range strings.Split(content, "\n") {
		matches := reTags.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			tags = append(tags, match[1])
		}
		if !reOnlyTags.MatchString(line) {
			res.WriteString(line + "\n")
		}
	}

	return text.SquashBlankLines(res.String()), tags
}

// removeTags removes all tags from a text.
func removeTags(content string) string {
	reTags := regexp.MustCompile("`#(\\S+)`")
	var res bytes.Buffer
	for _, line := range strings.Split(content, "\n") {
		newLine := reTags.ReplaceAllLiteralString(line, "")
		if !text.IsBlank(newLine) {
			res.WriteString(newLine + "\n")
		}
	}
	return strings.TrimSpace(text.SquashBlankLines(res.String()))
}

func extractAttributes(content string) (string, map[string]interface{}) {
	attributes := make(map[string]interface{})

	reAttribute := regexp.MustCompile(`<!--\s*(\w+)\s*:\s*(.*?)\s*-->\s*$`)

	var res bytes.Buffer
	for _, line := range strings.Split(content, "\n") {
		match := reAttribute.FindStringSubmatch(line)
		if match != nil {
			attributes[match[1]] = match[2]
		} else {
			res.WriteString(line + "\n")
		}
	}
	return text.SquashBlankLines(res.String()), attributes
}

// GetMedias extracts medias from the note.
func (n *Note) GetMedias() []*Media {
	return extractMediasFromMarkdown(n.File.RelativePath, n.ContentRaw)
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
				description = removeTags(submatch[1]) // Remove tags
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

// TODO add SetParent method and traverse the hierachy to merge attributes/tags

func (n *Note) Check() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = n.CheckWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil

}

func (n *Note) CheckWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Checking note %s...", n.Wikilink)
	n.LastCheckedAt = clock.Now()
	query := `
		UPDATE note
		SET last_checked_at = ?
		WHERE oid = ?;`
	if _, err := tx.Exec(query, timeToSQL(n.LastCheckedAt), n.OID); err != nil {
		return err
	}
	query = `
		UPDATE flashcard
		SET last_checked_at = ?
		WHERE note_oid = ?;`
	if _, err := tx.Exec(query, timeToSQL(n.LastCheckedAt), n.OID); err != nil {
		return err
	}
	query = `
		UPDATE link
		SET last_checked_at = ?
		WHERE note_oid = ?;`
	if _, err := tx.Exec(query, timeToSQL(n.LastCheckedAt), n.OID); err != nil {
		return err
	}
	query = `
		UPDATE reminder
		SET last_checked_at = ?
		WHERE note_oid = ?;`
	if _, err := tx.Exec(query, timeToSQL(n.LastCheckedAt), n.OID); err != nil {
		return err
	}

	return nil
}

func (n *Note) Save(tx *sql.Tx) error {
	var err error
	switch n.State() {
	case Added:
		err = n.InsertWithTx(tx)
	case Modified:
		err = n.UpdateWithTx(tx)
	case Deleted:
		err = n.DeleteWithTx(tx)
	default:
		err = n.CheckWithTx(tx)
	}
	n.new = false
	n.stale = false
	return err
}

func (n *Note) OldSave() error { // FIXME remove deprecated
	if !n.stale {
		return n.Check()
	}

	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = n.SaveWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	n.new = false
	n.stale = false

	return nil
}

func (n *Note) SaveWithTx(tx *sql.Tx) error { // FIXME remove deprecated
	if !n.stale {
		return n.CheckWithTx(tx)
	}

	// There is no common interface between sql.DB and sql.Txt
	// See https://github.com/golang/go/issues/14468

	now := clock.Now()
	n.UpdatedAt = now
	n.LastCheckedAt = now

	if !n.new {
		if err := n.UpdateWithTx(tx); err != nil {
			return err
		}
	} else {
		n.CreatedAt = now
		if err := n.InsertWithTx(tx); err != nil {
			return err
		}

		// Update note ID
		links := n.GetLinks()
		for _, link := range links {
			link.NoteOID = n.OID
		}

		// Save reminders
		reminders := n.GetReminders()
		for _, reminder := range reminders {
			reminder.NoteOID = n.OID
		}
	}

	n.new = false
	n.stale = false

	// Save the links
	links := n.GetLinks()
	for _, link := range links {
		if err := link.SaveWithTx(tx); err != nil {
			return err
		}
	}

	// Save reminders
	reminders := n.GetReminders()
	for _, reminder := range reminders {
		if err := reminder.SaveWithTx(tx); err != nil {
			return err
		}
	}

	return nil
}

func (n *Note) InsertWithTx(tx *sql.Tx) error {
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
			short_title,
			attributes_yaml,
			attributes_json,
			tags,
			"line",
			content_raw,
			hashsum,
			content_markdown,
			content_html,
			content_text,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	attributesYAML, err := n.AttributesYAML()
	if err != nil {
		return err
	}
	attributesJSON, err := n.AttributesJSON()
	if err != nil {
		return err
	}

	_, err = tx.Exec(query,
		n.OID,
		n.FileOID,
		n.ParentNoteOID,
		n.NoteKind,
		n.RelativePath,
		n.Wikilink,
		n.Title,
		n.ShortTitle,
		attributesYAML,
		attributesJSON,
		strings.Join(n.Tags, ","),
		n.Line,
		n.ContentRaw,
		n.Hash,
		n.ContentMarkdown,
		n.ContentHTML,
		n.ContentText,
		timeToSQL(n.CreatedAt),
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.DeletedAt),
		timeToSQL(n.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *Note) UpdateWithTx(tx *sql.Tx) error {
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
			short_title = ?,
			attributes_yaml = ?,
			attributes_json = ?,
			tags = ?,
			"line" = ?,
			content_raw = ?,
			hashsum = ?,
			content_markdown = ?,
			content_html = ?,
			content_text = ?,
			updated_at = ?,
			deleted_at = ?,
			last_checked_at = ?
		WHERE oid = ?;
	`

	attributesYAML, err := n.AttributesYAML()
	if err != nil {
		return err
	}
	attributesJSON, err := n.AttributesJSON()
	if err != nil {
		return err
	}

	_, err = tx.Exec(query,
		n.FileOID,
		n.ParentNoteOID,
		n.NoteKind,
		n.RelativePath,
		n.Wikilink,
		n.Title,
		n.ShortTitle,
		attributesYAML,
		attributesJSON,
		strings.Join(n.Tags, ","),
		n.Line,
		n.ContentRaw,
		n.Hash,
		n.ContentMarkdown,
		n.ContentHTML,
		n.ContentText,
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.DeletedAt),
		timeToSQL(n.LastCheckedAt),
		n.OID,
	)

	return err
}

func (n *Note) Delete() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = n.DeleteWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (n *Note) DeleteWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Deleting note %s...", n.Wikilink)
	query := `DELETE FROM note WHERE oid = ?;`
	_, err := tx.Exec(query, n.OID)
	return err
}

func (n *Note) AttributesJSON() (string, error) {
	var buf bytes.Buffer
	bufEncoder := json.NewEncoder(&buf)
	err := bufEncoder.Encode(n.Attributes)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (n *Note) AttributesYAML() (string, error) {
	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(Indent)
	err := bufEncoder.Encode(n.Attributes)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CountNotes returns the total number of notes.
func CountNotes() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM note WHERE deleted_at = ''`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func LoadNoteByOID(oid string) (*Note, error) {
	return QueryNote(`WHERE oid = ?`, oid)
}

func FindNoteByTitle(title string) (*Note, error) {
	return QueryNote(`WHERE title = ?`, title)
}

func FindNoteByHash(hash string) (*Note, error) {
	return QueryNote(`WHERE hashsum = ?`, hash)
}

func FindMatchingNotes(title, hash string) (*Note, error) {
	return QueryNote(`WHERE title = ? OR hashsum = ?`, title, hash)
}

func FindNoteByWikilink(wikilink string) (*Note, error) {
	return QueryNote(`WHERE wikilink LIKE ?`, "%"+wikilink)
}

func FindNotesByWikilink(wikilink string) ([]*Note, error) {
	return QueryNotes(`WHERE wikilink LIKE ?`, "%"+wikilink)
}

func FindNotesLastCheckedBefore(point time.Time, path string) ([]*Note, error) {
	return QueryNotes(`WHERE last_checked_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

func SearchNotes(kind NoteKind, q string) ([]*Note, error) {
	db := CurrentDB().Client()
	queryFTS, err := db.Prepare("SELECT rowid FROM note_fts WHERE kind = ? and note_fts MATCH ? ORDER BY rank LIMIT 10;")
	if err != nil {
		return nil, err
	}
	res, err := queryFTS.Query(kind, q)
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

	query := "WHERE rowid IN (" + strings.Join(ids, ",") + ")"
	return QueryNotes(query)
}

/* SQL Helpers */

func QueryNote(whereClause string, args ...any) (*Note, error) {
	db := CurrentDB().Client()

	var n Note
	var createdAt string
	var updatedAt string
	var deletedAt string
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
			short_title,
			attributes_yaml,
			tags,
			"line",
			content_raw,
			hashsum,
			content_markdown,
			content_html,
			content_text,
			created_at,
			updated_at,
			deleted_at,
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
			&n.ShortTitle,
			&attributesRaw,
			&tagsRaw,
			&n.Line,
			&n.ContentRaw,
			&n.Hash,
			&n.ContentMarkdown,
			&n.ContentHTML,
			&n.ContentText,
			&createdAt,
			&updatedAt,
			&deletedAt,
			&lastCheckedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var attributes map[string]interface{}
	err := yaml.Unmarshal([]byte(attributesRaw), &attributes)
	if err != nil {
		return nil, err
	}

	n.Attributes = attributes
	n.Tags = strings.Split(tagsRaw, ",")
	n.CreatedAt = timeFromSQL(createdAt)
	n.UpdatedAt = timeFromSQL(updatedAt)
	n.DeletedAt = timeFromSQL(deletedAt)
	n.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &n, nil
}

func QueryNotes(whereClause string, args ...any) ([]*Note, error) {
	db := CurrentDB().Client()

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
			short_title,
			attributes_yaml,
			tags,
			"line",
			content_raw,
			hashsum,
			content_markdown,
			content_html,
			content_text,
			created_at,
			updated_at,
			deleted_at,
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
		var deletedAt string
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
			&n.ShortTitle,
			&attributesRaw,
			&tagsRaw,
			&n.Line,
			&n.ContentRaw,
			&n.Hash,
			&n.ContentMarkdown,
			&n.ContentHTML,
			&n.ContentText,
			&createdAt,
			&updatedAt,
			&deletedAt,
			&lastCheckedAt,
		)
		if err != nil {
			return nil, err
		}

		var attributes map[string]interface{}
		err := yaml.Unmarshal([]byte(attributesRaw), &attributes)
		if err != nil {
			return nil, err
		}

		n.Attributes = attributes
		n.Tags = strings.Split(tagsRaw, ",")
		n.CreatedAt = timeFromSQL(createdAt)
		n.UpdatedAt = timeFromSQL(updatedAt)
		n.DeletedAt = timeFromSQL(deletedAt)
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
		OID             string                 `json:"oid"`
		RelativePath    string                 `json:"relativePath"`
		Wikilink        string                 `json:"wikilink"`
		FrontMatter     map[string]interface{} `json:"frontMatter"`
		Tags            []string               `json:"tags"`
		ContentRaw      string                 `json:"contentRaw"`
		ContentMarkdown string                 `json:"contentMarkdown"`
		ContentHTML     string                 `json:"contentHTML"`
		ContentText     string                 `json:"contentText"`
	}
	repr := NoteRepresentation{
		OID:             n.OID,
		RelativePath:    n.RelativePath,
		Wikilink:        n.Wikilink,
		FrontMatter:     n.GetAttributes(),
		Tags:            n.GetTags(),
		ContentRaw:      n.ContentRaw,
		ContentMarkdown: n.ContentMarkdown,
		ContentHTML:     n.ContentHTML,
		ContentText:     n.ContentText,
	}
	output, _ := json.MarshalIndent(repr, "", " ")
	return string(output)
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
