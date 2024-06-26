package core

import (
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/helpers"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type Attribute struct { // TODO remove
	Key   string
	Value interface{}
}

type File struct {
	// A unique identifier among all files
	OID string `yaml:"oid" json:"oid"`
	// A unique human-friendly slug
	Slug string `yaml:"slug" json:"slug"`

	// Optional parent file (= index.md)
	ParentFileOID string `yaml:"file_oid,omitempty" json:"file_oid,omitempty"`
	ParentFile    *File  `yaml:"-" json:"-"` // Lazy-loaded

	// A relative path to the repository directory
	RelativePath string `yaml:"relative_path" json:"relative_path"`
	// The full wikilink to this file (without the extension)
	Wikilink string `yaml:"wikilink" json:"wikilink"`

	// The FrontMatter for the note file
	FrontMatter markdown.FrontMatter `yaml:"front_matter" json:"front_matter"`

	// Merged attributes
	Attributes AttributeSet `yaml:"attributes,omitempty" json:"attributes,omitempty"`

	// Original title of the main heading without leading # characters
	Title markdown.Document `yaml:"title,omitempty" json:"title,omitempty"`
	// Short title of the main heading without the kind prefix if present
	ShortTitle markdown.Document `yaml:"short_title,omitempty" json:"short_title,omitempty"`

	Body     markdown.Document `yaml:"body" json:"body"`
	BodyLine int               `yaml:"body_line" json:"body_line"`

	// Subobjects (lazy-loaded)
	notes      []*Note      `yaml:"-" json:"-"`
	flashcards []*Flashcard `yaml:"-" json:"-"`

	// Permission of the file (required to save back)
	Mode fs.FileMode `yaml:"mode" json:"mode"`
	// Size of the file (can be useful to detect changes)
	Size int64 `yaml:"size" json:"size"`
	// Hash of the content (can be useful to detect changes too)
	Hash string `yaml:"hash" json:"hash"`
	// Content last modification date
	MTime time.Time `yaml:"mtime" json:"mtime"`

	CreatedAt     time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at" json:"updated_at"`
	DeletedAt     time.Time `yaml:"deleted_at,omitempty" json:"deleted_at,omitempty"`
	LastCheckedAt time.Time `yaml:"-" json:"-"`

	new   bool
	stale bool
}

// NewXFromParsedX
// -> do not check the database
// NewXOrExistingFromParsedX
// -> check the database and then call update() or NewXFromParsedX()

/*
A file can be new but contains notes/flashcards that was moved => recreating would mean relearning the flashcard...
A file can be new and contains GoLink that were moved => could be deleted/recreated without consequences
A file can be new and references existing medias => must not recreate blobs if they already exists!

The minimum building block that can be added is a Markdown file.
When adding a file:
- Create/Update the file
- Create/Update the notes
- Create/Update the medias
- Create/Update the reminders
- Create/Update the links

CurrentRepository().FindParentFileFromParsedFile(ParsedFile) // Useful to make NewOrExisting clearer
CurrentRepository().FindFileFromParsedFile(ParsedFile)
CurrentRepository().FindNoteFromParsedNote(ParsedNote)
...

IDEA rename CurrentX() by X() => make less obvious the singleton pattern but make the code shorter 🤷‍♂️
Repository().
Database().
Logger


func NewOrExistingFile(parsedFile *ParsedFile) (*File, error) {
	parent, _ := CurrentRepository().FindParentFileFromParsedFile(parsedFile)
	existing, _ := CurrentRepository().FindFileFromParsedFile(parsedFile)
	if existing != nil {
		err := existing.Update(parent, parsedFile)
		return existing, err
	} else {
	 	return NewFile(parent, parsedFile)
	}
}

func NewFile(parent *File, parsedFile *Parsed) (*File, error) {
	return &File{
		// use parsedFile to copy values
	}
}

func (f *file) Update(parent, parsedFile) error {
	// Update parsedFile to copy values
}

The problem is where to create SubObjects()?

1. In NewOrExistingFile() + NewFile() + Update()
  * Duplication
2. In Add() command:

```go
repository.Walk(func(md *MarkdownFile) error {
	// Parse all objects inside the Markdown document
	parsedFile := NewParseFile(md) // NewFile // NewPackFile

	// Process new medias (they are independant of the file)
	var newMedias []Media
	for _, media := range parsedFile.Medias {
		media := NewOrExistingParsedMedia(parsedMedia)
		if media.Stale() {
			newMedias = append(newMedias, media)
		}
	}

	// Process blobs first outside the SQL transaction (takes a long time and resuming if the command is interrupted is not dangerous, just some CPU cycles lost)
	CurrentLogger().Printf("Processing %d medias", len(newMedias))
	for _, newMedia := range newMedias {
		newMedia.WriteBlobs()
	}

	CurrentDB().BeginTransaction()
	defer CurrentDB().RollbackTransaction()

	// Same the medias
	for _, newMedia := range newMedias {
		newMedia.Save()
	}

	// Process links (they are independant of the file)
	...

	// Process the file
	file := NewOrExistingParsedFile(parsedFile)
	if file.Stale() {
		file.Save()
	}

	// Process the note
	for _, parsedNote := range parsedFile.Notes {
		note := NewOrExistingNote(parsedNote)
		if note.Stale() {
			note.Save()
		}
	}
	// Note can reference and embed each other.
	// Do a second pass (on first pass, a embedded note may not have been found because it was processed later)
	for _, parsedNote := range parsedFile.Notes {
		note := NewOrExistingNote(parsedNote)
		if note.Stale() {
			note.Save()
		}
		if parsedNote.Flashcard != nil {
			flashcard := NewOrExistingFlashcard(parsedNote.Flashcard)
			if flashcard.Stale() {
				flashcard.Save()
			}
		}
		for _, parsedReminder := range parsedNote.Reminders {
			reminder := NewOrExistingReminder(parsedReminder)
			if reminder.Stale() {
				reminder.Save()
			}
		}
	}

	CurrentDB().CommitTransaction()

	packFile := file.ToPackFile()
	CurrentIndex().StagePackFile(packFile)
	// create the file on disk
	// update the index file to note the packfile OID in the staging area
})

func (r *Repository) Reset() {
	In short, we must revert changes done on files.

	// Read the staging area to find staged packfiles
	// Read the index to find the latest packfile for every staged packfile

	DeleteObject in staged packFile => What about flashcard attributes?????
	SaveObject in indexed packFile

	OR

	re-SaveObject() in indexed packFile
	DeleteObject() not present in indexed packFile
	DeletePackFile()

	use .pack and .blob as extension to make easy to clean up? (yes, but medias are pack files too)
}
```


NewOrExistingFile(*ParsedFile)

NewFile(parent *File, *ParsedFile)
-> Do not




*/

// NewFileFromParsedFile must iterate over subobjects (ex: notes) to call NewNoteFromParsedNote()
// If

/* Creation */

func NewEmptyFile(name string) *File { // TODO still useful?
	return &File{
		OID:          NewOID(),
		Slug:         "",
		stale:        true,
		new:          true,
		Wikilink:     name,
		RelativePath: name,
		Attributes:   make(map[string]interface{}),
	}
}

func NewOrExistingFile(parsedFile *ParsedFile) (*File, error) {
	var existingParent *File
	var existingFile *File

	// Look for the parent file
	if parsedFile.Filename() != "index.md" {
		file, err := CurrentRepository().FindMatchingParentFile(parsedFile)
		if err != nil {
			return nil, err
		}
		existingParent = file
	}

	file, err := CurrentRepository().FindMatchingFile(parsedFile)
	if err != nil {
		return nil, err
	}
	existingFile = file

	if existingFile != nil {
		err := existingFile.update(existingParent, parsedFile)
		return existingFile, err
	} else {
		return NewFile(existingParent, parsedFile)
	}
}

func NewFile(parent *File, parsedFile *ParsedFile) (*File, error) {
	file := &File{
		OID:          NewOID(),
		Slug:         parsedFile.Slug,
		RelativePath: parsedFile.RelativePath,
		Wikilink:     text.TrimExtension(parsedFile.RelativePath),
		Mode:         parsedFile.Markdown.LStat.Mode(),
		Size:         parsedFile.Markdown.LStat.Size(),
		Hash:         helpers.Hash(parsedFile.Markdown.Content),
		MTime:        parsedFile.Markdown.LStat.ModTime(),
		Attributes:   make(map[string]any),
		FrontMatter:  parsedFile.Markdown.FrontMatter,
		Body:         parsedFile.Markdown.Body,
		BodyLine:     parsedFile.Markdown.BodyLine,
		CreatedAt:    clock.Now(),
		UpdatedAt:    clock.Now(),
		stale:        true,
		new:          true,
	}
	if parent != nil {
		file.ParentFileOID = parent.OID
		file.ParentFile = parent
	}
	newAttributes := parsedFile.FileAttributes
	if parent != nil {
		// TODO now cast attributes
		newAttributes = file.mergeAttributes(parent.GetAttributes(), newAttributes)
	}
	file.Attributes = newAttributes

	return file, nil
}

func (f *File) mergeAttributes(attributes ...AttributeSet) AttributeSet { // TODO r
	// File attributes are always inheritable to top level-notes
	// (NB: `source` is configured to be non-inheritable).
	//
	// Ex:
	//   ---
	//   source: XXX
	//   ---
	//   # Example
	//   ## Note: Parent
	//   ### Note: Child
	//
	// Is the same as:
	//
	//   # Example
	//   ## Note: Parent
	//   `@source:XXX`
	//   ### Note: Child
	return EmptyAttributes.Merge(attributes...)
}

/* Object */

func (f *File) Kind() string {
	return "file"
}

func (f *File) UniqueOID() string {
	return f.OID
}

func (f *File) Refresh() (bool, error) {
	// No dependencies = no need to refresh
	return false, nil
}

func (f *File) Stale() bool {
	return f.stale
}

func (f *File) State() State {
	if !f.DeletedAt.IsZero() {
		return Deleted
	}
	if f.new {
		return Added
	}
	if f.stale {
		return Modified
	}
	return None
}

func (f *File) ForceState(state State) {
	switch state {
	case Added:
		f.new = true
	case Deleted:
		f.DeletedAt = clock.Now()
	}
	f.stale = true
}

func (f *File) SetAlive() {
	f.DeletedAt = clock.Now()
	f.stale = true
}

func (f *File) ModificationTime() time.Time {
	return f.MTime
}

func (f *File) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(f)
	if err != nil {
		return err
	}
	return nil
}

func (f *File) Write(w io.Writer) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (f *File) Relations() []*Relation {
	// We consider only relations related to notes
	return nil
}

func (f *File) Blobs() []*BlobRef {
	return nil
}

func (f File) String() string {
	return fmt.Sprintf("file %q [%s]", f.RelativePath, f.OID)
}

/* Update */

func (f *File) update(parent *File, parsedFile *ParsedFile) error {
	newAttributes := parsedFile.FileAttributes
	if parent != nil {
		newAttributes = f.mergeAttributes(parent.GetAttributes(), newAttributes)
	}

	// Check if attributes have changed
	if !reflect.DeepEqual(newAttributes, f.Attributes) {
		f.stale = true
		f.Attributes = newAttributes
	}

	md := parsedFile.Markdown

	// Check if local file has changed
	if f.MTime != md.LStat.ModTime() || f.Size != md.LStat.Size() {
		// file change
		f.stale = true

		f.Mode = md.LStat.Mode()
		f.Size = md.LStat.Size()
		f.Hash = helpers.Hash(md.Content)
		f.FrontMatter = md.FrontMatter
		f.Attributes = parsedFile.FileAttributes
		if parent != nil {
			f.Attributes = f.mergeAttributes(parent.GetAttributes(), f.Attributes)
		}
		f.MTime = md.LStat.ModTime()
		f.Body = md.Body
		f.BodyLine = md.BodyLine
	}

	return nil
}

/* State Management */

func (f *File) New() bool {
	return f.new
}

func (f *File) Updated() bool {
	return f.stale
}

/* Front Matter */

// AbsoluteBodyLine returns the line number in the file by taking into consideration the front matter.
func (f *File) AbsoluteBodyLine(bodyLine int) int {
	return f.BodyLine + bodyLine - 1
}

// GetAttributes returns all file-specific and inherited attributes.
func (f *File) GetAttributes() map[string]interface{} {
	return f.Attributes
}

// GetAttribute extracts a single attribute value at the top.
func (f *File) GetAttribute(key string) interface{} {
	value, ok := f.Attributes[key]
	if !ok {
		return nil
	}
	return value
}

// GetTags returns all defined tags.
func (f *File) GetTags() []string {
	value := f.GetAttribute("tags")
	if tag, ok := value.(string); ok {
		return []string{tag}
	}
	if tags, ok := value.([]string); ok {
		return tags
	}
	if rawTags, ok := value.([]interface{}); ok {
		var tags []string
		for _, rawTag := range rawTags {
			if tag, ok := rawTag.(string); ok {
				tags = append(tags, tag)
			}
		}
		return tags
	}
	return nil
}

// HasTag returns if a file has a given tag.
func (f *File) HasTag(name string) bool {
	return slices.Contains(f.GetTags(), name)
}

/* Content */

func (f *File) GetNotes() []*Note {
	if f.notes != nil {
		return f.notes
	}

	// TODO CurrentRepository().FindNotes()
	return nil
}

func (f *File) GetFlashcards() []*Flashcard {
	if f.flashcards != nil {
		return f.flashcards
	}

	// TODO CurrentRepository().FindFlashcards()
	return nil
}

// FindNoteByKindAndShortTitle searches for a given note based on its kind and title.
func (f *File) FindNoteByKindAndShortTitle(kind NoteKind, shortTitle string) *Note {
	for _, note := range f.GetNotes() {
		if note.NoteKind == kind && note.ShortTitle == markdown.Document(shortTitle) {
			return note
		}
	}
	return nil
}

// FindFlashcardByTitle searches for a given flashcard based on its title.
func (f *File) FindFlashcardByTitle(shortTitle string) *Flashcard {
	for _, flashcard := range f.GetFlashcards() {
		if flashcard.ShortTitle == markdown.Document(shortTitle) {
			return flashcard
		}
	}
	return nil
}

/* Data Management */

func (f *File) Check() error {
	client := CurrentDB().Client()
	CurrentLogger().Debugf("Checking file %s...", f.RelativePath)
	f.LastCheckedAt = clock.Now()
	query := `
		UPDATE file
		SET last_checked_at = ?
		WHERE oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}
	query = `
		UPDATE note
		SET last_checked_at = ?
		WHERE file_oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}
	query = `
		UPDATE flashcard
		SET last_checked_at = ?
		WHERE file_oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}
	query = `
		UPDATE reminder
		SET last_checked_at = ?
		WHERE file_oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}

	return nil
}

func (f *File) Save() error {
	var err error
	f.UpdatedAt = clock.Now()
	f.LastCheckedAt = clock.Now()
	switch f.State() {
	case Added:
		err = f.Insert()
	case Modified:
		err = f.Update()
	case Deleted:
		err = f.Delete()
	default:
		err = f.Check()
	}
	if err != nil {
		return err
	}
	f.new = false
	f.stale = false
	return nil
}

func (f *File) Insert() error {
	CurrentLogger().Debugf("Inserting file %s...", f.RelativePath)
	query := `
		INSERT INTO file(
			oid,
			file_oid,
			slug,
			relative_path,
			wikilink,
			front_matter,
			attributes,
			title,
			short_title,
			body,
			body_line,
			created_at,
			updated_at,
			last_checked_at,
			mtime,
			size,
			hashsum,
			mode
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	frontMatter, err := f.FrontMatter.AsBeautifulYAML()
	if err != nil {
		return err
	}
	attributesJSON, err := f.Attributes.ToJSON()
	if err != nil {
		return err
	}

	_, err = CurrentDB().Client().Exec(query,
		f.OID,
		f.ParentFileOID,
		f.Slug,
		f.RelativePath,
		f.Wikilink,
		frontMatter,
		attributesJSON,
		f.Title,
		f.ShortTitle,
		f.Body,
		f.BodyLine,
		timeToSQL(f.CreatedAt),
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.LastCheckedAt),
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
		f.Mode,
	)
	if err != nil {
		return err
	}

	return nil
}

func (f *File) Update() error {
	CurrentLogger().Debugf("Updating file %s...", f.RelativePath)
	query := `
		UPDATE file
		SET
		    file_oid = ?,
		    slug = ?,
			relative_path = ?,
			wikilink = ?,
			front_matter = ?,
			attributes = ?,
			title = ?,
			short_title = ?,
			body = ?,
			body_line = ?,
			updated_at = ?,
			last_checked_at = ?,
			mtime = ?,
			size = ?,
			hashsum = ?,
			mode = ?
		WHERE oid = ?;
	`
	frontMatter, err := f.FrontMatter.AsBeautifulYAML()
	if err != nil {
		return err
	}
	attributesJSON, err := f.Attributes.ToJSON()
	if err != nil {
		return err
	}
	_, err = CurrentDB().Client().Exec(query,
		f.ParentFileOID,
		f.Slug,
		f.RelativePath,
		f.Wikilink,
		frontMatter,
		attributesJSON,
		f.Title,
		f.ShortTitle,
		f.Body,
		f.BodyLine,
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.LastCheckedAt),
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
		f.Mode,
		f.OID,
	)
	return err
}

func (f *File) Delete() error {
	CurrentLogger().Debugf("Deleting file %s...", f.RelativePath)
	query := `DELETE FROM file WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query, f.OID)
	return err
}

func (r *Repository) LoadFileByOID(oid string) (*File, error) {
	return QueryFile(CurrentDB().Client(), `WHERE oid = ?`, oid)
}

func (r *Repository) FindFileByRelativePath(relativePath string) (*File, error) {
	return QueryFile(CurrentDB().Client(), `WHERE relative_path = ?`, relativePath)
}

func (r *Repository) FindMatchingFile(parsedFile *ParsedFile) (*File, error) {
	return QueryFile(CurrentDB().Client(), `WHERE relative_path = ?`, parsedFile.RelativePath)
}

func (r *Repository) FindMatchingParentFile(parsedFile *ParsedFile) (*File, error) {
	if parsedFile.Filename() == "index.md" {
		return nil, nil
	}
	parentRelativePath := filepath.Join(parsedFile.RelativeDir(), "index.md")
	return r.FindFileByRelativePath(parentRelativePath)
}

func (r *Repository) FindFilesByRelativePathPrefix(relativePathPrefix string) ([]*File, error) {
	return QueryFiles(CurrentDB().Client(), `WHERE relative_path LIKE ?`, relativePathPrefix+"%")
}

func (r *Repository) FindFileByWikilink(wikilink string) (*File, error) {
	return QueryFile(CurrentDB().Client(), `WHERE wikilink LIKE ?`, "%"+text.TrimExtension(wikilink))
}

func (r *Repository) FindFilesByWikilink(wikilink string) ([]*File, error) {
	return QueryFiles(CurrentDB().Client(), `WHERE wikilink LIKE ?`, "%"+text.TrimExtension(wikilink))
}

func (r *Repository) FindFilesLastCheckedBefore(point time.Time, path string) ([]*File, error) {
	if path == "." {
		path = ""
	}
	return QueryFiles(CurrentDB().Client(), `WHERE last_checked_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

// CountFiles returns the total number of files.
func (r *Repository) CountFiles() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM file`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

/* SQL Helpers */

func QueryFile(db SQLClient, whereClause string, args ...any) (*File, error) {
	var f File
	var createdAt string
	var updatedAt string
	var lastCheckedAt string
	var mTime string
	var attributesRaw string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			slug,
			relative_path,
			wikilink,
			front_matter,
			attributes,
			title,
			short_title,
			body,
			body_line,
			created_at,
			updated_at,
			last_checked_at,
			mtime,
			size,
			hashsum,
			mode
		FROM file
		%s;`, whereClause), args...).
		Scan(
			&f.OID,
			&f.ParentFileOID,
			&f.Slug,
			&f.RelativePath,
			&f.Wikilink,
			&f.FrontMatter,
			&attributesRaw,
			&f.Title,
			&f.ShortTitle,
			&f.Body,
			&f.BodyLine,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
			&mTime,
			&f.Size,
			&f.Hash,
			&f.Mode,
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

	f.Attributes = attributes.Cast(GetSchemaAttributeTypes())
	f.CreatedAt = timeFromSQL(createdAt)
	f.UpdatedAt = timeFromSQL(updatedAt)
	f.LastCheckedAt = timeFromSQL(lastCheckedAt)
	f.MTime = timeFromSQL(mTime)

	return &f, nil
}

func QueryFiles(db SQLClient, whereClause string, args ...any) ([]*File, error) {
	var files []*File

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			slug,
			relative_path,
			wikilink,
			front_matter,
			attributes,
			title,
			short_title,
			body,
			body_line,
			created_at,
			updated_at,
			last_checked_at,
			mtime,
			size,
			hashsum,
			mode
		FROM file
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f File
		var createdAt string
		var updatedAt string
		var lastCheckedAt string
		var mTime string
		var attributesRaw string

		err = rows.Scan(
			&f.OID,
			&f.ParentFileOID,
			&f.Slug,
			&f.RelativePath,
			&f.Wikilink,
			&f.FrontMatter,
			&attributesRaw,
			&f.Title,
			&f.ShortTitle,
			&f.Body,
			&f.BodyLine,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
			&mTime,
			&f.Size,
			&f.Hash,
			&f.Mode,
		)
		if err != nil {
			return nil, err
		}

		attributes, err := NewAttributeSetFromYAML(attributesRaw)
		if err != nil {
			return nil, err
		}

		f.Attributes = attributes.Cast(GetSchemaAttributeTypes())
		f.CreatedAt = timeFromSQL(createdAt)
		f.UpdatedAt = timeFromSQL(updatedAt)
		f.LastCheckedAt = timeFromSQL(lastCheckedAt)
		f.MTime = timeFromSQL(mTime)

		files = append(files, &f)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return files, err
}

/* Format */

func (f *File) ToYAML() string {
	return ToBeautifulYAML(f)
}

func (f *File) ToJSON() string {
	return ToBeautifulJSON(f)
}

func (f *File) ToMarkdown() string {
	var sb strings.Builder
	frontMatter, err := f.FrontMatter.AsBeautifulYAML()
	if err != nil {
		sb.WriteString(frontMatter)
	}
	sb.WriteRune('\n')
	sb.WriteRune('\n')
	sb.WriteString(string(f.Body))
	return sb.String()
}
