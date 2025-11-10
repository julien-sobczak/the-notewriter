package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"reflect"
	"regexp"
	"strings"
	"time"

	"slices"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

// NoteLongTitleSeparator represents the separator when determine the long title of a note.
const NoteLongTitleSeparator string = " / "

// ListItem represents a single item in a Markdown list
type ListItem struct {
	Line       int               `yaml:"line" json:"line"`
	Text       markdown.Document `yaml:"text" json:"text"`
	Tags       TagSet            `yaml:"tags,omitempty" json:"tags,omitempty"`
	Attributes AttributeSet      `yaml:"attributes,omitempty" json:"attributes,omitempty"`
	Emojis     EmojiSet          `yaml:"emojis,omitempty" json:"emojis,omitempty"`
	Children   ListItems         `yaml:"children,omitempty" json:"children,omitempty"`
}

type ListItems []*ListItem

// ListItems represents the extracted list items from Markdown content
type Items struct {
	Children   ListItems `yaml:"children" json:"children"`
	Attributes []string  `yaml:"attributes,omitempty" json:"attributes,omitempty"` // All unique attribute names
	Tags       TagSet    `yaml:"tags,omitempty" json:"tags,omitempty"`             // All unique tag values
	Emojis     EmojiSet  `yaml:"emojis,omitempty" json:"emojis,omitempty"`         // All unique emojis
}

// AttributeNames returns the names of all attributes in the list item
func (li *ListItem) AttributeNames() []string {
	var names []string
	for name := range li.Attributes {
		names = append(names, name)
		for _, child := range li.Children {
			names = append(names, child.AttributeNames()...)
		}
	}
	return names
}

// UniqueAttribute returns the unique attributes in the list item and its children
func (li ListItems) UniqueAttribute() AttributeSet {
	attributes := make(AttributeSet)
	for _, item := range li {
		for name, value := range item.Attributes {
			if _, exists := attributes[name]; !exists {
				attributes[name] = value
			} else {
				// Non unique...
				delete(attributes, name)
			}
		}
		childAttributes := item.Children.UniqueAttribute()
		for name, value := range childAttributes {
			if _, exists := attributes[name]; !exists {
				attributes[name] = value
			} else {
				// Non unique...
				delete(attributes, name)
			}
		}
	}
	return attributes
}

func NewItems(children ListItems) *Items {
	return &Items{
		Children: children,
		// Collect aggregated attributes/tags/emojis
		Attributes: children.CollectAttributesNames(),
		Tags:       children.CollectTags(),
		Emojis:     children.CollectEmojis(),
	}
}

// CollectAttributes collects all unique attributes from the list items.
func (l ListItems) CollectAttributesNames() []string {
	return CollectStringFromListItems(l, func(li *ListItem) []string {
		return li.AttributeNames()
	})
}

// CollectAttributes collects all unique attributes from the list items.
func (l ListItems) CollectTags() []string {
	return CollectStringFromListItems(l, func(li *ListItem) []string {
		return li.Tags
	})
}

// CollectEmojis collects all unique emojis from the list items.
func (l ListItems) CollectEmojis() []string {
	return CollectStringFromListItems(l, func(li *ListItem) []string {
		return li.Emojis
	})
}

func CollectFromListItem[T comparable](i *ListItem, collectFunc func(*ListItem) []T) map[T]bool {
	values := make(map[T]bool)
	for _, value := range collectFunc(i) {
		values[value] = true
	}
	maps.Copy(values, CollectFromListItems(i.Children, collectFunc))
	return values
}
func CollectFromListItems[T comparable](l ListItems, collectFunc func(*ListItem) []T) map[T]bool {
	values := make(map[T]bool)
	for _, child := range l {
		for value := range CollectFromListItem(child, collectFunc) {
			values[value] = true
		}
	}
	return values
}
func CollectStringFromListItems(l ListItems, collectFunc func(*ListItem) []string) []string {
	values := CollectFromListItems(l, collectFunc)
	return slices.Sorted(maps.Keys(values))
}

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
	Type string `yaml:"type" json:"type"`

	// Original title of the note without leading # characters
	Title markdown.Document `yaml:"title" json:"title"`
	// Long title of the note without the type prefix but prefixed by parent note's short titles
	LongTitle markdown.Document `yaml:"long_title" json:"long_title"`
	// Short title of the note without the type prefix
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

	// List items extracted from Markdown lists
	Items *Items `yaml:"items,omitempty" json:"items,omitempty"`

	// Timestamps to track changes
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
	IndexedAt time.Time `yaml:"indexed_at,omitempty" json:"indexed_at,omitempty"`

	// Operation-related fields (not mapped in object representation but using CDRTs)
	Marked      bool         `yaml:"-" json:"-"`
	MarkedAt    time.Time    `yaml:"-" json:"-"`
	Annotations []Annotation `yaml:"-" json:"-"`
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
		LongTitle:    parsedNote.LongTitle,
		ShortTitle:   parsedNote.ShortTitle,
		Type:         parsedNote.Type,
		RelativePath: file.RelativePath,
		Attributes:   parsedNote.Attributes,
		Tags:         parsedNote.Attributes.Tags(),
		Wikilink:     file.Wikilink + "#" + string(parsedNote.Title.TrimSpace()),
		Content:      parsedNote.Content,
		Hash:         parsedNote.Content.Hash(),
		Body:         parsedNote.Body,
		Comment:      parsedNote.Comment,
		Items:        parsedNote.Items,
		Line:         parsedNote.Line,
		CreatedAt:    packFile.CTime,
		UpdatedAt:    packFile.CTime,
		IndexedAt:    packFile.CTime,
		// Operation-related fields
		Marked: false,
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
	// Remap attributes to expected type
	n.Attributes = n.Attributes.CastOrIgnore(CurrentConfigFile().Attributes)
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

func (n *Note) Links() []*Link {
	var links []*Link

	// Utility function to append wikilink to the returned links
	addWikilink := func(wikilinkTxt string, relationship string) {
		wikilink, err := markdown.NewWikilink(wikilinkTxt)
		if err != nil {
			// Ignore malformed links
			return
		}

		if wikilink.Section() != "" {
			note, _ := CurrentRepository().FindNoteByWikilink(wikilink.Link)
			if note != nil {
				links = append(links, &Link{
					SourceOID:  n.OID,
					SourceKind: "note",
					TargetOID:  note.OID,
					TargetKind: "note",
					Type:       relationship,
				})
			}
		} else {
			file, _ := CurrentRepository().FindFileByWikilink(wikilink.Link)
			if file != nil {
				links = append(links, &Link{
					SourceOID:  n.OID,
					SourceKind: "note",
					TargetOID:  file.OID,
					TargetKind: "file",
					Type:       relationship,
				})
			}
		}
	}

	// Search for embedded notes
	reEmbeddedNote := regexp.MustCompile(`!(\[\[(.*)(?:\|.*)?\]\])\s*`)
	matches := reEmbeddedNote.FindAllStringSubmatch(string(n.Body), -1)
	for _, match := range matches {
		wikilink := match[1]
		addWikilink(wikilink, "embeds")
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
		references := n.GetAttribute("references").([]string) // Enforced by linter
		for _, reference := range references {
			if markdown.MatchWikilink(reference) {
				addWikilink(reference, "referenced_by")
			}
		}
	}

	// Check attribute "inspirations"
	if n.HasAttribute("inspirations") {
		inspirations := n.GetAttribute("inspirations").([]string) // Enforced by linter
		for _, inspiration := range inspirations {
			if markdown.MatchWikilink(inspiration) {
				addWikilink(inspiration, "inspired_by")
			}
		}
	}

	return links
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
		n.Type = parsedNote.Type
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

	if !reflect.DeepEqual(n.Items, parsedNote.Items) {
		n.Items = parsedNote.Items
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

func (n *Note) GetAttributes() map[string]any {
	// Present to be consistent with File.GetAttributes()
	return n.Attributes
}

func (n *Note) HasAttribute(name string) bool {
	_, ok := n.Attributes[name]
	return ok
}

func (n *Note) SetAttribute(name string, value any) {
	if n.Attributes == nil {
		n.Attributes = make(map[string]any)
	}
	n.Attributes[name] = value
}

func (n *Note) GetAttribute(name string) any {
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

/* State Management */

func (n *Note) Save() error {
	CurrentLogger().Debugf("Saving note %s...", n.Wikilink)
	query := `
		INSERT INTO note(
			oid,
			packfile_oid,
			file_oid,
			slug,
			note_type,
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
			items,
			created_at,
			updated_at,
			indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			file_oid = ?,
			slug = ?,
			note_type = ?,
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
			items = ?,
			updated_at = ?,
			indexed_at = ?
		;
	`

	attributesJSON, err := n.Attributes.ToJSON()
	if err != nil {
		return err
	}

	itemsJSON, err := json.Marshal(n.Items)
	if err != nil {
		return err
	}

	_, err = CurrentDB().Client().Exec(query,
		// Insert
		n.OID,
		n.PackFileOID,
		n.FileOID,
		n.Slug,
		n.Type,
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
		string(itemsJSON),
		timeToSQL(n.CreatedAt),
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.IndexedAt),
		// Update
		n.PackFileOID,
		n.FileOID,
		n.Slug,
		n.Type,
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
		string(itemsJSON),
		timeToSQL(n.UpdatedAt),
		timeToSQL(n.IndexedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (n *Note) SaveMetadata() error {
	CurrentLogger().Debugf("Saving note %s...", n.Wikilink)
	query := `
		UPDATE note
		SET
			marked = ?,
			marked_at = ?,
			annotations = ?
		WHERE oid = ?
		;
	`

	annotationsJSON, err := json.MarshalIndent(n.Annotations, "", "  ")
	if err != nil {
		return err
	}

	_, err = CurrentDB().Client().Exec(query,
		n.Marked,
		timeToSQL(n.MarkedAt),
		annotationsJSON,
		n.OID,
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

// CountNotesByTypes returns the total number of notes for every type.
func (r *Repository) CountNotesByTypes() (map[string]int, error) {
	// Prepare the query
	query := `SELECT note_type, count(*) FROM note GROUP BY note_type ORDER BY count(*) DESC;`

	// Execute the query
	rows, err := CurrentDB().Client().Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Iterate over the results
	result := make(map[string]int)
	for rows.Next() {
		var noteType string
		var count int

		// Scan the row into variables
		if err := rows.Scan(&noteType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Store the result in the map
		result[noteType] = count
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return result, nil
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



func (r *Repository) LoadNoteByOID(oid oid.OID) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE oid = ?`, "", oid)
}



func (r *Repository) FindNotesByFileOID(oid oid.OID) ([]*Note, error) {
	return QueryNotes(CurrentDB().Client(), `WHERE file_oid = ?`, oid)
}

func (r *Repository) FindNoteByTitle(title string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE title = ?`, "", title)
}

func (r *Repository) FindNoteBySlug(slug string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE slug = ?`, "", slug)
}



func (r *Repository) FindNoteByPathAndTitle(relativePath string, title string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE relative_path = ? AND title = ?`, "", relativePath, title)
}

func (r *Repository) FindMatchingNote(parsedNote *ParsedNote) (*Note, error) {
	// Try by slug
	note, _ := r.FindNoteBySlug(parsedNote.Slug)
	if note != nil {
		return note, nil
	}

	// Try by wikilink
	note, _ = r.FindNoteByWikilink(parsedNote.RelativePath + "#" + string(parsedNote.Title))
	if note != nil {
		return note, nil
	}

	// Last by same title or same content in the same file
	return QueryNote(CurrentDB().Client(), `WHERE relative_path = ? AND (title = ? OR hashsum = ?)`, "", parsedNote.RelativePath, parsedNote.Title, parsedNote.Hash())
}

func (r *Repository) FindNoteByWikilink(wikilink string) (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE wikilink LIKE ?`, "", "%"+wikilink)
}

func (r *Repository) FindNotesByWikilink(wikilink string) ([]*Note, error) {
	return QueryNotes(CurrentDB().Client(), `WHERE wikilink LIKE ?`, "%"+wikilink)
}

func (r *Repository) GetRandomQuote() (*Note, error) {
	return QueryNote(CurrentDB().Client(), `WHERE note_type = "Quote"`, "ORDER BY RANDOM() LIMIT 1")
}

// SearchNotes query notes to find the ones matching a list of criteria.
//
// Examples:
//
//	tag:favorite type:reference type:note path:projects/
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
	if len(query.Types) > 0 {
		var typesSQL []string
		for _, noteType := range query.Types {
			typesSQL = append(typesSQL, fmt.Sprintf(`"%s"`, noteType))
		}
		querySQL.WriteString(fmt.Sprintf("AND note.note_type IN (%s) ", strings.Join(typesSQL, ",")))
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
	CurrentLogger().Trace(querySQL.String())
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

func QueryNote(db SQLClient, whereClause string, randomClause string, args ...any) (*Note, error) {
	var n Note
	var createdAt string
	var updatedAt string
	var lastIndexedAt string
	var markedAt sql.NullString
	var tagsRaw string
	var attributesRaw string
	var annotationsRaw string
	var itemsRaw string

	// Build the complete query with optional random clause
	query := fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			file_oid,
			slug,
			note_type,
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
			items,
			created_at,
			updated_at,
			indexed_at,
			marked,
			marked_at,
			annotations
		FROM note
		%s %s;`, whereClause, randomClause)

	// Query for a value based on a single row.
	if err := db.QueryRow(query, args...).
		Scan(
			&n.OID,
			&n.PackFileOID,
			&n.FileOID,
			&n.Slug,
			&n.Type,
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
			&itemsRaw,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
			&n.Marked,
			&markedAt,
			&annotationsRaw,
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

	var items Items
	if itemsRaw != "" {
		err = json.Unmarshal([]byte(itemsRaw), &items)
		if err != nil {
			return nil, err
		}
	}

	annotations := make([]Annotation, 0)
	if annotationsRaw != "" {
		err = yaml.Unmarshal([]byte(annotationsRaw), &annotations)
		if err != nil {
			return nil, err
		}
	}

	n.Attributes = attributes.CastOrIgnore(CurrentConfigFile().Attributes)
	n.Tags = strings.Split(tagsRaw, ",")
	n.Items = &items
	n.CreatedAt = timeFromSQL(createdAt)
	n.UpdatedAt = timeFromSQL(updatedAt)
	n.IndexedAt = timeFromSQL(lastIndexedAt)
	n.MarkedAt = timeFromNullableSQL(markedAt)
	n.Annotations = annotations

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
			note_type,
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
			items,
			created_at,
			updated_at,
			indexed_at,
			marked,
			marked_at,
			annotations
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
		var markedAt sql.NullString
		var tagsRaw string
		var attributesRaw string
		var annotationsRaw string
		var itemsRaw string

		err = rows.Scan(
			&n.OID,
			&n.PackFileOID,
			&n.FileOID,
			&n.Slug,
			&n.Type,
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
			&itemsRaw,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
			&n.Marked,
			&markedAt,
			&annotationsRaw,
		)
		if err != nil {
			return nil, err
		}

		attributes, err := NewAttributeSetFromYAML(attributesRaw)
		if err != nil {
			return nil, err
		}

		var items Items
		if itemsRaw != "" {
			err = json.Unmarshal([]byte(itemsRaw), &items)
			if err != nil {
				return nil, err
			}
		}

		annotations := make([]Annotation, 0)
		if annotationsRaw != "" {
			err = yaml.Unmarshal([]byte(annotationsRaw), &annotations)
			if err != nil {
				return nil, err
			}
		}

		n.Attributes = attributes.CastOrIgnore(CurrentConfigFile().Attributes)
		n.Tags = strings.Split(tagsRaw, ",")
		n.Items = &items
		n.CreatedAt = timeFromSQL(createdAt)
		n.UpdatedAt = timeFromSQL(updatedAt)
		n.IndexedAt = timeFromSQL(lastIndexedAt)
		n.MarkedAt = timeFromNullableSQL(markedAt)
		n.Annotations = annotations

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

/* Operations */

// Mark set a flag on the note.
func (n *Note) Mark(timestamp time.Time) {
	if n.MarkedAt.IsZero() {
		n.Marked = true
		n.MarkedAt = timestamp
		return
	}

	if n.MarkedAt.After(timestamp) {
		// Ignore out-of-order operations
		return
	}

	n.Marked = true
	n.MarkedAt = timestamp
}

// Unmark set a flag on the note.
func (n *Note) Unmark(timestamp time.Time) {
	if n.MarkedAt.IsZero() {
		n.Marked = false
		n.MarkedAt = timestamp
		return
	}

	if n.MarkedAt.After(timestamp) {
		// Ignore out-of-order operations
		return
	}

	n.Marked = false
	n.MarkedAt = timestamp
}

type Annotation struct {
	OID       oid.OID   `yaml:"oid" json:"oid"`
	Text      string    `yaml:"text" json:"text"`
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
}

// AddAnnotation adds an annotation to the note.
func (n *Note) AddAnnotation(timestamp time.Time, annotation Annotation) {
	for _, existing := range n.Annotations {
		if existing.OID == annotation.OID {
			// Ignore out-of-order operations
			if existing.CreatedAt.Before(timestamp) {
				// Remove before inserting the updated annotation
				n.RemoveAnnotation(timestamp, existing)
			}
			break
		}
	}

	annotation.CreatedAt = timestamp
	n.Annotations = append(n.Annotations, annotation)
}

// RemoveAnnotation removes an annotation from the note.
func (n *Note) RemoveAnnotation(timestamp time.Time, annotation Annotation) {
	for i, existing := range n.Annotations {
		if existing.OID == annotation.OID {
			// Ignore out-of-order operations
			if existing.CreatedAt.Before(timestamp) {
				// Remove existing annotation
				n.Annotations = append(n.Annotations[:i], n.Annotations[i+1:]...)
				return
			}
		}
	}
}
