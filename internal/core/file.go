package core

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/helpers"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/internal/medias"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type File struct {
	// A unique identifier among all files
	OID oid.OID `yaml:"oid" json:"oid"`
	// A unique human-friendly slug
	Slug string `yaml:"slug" json:"slug"`

	// Pack file where this object belongs
	PackFileOID oid.OID `yaml:"packfile_oid" json:"packfile_oid"`

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
	notes      []*Note      `yaml:"-" json:"-"` // TODO still useful?
	flashcards []*Flashcard `yaml:"-" json:"-"` // TODO still useful?

	// TODO still useful as these fields are stored in the index?
	// Size of the file (can be useful to detect changes)
	Size int64 `yaml:"size" json:"size"`
	// Hash of the content (can be useful to detect changes too)
	Hash string `yaml:"hash" json:"hash"`
	// Content last modification date
	MTime time.Time `yaml:"mtime" json:"mtime"`

	// Eager-loaded list of blobs
	BlobRefs []*BlobRef `yaml:"blobs,omitempty" json:"blobs,omitempty"`

	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
	IndexedAt time.Time `yaml:"indexed_at,omitempty" json:"indexed_at,omitempty"`
}

/* Creation */

func NewEmptyFile(name string) *File { // TODO still useful?
	return &File{
		OID:          oid.New(),
		Slug:         "",
		Wikilink:     name,
		RelativePath: name,
		Attributes:   make(map[string]interface{}),
	}
}

func NewOrExistingFile(packFile *PackFile, parsedFile *ParsedFile) (*File, error) {
	// Try to find an existing object (instead of recreating it from scratch after every change)
	existingFile, err := CurrentRepository().FindMatchingFile(parsedFile)
	if err != nil {
		return nil, err
	}
	if existingFile != nil {
		err := existingFile.update(packFile, parsedFile)
		return existingFile, err
	}
	return NewFile(packFile, parsedFile)
}

func NewFile(packFile *PackFile, parsedFile *ParsedFile) (*File, error) {
	file := &File{
		OID:          oid.New(),
		PackFileOID:  packFile.OID,
		Slug:         parsedFile.Slug,
		RelativePath: parsedFile.RelativePath,
		Wikilink:     text.TrimExtension(parsedFile.RelativePath),
		Size:         parsedFile.Markdown.Size,
		MTime:        parsedFile.Markdown.MTime,
		Hash:         helpers.Hash(parsedFile.Markdown.Content),
		Attributes:   parsedFile.FileAttributes,
		FrontMatter:  parsedFile.Markdown.FrontMatter,
		Title:        parsedFile.Title,
		ShortTitle:   parsedFile.ShortTitle,
		Body:         parsedFile.Markdown.Body,
		BodyLine:     parsedFile.Markdown.BodyLine,
		CreatedAt:    packFile.CTime,
		UpdatedAt:    packFile.CTime,
		IndexedAt:    packFile.CTime,
	}

	return file, nil
}

/* Object */

func (f *File) Kind() string {
	return "file"
}

func (f *File) UniqueOID() oid.OID {
	return f.OID
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

func (f File) String() string {
	return fmt.Sprintf("file %q [%s]", f.RelativePath, f.OID)
}

/* Update */

func (f *File) update(packFile *PackFile, parsedFile *ParsedFile) error {
	stale := false

	newAttributes := parsedFile.FileAttributes

	// Check if attributes have changed
	if !reflect.DeepEqual(newAttributes, f.Attributes) {
		stale = true
		f.Attributes = newAttributes
	}

	md := parsedFile.Markdown

	// Check if local file has changed
	if f.MTime != md.MTime || f.Size != md.Size {
		stale = true

		f.Size = md.Size
		f.MTime = md.MTime
		f.Hash = helpers.Hash(md.Content)
		f.FrontMatter = md.FrontMatter
		f.Attributes = parsedFile.FileAttributes
		f.Body = md.Body
		f.BodyLine = md.BodyLine
	}

	f.PackFileOID = packFile.OID
	f.IndexedAt = packFile.CTime

	if stale {
		f.UpdatedAt = packFile.CTime
	}

	return nil
}

/* Front Matter */

// AbsoluteBodyLine returns the line number in the file by taking into consideration the front matter.
func (f *File) AbsoluteBodyLine(bodyLine int) int {
	return f.BodyLine + bodyLine - 1
}

// GetTags returns the tags defined in attributes.
func (f *File) GetTags() []string {
	return f.Attributes.Tags()
}

// HasTag returns if a file has a given tag.
func (f *File) HasTag(name string) bool {
	return slices.Contains(f.Attributes.Tags(), name)
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

func (f *File) Save() error {
	CurrentLogger().Debugf("Saving file %s...", f.RelativePath)
	query := `
		INSERT INTO file(
			oid,
			packfile_oid,
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
			indexed_at,
			mtime,
			size,
			hashsum
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
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
			indexed_at = ?,
			mtime = ?,
			size = ?,
			hashsum = ?;
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
		// Insert
		f.OID,
		f.PackFileOID,
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
		timeToSQL(f.IndexedAt),
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
		// Update
		f.PackFileOID,
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
		timeToSQL(f.IndexedAt),
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
	)
	if err != nil {
		return err
	}

	return nil
}

func (f *File) Delete() error {
	CurrentLogger().Debugf("Deleting file %s...", f.RelativePath)
	query := `DELETE FROM file WHERE oid = ? AND packfile_oid = ?;`
	_, err := CurrentDB().Client().Exec(query, f.OID, f.PackFileOID)
	return err
}

func (r *Repository) LoadFileByOID(oid oid.OID) (*File, error) {
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
	return QueryFiles(CurrentDB().Client(), `WHERE indexed_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
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
	var lastIndexedAt string
	var mTime string
	var attributesRaw string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
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
			indexed_at,
			mtime,
			size,
			hashsum
		FROM file
		%s;`, whereClause), args...).
		Scan(
			&f.OID,
			&f.PackFileOID,
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
			&lastIndexedAt,
			&mTime,
			&f.Size,
			&f.Hash,
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

	f.Attributes = attributes.CastOrIgnore(GetSchemaAttributeTypes())
	f.CreatedAt = timeFromSQL(createdAt)
	f.UpdatedAt = timeFromSQL(updatedAt)
	f.IndexedAt = timeFromSQL(lastIndexedAt)
	f.MTime = timeFromSQL(mTime)

	return &f, nil
}

func QueryFiles(db SQLClient, whereClause string, args ...any) ([]*File, error) {
	var files []*File

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
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
			indexed_at,
			mtime,
			size,
			hashsum
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
		var lastIndexedAt string
		var mTime string
		var attributesRaw string

		err = rows.Scan(
			&f.OID,
			&f.PackFileOID,
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
			&lastIndexedAt,
			&mTime,
			&f.Size,
			&f.Hash,
		)
		if err != nil {
			return nil, err
		}

		attributes, err := NewAttributeSetFromYAML(attributesRaw)
		if err != nil {
			return nil, err
		}

		f.Attributes = attributes.CastOrIgnore(GetSchemaAttributeTypes())
		f.CreatedAt = timeFromSQL(createdAt)
		f.UpdatedAt = timeFromSQL(updatedAt)
		f.IndexedAt = timeFromSQL(lastIndexedAt)
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
	if !text.IsBlank(string(f.FrontMatter)) {
		frontMatter, err := f.FrontMatter.AsBeautifulYAML()
		sb.WriteString("---\n")
		if err == nil {
			sb.WriteString(frontMatter)
		}
		sb.WriteString("---\n\n")
	}
	sb.WriteString(string(f.Body))
	return sb.String()
}

/* Blob management */

func (f *File) GenerateBlobs() {
	if CurrentConfig().DryRun {
		return
	}

	src := CurrentRepository().GetAbsolutePath(f.RelativePath)
	data, err := os.ReadFile(src)
	if err != nil {
		log.Fatalf("Error reading Markdown file %s: %v", f.RelativePath, err)
	}

	oid := oid.NewFromBytes(data)
	blob := &BlobRef{
		OID:      oid,
		MimeType: medias.MimeType(".gz"),
		Tags:     []string{"original", "markdown"},
	}
	if err := CurrentDB().WriteBlobOnDisk(blob.OID, data); err != nil {
		log.Fatalf("Unable to write blob from file %q: %v", f.RelativePath, err)
	}
	f.BlobRefs = append(f.BlobRefs, blob)
}

/* FileObject interface */

func (f *File) FileRelativePath() string {
	return f.RelativePath
}
func (f *File) FileMTime() time.Time {
	return f.MTime
}
func (f *File) FileSize() int64 {
	return f.Size
}
func (f *File) FileHash() string {
	return f.Hash
}
func (f *File) Blobs() []*BlobRef {
	if len(f.BlobRefs) == 0 {
		f.GenerateBlobs()
	}
	return f.BlobRefs
}
