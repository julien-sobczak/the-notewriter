package core

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"reflect"
	"regexp"
	"regexp/syntax"
	"strings"
	"time"
	"unicode/utf8"

	"slices"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"gopkg.in/yaml.v3"
)

// NoteLongTitleSeparator represents the separator when determine the long title of a note.
const NoteLongTitleSeparator string = " / "

type NoteKind string

const (
	KindReference  NoteKind = "reference"
	KindNote       NoteKind = "note"
	KindFlashcard  NoteKind = "flashcard"
	KindCheatsheet NoteKind = "cheatsheet"
	KindQuote      NoteKind = "quote"
	KindJournal    NoteKind = "journal"
	KindTodo       NoteKind = "todo"
	KindArtwork    NoteKind = "artwork"
	KindSnippet    NoteKind = "snippet"
	KindGenerator  NoteKind = "generator"
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

// FIXME add json field tag + sql field tag
type Note struct {
	// A unique identifier among all files
	OID oid.OID `yaml:"oid" json:"oid"`
	// A unique human-friendly slug
	Slug string `yaml:"slug" json:"slug"`

	// Pack file where this object belongs
	PackFileOID oid.OID `yaml:"packfile_oid" json:"packfile_oid"`

	// File containing the note
	FileOID oid.OID `yaml:"file_oid" json:"file_oid"`

	// Type of note
	NoteKind NoteKind `yaml:"kind" json:"kind"`

	// Original title of the note without leading # characters
	Title markdown.Document `yaml:"title" json:"title"`
	// Long title of the note without the kind prefix but prefixed by parent note's short titles
	LongTitle markdown.Document `yaml:"long_title" json:"long_title"`
	// Short title of the note without the kind prefix
	ShortTitle markdown.Document `yaml:"short_title" json:"short_title"`

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path" json:"relative_path"`
	// The full wikilink to this note (without the extension)
	Wikilink string `yaml:"wikilink" json:"wikilink"`

	// Merged attributes
	Attributes AttributeSet `yaml:"attributes,omitempty" json:"attributes,omitempty"`

	// Merged tags
	Tags TagSet `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Line number (1-based index) of the note section title
	Line int `yaml:"line" json:"line"`

	// Content
	Content markdown.Document `yaml:"content" json:"content"`
	Hash    string            `yaml:"content_hash" json:"content_hash"`
	Body    markdown.Document `yaml:"body" json:"body"`
	Comment markdown.Document `yaml:"comment,omitempty" json:"comment,omitempty"`

	// Timestamps to track changes
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
	IndexedAt time.Time `yaml:"indexed_at,omitempty" json:"indexed_at,omitempty"`
}

// NewNote creates a new note.
func NewNote(packFile *PackFile, file *File, parsedNote *ParsedNote) (*Note, error) {
	// Set basic properties
	n := &Note{
		OID:          oid.New(),
		Slug:         parsedNote.Slug,
		PackFileOID:  packFile.OID,
		FileOID:      file.OID,
		Title:        parsedNote.Title,
		ShortTitle:   parsedNote.ShortTitle,
		NoteKind:     parsedNote.Kind,
		RelativePath: file.RelativePath,
		Attributes:   parsedNote.Attributes,
		Tags:         parsedNote.Attributes.Tags(),
		Wikilink:     file.Wikilink + "#" + string(parsedNote.Title.TrimSpace()),
		Content:      parsedNote.Content,
		Hash:         parsedNote.Content.Hash(),
		Body:         parsedNote.Body,
		Comment:      parsedNote.Comment,
		Line:         parsedNote.Line,
		CreatedAt:    packFile.CTime,
		UpdatedAt:    packFile.CTime,
		IndexedAt:    packFile.CTime,
	}

	return n, nil
}

// NewOrExistingNote loads and updates an existing note or creates a new one if new.
func NewOrExistingNote(packFile *PackFile, f *File, parsedNote *ParsedNote) (*Note, error) {
	// Try to find an existing note (instead of recreating it from scratch after every change)
	existingNote, err := CurrentRepository().FindMatchingNote(parsedNote)
	if err != nil {
		return nil, err
	}
	if existingNote != nil {
		existingNote.update(packFile, f, parsedNote)
		return existingNote, nil
	}
	return NewNote(packFile, f, parsedNote)
}

/* Object */

func (n *Note) FileRelativePath() string {
	return n.RelativePath
}

func (n *Note) Kind() string {
	return "note"
}

func (n *Note) UniqueOID() oid.OID {
	return n.OID
}

func (n *Note) ModificationTime() time.Time {
	return n.UpdatedAt
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

func (n *Note) Relations() []*Relation {
	var relations []*Relation

	// Utility function to append wikilink to the returned relations
	addWikilink := func(wikilinkTxt string, relationType string) {
		wikilink, err := markdown.NewWikilink(wikilinkTxt)
		if err != nil {
			// Ignore malformed links
			return
		}

		if wikilink.Section() != "" {
			note, _ := CurrentRepository().FindNoteByWikilink(wikilink.Link)
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
			file, _ := CurrentRepository().FindFileByWikilink(wikilink.Link)
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
	matches := reEmbeddedNote.FindAllStringSubmatch(string(n.Content), -1)
	for _, match := range matches {
		wikilink := match[1]
		addWikilink(wikilink, "includes")
	}

	// Check attribute "source"
	if n.HasAttribute("source") {
		source := n.GetAttribute("source").(string) // Enforced by linter
		if markdown.MatchWikilink(source) {
			addWikilink(source, "references")
		}
	}

	// Check attribute "references"
	if n.HasAttribute("references") {
		references := n.GetAttribute("references").([]interface{}) // Enforced by linter
		for _, referenceRaw := range references {
			if reference, ok := referenceRaw.(string); ok {
				if markdown.MatchWikilink(reference) {
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
				if markdown.MatchWikilink(inspiration) {
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

func (n *Note) update(packFile *PackFile, f *File, parsedNote *ParsedNote) {
	stale := false

	// Set basic properties
	if n.FileOID != f.OID {
		n.FileOID = f.OID
		n.RelativePath = f.RelativePath
		stale = true
	}

	if n.Title != parsedNote.Title {
		n.Title = parsedNote.Title
		n.ShortTitle = parsedNote.ShortTitle
		n.NoteKind = parsedNote.Kind
		stale = true
	}
	if n.Body != parsedNote.Body {
		n.Body = parsedNote.Body
		stale = true
	}
	if n.Comment != parsedNote.Comment {
		n.Comment = parsedNote.Comment
		stale = true
	}

	newWikilink := f.Wikilink + "#" + string(parsedNote.Title.TrimSpace())
	if n.Wikilink != newWikilink {
		n.Wikilink = newWikilink
		stale = true
	}

	newLine := f.AbsoluteBodyLine(parsedNote.Line)
	if n.Line != newLine {
		n.Line = newLine
		stale = true
	}

	if !reflect.DeepEqual(n.Attributes, parsedNote.Attributes) {
		n.Attributes = parsedNote.Attributes
		stale = true
	}

	if n.Content != parsedNote.Content {
		n.Content = parsedNote.Content
		n.Hash = parsedNote.Content.Hash()
		stale = true
	}

	if n.Slug != parsedNote.Slug {
		n.Hash = parsedNote.Slug
		stale = true
	}

	n.PackFileOID = packFile.OID
	n.IndexedAt = packFile.CTime

	if stale {
		n.UpdatedAt = packFile.CTime
	}
}

/* Database Management */

// ReplaceMediasByOIDLinks replaces all non-dangling links by a OID fake link.
func (n *Note) ReplaceMediasByOIDLinks(md string) string {
	regexMedias := regexp.MustCompile(`!\[.*?\]\((\S*?)(?:\s+"(.*?)")?\)`)

	var result strings.Builder
	prevIndex := 0
	matches := regexMedias.FindAllStringSubmatchIndex(md, -1)
	for _, match := range matches {
		result.WriteString(md[prevIndex:match[2]])

		link := md[match[2]:match[3]]
		relativePath, err := CurrentRepository().GetNoteRelativePath(n.GetFile().RelativePath, link)
		if err != nil {
			// Use a 404 image
			result.WriteString("oid:" + oid.Missing)
			prevIndex = match[3]
			continue
		}

		media, err := CurrentRepository().FindMediaByRelativePath(relativePath)
		if err != nil || media == nil {
			// Use a 404 image
			result.WriteString("oid:" + oid.Missing)
			prevIndex = match[3]
			continue
		}

		if media.Dangling {
			// Use a 404 image
			result.WriteString("oid:" + oid.Missing)
			prevIndex = match[3]
			continue
		}

		result.WriteString(fmt.Sprintf("oid:%s", media.OID))
		prevIndex = match[3]
	}
	// Add remaining text
	result.WriteString(md[prevIndex:])

	return result.String()
}

// GetFile returns the containing file, loading it from database if necessary.
func (n *Note) GetFile() *File {
	if n.FileOID == "" {
		return nil
	}
	file, err := CurrentRepository().LoadFileByOID(n.FileOID)
	if err != nil {
		log.Fatalf("Unable to find file %q: %v", n.FileOID, err)
	}
	return file
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
	if n.Attributes == nil {
		n.Attributes = make(map[string]interface{})
	}
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
	return false, "", text
}

/* State Management */

func (n *Note) Save() error {
	CurrentLogger().Debugf("Saving note %s...", n.Wikilink)
	query := `
		INSERT INTO note(
			oid,
			packfile_oid,
			file_oid,
			slug,
			kind,
			relative_path,
			wikilink,
			title,
			long_title,
			short_title,
			attributes,
			tags,
			"line",
			content,
			hashsum,
			body,
			comment,
			created_at,
			updated_at,
			indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			file_oid = ?,
			slug = ?,
			kind = ?,
			relative_path = ?,
			wikilink = ?,
			title = ?,
			long_title = ?,
			short_title = ?,
			attributes = ?,
			tags = ?,
			"line" = ?,
			content = ?,
			hashsum = ?,
			body = ?,
			comment = ?,
			updated_at = ?,
			indexed_at = ?
		;
	`

	attributesJSON, err := n.Attributes.ToJSON()
	if err != nil {
		return err
	}

	_, err = CurrentDB().Client().Exec(query,
		// Insert
		n.OID,
		n.PackFileOID,
		n.FileOID,
		n.Slug,
		n.NoteKind,
		n.RelativePath,
		n.Wikilink,
		n.Title,
		n.LongTitle,
		n.ShortTitle,
		attributesJSON,
		strings.Join(n.Tags, ","),
		n.Line,
		n.Content,
		n.Hash,
		n.Body,
		n.Comment,
		timeToSQL(n.CreatedAt),
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.IndexedAt),
		// Update
		n.PackFileOID,
		n.FileOID,
		n.Slug,
		n.NoteKind,
		n.RelativePath,
		n.Wikilink,
		n.Title,
		n.LongTitle,
		n.ShortTitle,
		attributesJSON,
		strings.Join(n.Tags, ","),
		n.Line,
		n.Content,
		n.Hash,
		n.Body,
		n.Comment,
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.IndexedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *Note) Delete() error {
	CurrentLogger().Debugf("Deleting note %s...", n.Wikilink)
	query := `DELETE FROM note WHERE oid = ? AND packfile_oid = ?;`
	_, err := CurrentDB().Client().Exec(query, n.OID, n.PackFileOID)
	return err
}

// CountNotes returns the total number of notes.
func (r *Repository) CountNotes() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM note`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// CountNotesByKind returns the total number of notes for every kind.
func (r *Repository) CountNotesByKind() (map[NoteKind]int, error) {
	res := map[NoteKind]int{
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
func (r *Repository) CountTags() (map[string]int, error) {
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
func (r *Repository) CountAttributes() (map[string]int, error) {
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

func (r *Repository) DumpNotes() error {
	notes, err := QueryNotes(CurrentDB().Client(), "")
	if err != nil {
		return err
	}
	for _, note := range notes {
		CurrentLogger().Infof("Note %s [%s] [[%s]]\n", note.LongTitle, note.OID, note.Wikilink)
	}
	return nil
}

func (r *Repository) LoadNoteByOID(oid oid.OID) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE oid = ?`, oid)
}

func (r *Repository) FindNotesByFileOID(oid oid.OID) ([]*Note, error) {
	return QueryNotes(CurrentDB().Client(), `WHERE file_oid = ?`, oid)
}

func (r *Repository) FindNoteByTitle(title string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE title = ?`, title)
}

func (r *Repository) FindNoteBySlug(slug string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE slug = ?`, slug)
}

func (r *Repository) FindNoteByHash(hash string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE hashsum = ?`, hash)
}

func (r *Repository) FindNoteByPathAndTitle(relativePath string, title string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE relative_path = ? AND title = ?`, relativePath, title)
}

func (r *Repository) FindMatchingNote(parsedNote *ParsedNote) (*Note, error) {
	// Try by slug
	note, _ := r.FindNoteBySlug(parsedNote.Slug)
	if note != nil {
		return note, nil
	}

	// Try by wikilink
	note, _ = r.FindNoteByWikilink(parsedNote.RelativePath + "#" + string(parsedNote.Title)) // FIXME trim extension?
	if note != nil {
		return note, nil
	}

	// Last by same title or same content in the same file
	return QueryNote(CurrentDB().Client(), `WHERE relative_path = ? AND (title = ? OR hashsum = ?)`, parsedNote.RelativePath, parsedNote.Title, parsedNote.Hash())
}

func (r *Repository) FindNoteByWikilink(wikilink string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE wikilink LIKE ?`, "%"+wikilink)
}

func (r *Repository) FindNotesByWikilink(wikilink string) ([]*Note, error) {
	return QueryNotes(CurrentDB().Client(), `WHERE wikilink LIKE ?`, "%"+wikilink)
}

func (r *Repository) FindNotesLastCheckedBefore(point time.Time, path string) ([]*Note, error) {
	if path == "." {
		path = ""
	}
	return QueryNotes(CurrentDB().Client(), `WHERE indexed_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

// SearchNotes query notes to find the ones matching a list of criteria.
//
// Examples:
//
//	tag:favorite kind:reference kind:note path:projects/
func (r *Repository) SearchNotes(q string) ([]*Note, error) {
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
	if query.Slug != "" {
		querySQL.WriteString(fmt.Sprintf("AND note.slug = '%s' ", query.Slug))
	}
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
	var lastIndexedAt string
	var tagsRaw string
	var attributesRaw string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			file_oid,
			slug,
			kind,
			relative_path,
			wikilink,
			title,
			long_title,
			short_title,
			attributes,
			tags,
			"line",
			content,
			hashsum,
			body,
			comment,
			created_at,
			updated_at,
			indexed_at
		FROM note
		%s;`, whereClause), args...).
		Scan(
			&n.OID,
			&n.PackFileOID,
			&n.FileOID,
			&n.Slug,
			&n.NoteKind,
			&n.RelativePath,
			&n.Wikilink,
			&n.Title,
			&n.LongTitle,
			&n.ShortTitle,
			&attributesRaw,
			&tagsRaw,
			&n.Line,
			&n.Content,
			&n.Hash,
			&n.Body,
			&n.Comment,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	attributes, err := NewAttributeSetFromYAML(attributesRaw)
	if err != nil {
		return nil, err
	}

	n.Attributes = attributes.CastOrIgnore(GetSchemaAttributeTypes())
	n.Tags = strings.Split(tagsRaw, ",")
	n.CreatedAt = timeFromSQL(createdAt)
	n.UpdatedAt = timeFromSQL(updatedAt)
	n.IndexedAt = timeFromSQL(lastIndexedAt)

	return &n, nil
}

func QueryNotes(db SQLClient, whereClause string, args ...any) ([]*Note, error) {
	var notes []*Note

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			file_oid,
			slug,
			kind,
			relative_path,
			wikilink,
			title,
			long_title,
			short_title,
			attributes,
			tags,
			"line",
			content,
			hashsum,
			body,
			comment,
			created_at,
			updated_at,
			indexed_at
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
		var lastIndexedAt string
		var tagsRaw string
		var attributesRaw string

		err = rows.Scan(
			&n.OID,
			&n.PackFileOID,
			&n.FileOID,
			&n.Slug,
			&n.NoteKind,
			&n.RelativePath,
			&n.Wikilink,
			&n.Title,
			&n.LongTitle,
			&n.ShortTitle,
			&attributesRaw,
			&tagsRaw,
			&n.Line,
			&n.Content,
			&n.Hash,
			&n.Body,
			&n.Comment,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
		)
		if err != nil {
			return nil, err
		}

		attributes, err := NewAttributeSetFromYAML(attributesRaw)
		if err != nil {
			return nil, err
		}

		n.Attributes = attributes.CastOrIgnore(GetSchemaAttributeTypes())
		n.Tags = strings.Split(tagsRaw, ",")
		n.CreatedAt = timeFromSQL(createdAt)
		n.UpdatedAt = timeFromSQL(updatedAt)
		n.IndexedAt = timeFromSQL(lastIndexedAt)
		notes = append(notes, &n)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return notes, err
}

/* Dumpable */

func (n *Note) ToYAML() string {
	return ToBeautifulYAML(n)
}

func (n *Note) ToJSON() string {
	return ToBeautifulJSON(n)
}

func (n *Note) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", n.Title))
	sb.WriteRune('\n')
	sb.WriteString(string(n.Body))
	return sb.String()
}

// FormatLongTitle formats the long title of a note.
func FormatLongTitle(titles ...markdown.Document) markdown.Document {
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
		title := string(titles[i])

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
