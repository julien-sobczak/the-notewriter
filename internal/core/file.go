package core

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/internal/helpers"
	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
	"github.com/julien-sobczak/the-notetaker/pkg/text"
	"gopkg.in/yaml.v3"
)

// Default indentation in front matter
const Indent int = 2

type Attribute struct {
	Key   string
	Value interface{}
}

type File struct {
	// A unique identifier among all files
	OID string `yaml:"oid"`

	// A relative path to the collection directory
	RelativePath string `yaml:"relative_path"`
	// The full wikilink to this file (without the extension)
	Wikilink string `yaml:"wikilink"`

	// The FrontMatter for the note file
	frontMatter *yaml.Node `yaml:"front_matter"`

	Content     string  `yaml:"content"`
	ContentLine int     `yaml:"content_line"`
	notes       []*Note `yaml:"-"`

	// Permission of the file (required to save back)
	Mode fs.FileMode `yaml:"mode"`
	// Size of the file (can be useful to detect changes)
	Size int64 `yaml:"size"`
	// Hash of the content (can be useful to detect changes too)
	Hash string `yaml:"hash"`
	// Content last modification date
	MTime time.Time `yaml:"mtime"`

	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	DeletedAt     time.Time `yaml:"deleted_at,omitempty"`
	LastCheckedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}

func NewOrExistingFile(path string) (*File, error) {
	relpath, err := CurrentCollection().GetFileRelativePath(path)
	if err != nil {
		log.Fatal(err)
	}
	existingFile, err := LoadFileByPath(relpath)
	if err != nil {
		log.Fatal(err)
	}

	if existingFile != nil {
		existingFile.Update()
		return existingFile, nil
	}

	return NewFileFromPath(path)
}

/* Creation */

func NewEmptyFile(name string) *File {
	return &File{
		OID:          NewOID(),
		stale:        true,
		new:          true,
		Wikilink:     name,
		RelativePath: name,
	}
}

func NewFileFromAttributes(name string, attributes []Attribute) *File {
	file := NewEmptyFile(name)
	for _, attribute := range attributes {
		file.SetAttribute(attribute.Key, attribute.Value)
	}
	return file
}

func NewFileFromPath(filepath string) (*File, error) {
	parsedFile, err := ParseFile(filepath)
	if err != nil {
		return nil, err
	}

	file := &File{
		OID:          NewOID(),
		RelativePath: parsedFile.RelativePath,
		Wikilink:     text.TrimExtension(parsedFile.RelativePath),
		Mode:         parsedFile.LStat.Mode(),
		Size:         parsedFile.LStat.Size(),
		Hash:         helpers.Hash(parsedFile.Bytes),
		MTime:        parsedFile.LStat.ModTime(),
		Content:      parsedFile.Body,
		ContentLine:  parsedFile.BodyLine,
		CreatedAt:    clock.Now(),
		UpdatedAt:    clock.Now(),
		stale:        true,
		new:          true,
	}
	if parsedFile.FrontMatter.Kind > 0 { // Happen when no Front Matter is present
		file.frontMatter = parsedFile.FrontMatter.Content[0]
	}

	return file, nil
}

/* Object */

func (f *File) Kind() string {
	return "file"
}

func (f *File) UniqueOID() string {
	return f.OID
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

func (f *File) SubObjects() []StatefulObject {
	var objs []StatefulObject
	for _, object := range f.GetNotes() {
		objs = append(objs, object)
		objs = append(objs, object.SubObjects()...)
	}
	for _, object := range f.GetFlashcards() {
		objs = append(objs, object)
		objs = append(objs, object.SubObjects()...)
	}
	// Medias are already saved through files
	// for _, object := range f.GetMedias() {
	// 	objs = append(objs, object)
	// 	objs = append(objs, object.SubObjects()...)
	// }
	return objs
}

func (f *File) Blobs() []*BlobRef {
	// Use Media.Blobs() instead
	return nil
}

func (f File) String() string {
	return fmt.Sprintf("file %q [%s]", f.RelativePath, f.OID)
}

/* Update */

func (f *File) Update() error {
	abspath := CurrentCollection().GetAbsolutePath(f.RelativePath)
	fileInfo, err := os.Lstat(abspath) // NB: os.Stat follows symlinks
	if err != nil {
		return err
	}

	if f.MTime == fileInfo.ModTime() && f.Size == fileInfo.Size() {
		// No file change
		return nil
	}

	f.stale = true

	parsedFile, err := ParseFile(abspath)
	if err != nil {
		return err
	}

	f.Mode = fileInfo.Mode()
	f.Size = fileInfo.Size()
	f.Hash = helpers.Hash(parsedFile.Bytes)
	if parsedFile.FrontMatter.Kind > 0 {
		f.frontMatter = parsedFile.FrontMatter
	} else {
		f.frontMatter = nil
	}
	f.MTime = fileInfo.ModTime()
	f.Content = parsedFile.Body
	f.ContentLine = parsedFile.BodyLine

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

// FrontMatterString formats the current attributes to the YAML front matter format.
func (f *File) FrontMatterString() (string, error) {
	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(Indent)
	err := bufEncoder.Encode(f.frontMatter)
	if err != nil {
		return "", err
	}
	return CompactYAML(buf.String()), nil
}

// GetAttributes parses the front matter to extract typed attributes.
func (f *File) GetAttributes() map[string]interface{} {
	if f.frontMatter == nil {
		return nil
	}

	result := make(map[string]interface{})
	i := 0
	for i < len(f.frontMatter.Content)-1 {
		keyNode := f.frontMatter.Content[i]
		valueNode := f.frontMatter.Content[i+1]
		result[keyNode.Value] = toSafeYAMLValue(valueNode)
		i += 2
	}

	return result
}

// GetAttribute extracts a single attribute value at the top.
func (f *File) GetAttribute(key string) interface{} {
	if f.frontMatter == nil {
		return nil
	}
	i := 0
	for i < len(f.frontMatter.Content)-1 {
		keyNode := f.frontMatter.Content[i]
		valueNode := f.frontMatter.Content[i+1]
		i += 2
		if keyNode.Value == key {
			return toSafeYAMLValue(valueNode)
		}
	}

	// Not found
	return nil
}

// SetAttribute overrides or defines a single attribute.
func (f *File) SetAttribute(key string, value interface{}) {
	if f.frontMatter == nil {
		var frontMatterContent []*yaml.Node
		f.frontMatter = &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: frontMatterContent,
		}
	}

	found := false
	for i := 0; i < len(f.frontMatter.Content)/2; i++ {
		keyNode := f.frontMatter.Content[i*2]
		valueNode := f.frontMatter.Content[i*2+1]
		if keyNode.Value != key {
			continue
		}

		found = true

		newValueNode := toSafeYAMLNode(value)
		if newValueNode.Kind == yaml.ScalarNode {
			valueNode.Value = newValueNode.Value
		} else if newValueNode.Kind == yaml.DocumentNode {
			valueNode.Content = newValueNode.Content[0].Content
		} else {
			valueNode.Content = newValueNode.Content
		}
	}

	if !found {
		f.frontMatter.Content = append(f.frontMatter.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
		})
		newValueNode := toSafeYAMLNode(value)
		switch newValueNode.Kind {
		case yaml.DocumentNode:
			f.frontMatter.Content = append(f.frontMatter.Content, newValueNode.Content[0])
		case yaml.ScalarNode:
			f.frontMatter.Content = append(f.frontMatter.Content, newValueNode)
		default:
			fmt.Printf("Unexcepted type %v\n", newValueNode.Kind)
			os.Exit(1)
		}
	}
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

/* Content */

func (f *File) GetNotes() []*Note {
	if f.notes != nil {
		return f.notes
	}

	parsedNotes := ParseNotes(f.Content)

	if len(parsedNotes) == 0 {
		return nil
	}

	// All notes collected until now
	var notes []*Note
	for i, currentNote := range parsedNotes {
		parentNoteIndex := -1
		for j, prevNote := range parsedNotes[0:i] {
			if prevNote.Level == currentNote.Level-1 {
				parentNoteIndex = j
			}
		}

		noteLine := f.ContentLine + currentNote.Line - 1
		note := NewOrExistingNote(f, currentNote.LongTitle, currentNote.Body, noteLine)
		if parentNoteIndex != -1 {
			note.ParentNote = notes[parentNoteIndex]
		}
		notes = append(notes, note)
	}

	if len(notes) > 0 {
		f.notes = notes
	}
	return f.notes
}

// ParsedNote represents a single raw note inside a file.
type ParsedNote struct {
	Level      int
	Kind       NoteKind
	LongTitle  string
	ShortTitle string
	Line       int
	Body       string
}

func ParseNotes(fileBody string) []*ParsedNote {
	type Section struct {
		level      int
		kind       NoteKind
		longTitle  string
		shortTitle string
		lineNumber int
	}
	var sections []*Section

	// Extract all sections
	lines := strings.Split(fileBody, "\n")

	// Check if the file contains typed notes.
	// If so, it means the top heading (= the title of file) does not represent a free note.
	// Otherwise, we will add this top heading as a standalone note.
	ignoreTopHeading := false
	for _, line := range lines {
		if ok, longTitle, level := markdown.IsHeading(line); ok {
			if ok, kind, _ := isSupportedNote(longTitle); ok {
				if level != 1 && kind != KindFree {
					ignoreTopHeading = true
					break
				}
			}
		}
	}

	// Current line number during the parsing
	var lineNumber int
	insideTypedNote := false
	for _, line := range lines {
		lineNumber++
		if ok, longTitle, level := markdown.IsHeading(line); ok {
			if level == 1 && ignoreTopHeading {
				continue
			}
			lastLevel := 0
			if len(sections) > 0 {
				lastLevel = sections[len(sections)-1].level
			}
			if level <= lastLevel {
				insideTypedNote = false
			}
			ok, kind, shortTitle := isSupportedNote(longTitle)
			if ok {
				sections = append(sections, &Section{
					level:      level,
					kind:       kind,
					longTitle:  longTitle,
					shortTitle: shortTitle,
					lineNumber: lineNumber,
				})
				insideTypedNote = true
			} else { // block inside a note or a free note?
				if !insideTypedNote { // new free note
					sections = append(sections, &Section{
						level:      level,
						kind:       KindFree,
						longTitle:  longTitle,
						shortTitle: shortTitle,
						lineNumber: lineNumber,
					})
				}
			}
		}
	}

	// Iterate over sections and use line numbers to split the raw content into notes
	if len(sections) == 0 {
		return nil
	}

	// All notes collected until now
	var notes []*ParsedNote
	for i, section := range sections {
		var nextSection *Section
		if i < len(sections)-1 {
			nextSection = sections[i+1]
		}

		lineStart := section.lineNumber + 1
		lineEnd := -1 // EOF
		if nextSection != nil {
			lineEnd = nextSection.lineNumber - 1
		}

		noteContent := text.ExtractLines(fileBody, lineStart, lineEnd)
		notes = append(notes, &ParsedNote{
			Level:      section.level,
			Kind:       section.kind,
			LongTitle:  section.longTitle,
			ShortTitle: section.shortTitle,
			Line:       section.lineNumber,
			Body:       noteContent,
		})
	}

	return notes

}

// FindNoteByKindAndShortTitle searches for a given note based on its kind and title.
func (f *File) FindNoteByKindAndShortTitle(kind NoteKind, shortTitle string) *Note {
	for _, note := range f.GetNotes() {
		if note.NoteKind == kind && note.ShortTitle == shortTitle {
			return note
		}
	}
	return nil
}

// FindFlashcardByTitle searches for a given flashcard based on its title.
func (f *File) FindFlashcardByTitle(shortTitle string) *Flashcard {
	for _, flashcard := range f.GetFlashcards() {
		if flashcard.ShortTitle == shortTitle {
			return flashcard
		}
	}
	return nil
}

// GetFlashcards extracts flashcards from the file.
func (f *File) GetFlashcards() []*Flashcard {
	var flashcards []*Flashcard
	for _, note := range f.GetNotes() {
		if note.NoteKind != KindFlashcard {
			continue
		}

		flashcard := NewOrExistingFlashcard(f, note)
		flashcards = append(flashcards, flashcard)
	}
	return flashcards
}

// GetMedias extracts medias from the file.
func (f *File) GetMedias() []*Media {
	return extractMediasFromMarkdown(f.RelativePath, f.Content)
}

/* Parsing */

type ParsedFile struct {
	// The paths to the file
	AbsolutePath string
	RelativePath string

	// Stat
	Stat  fs.FileInfo
	LStat fs.FileInfo

	// The raw file bytes
	Bytes []byte

	// The YAML Front Matter
	FrontMatter *yaml.Node

	// The content excluding the front matter
	Body     string
	BodyLine int
}

func ParseFile(filepath string) (*ParsedFile, error) {
	relativePath, err := CurrentCollection().GetFileRelativePath(filepath)
	if err != nil {
		return nil, err
	}
	absolutePath := CurrentCollection().GetAbsolutePath(relativePath)

	lstat, err := os.Lstat(absolutePath)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(absolutePath)
	if err != nil {
		return nil, err
	}

	contentBytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var rawFrontMatter bytes.Buffer
	var rawContent bytes.Buffer
	frontMatterStarted := false
	frontMatterEnded := false
	bodyStarted := false
	bodyStartLineNumber := 0
	for i, line := range strings.Split(strings.TrimSuffix(string(contentBytes), "\n"), "\n") {
		if strings.HasPrefix(line, "---") {
			if bodyStarted {
				// Flashcard Front/Back line separator
				rawContent.WriteString(line)
				rawContent.WriteString("\n")
			} else if !frontMatterStarted {
				frontMatterStarted = true
			} else if !frontMatterEnded {
				frontMatterEnded = true
			}
			continue
		}

		if frontMatterStarted && !frontMatterEnded {
			rawFrontMatter.WriteString(line)
			rawFrontMatter.WriteString("\n")
		} else {
			if !text.IsBlank(line) && !bodyStarted {
				bodyStarted = true
				bodyStartLineNumber = i + 1
			}
			rawContent.WriteString(line)
			rawContent.WriteString("\n")
		}
	}

	var frontMatter yaml.Node
	err = yaml.Unmarshal(rawFrontMatter.Bytes(), &frontMatter)
	if err != nil {
		return nil, err
	}

	return &ParsedFile{
		AbsolutePath: absolutePath,
		RelativePath: relativePath,
		Stat:         stat,
		LStat:        lstat,
		Bytes:        contentBytes,
		FrontMatter:  &frontMatter,
		Body:         strings.TrimSpace(rawContent.String()),
		BodyLine:     bodyStartLineNumber,
	}, nil
}

/* Data Management */

func (f *File) SaveOnDisk() error {
	// Persist to disk
	frontMatter, err := f.FrontMatterString()
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(frontMatter)
	sb.WriteString("---\n")
	sb.WriteString(f.Content)

	if f.RelativePath == "" {
		return errors.New("unable to save file as no path is defined")
	}
	rawContent := []byte(sb.String())
	absolutePath := CurrentCollection().GetAbsolutePath(f.RelativePath)
	os.WriteFile(absolutePath, rawContent, f.Mode)

	// Refresh file-specific attributes
	stat, err := os.Lstat(absolutePath)
	if err != nil {
		return err
	}
	f.Size = stat.Size()
	f.Mode = stat.Mode()
	f.Hash = helpers.Hash(rawContent)

	return nil
}

func (f *File) Check() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = f.CheckWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil

}

func (f *File) CheckWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Checking file %s...", f.RelativePath)
	f.LastCheckedAt = clock.Now()
	query := `
		UPDATE file
		SET last_checked_at = ?
		WHERE oid = ?;`
	if _, err := tx.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}
	query = `
		UPDATE note
		SET last_checked_at = ?
		WHERE file_oid = ?;`
	if _, err := tx.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}
	query = `
		UPDATE flashcard
		SET last_checked_at = ?
		WHERE file_oid = ?;`
	if _, err := tx.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}
	query = `
		UPDATE reminder
		SET last_checked_at = ?
		WHERE file_oid = ?;`
	if _, err := tx.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}

	return nil
}

func (f *File) Save(tx *sql.Tx) error {
	var err error
	f.UpdatedAt = clock.Now()
	f.LastCheckedAt = clock.Now()
	switch f.State() {
	case Added:
		err = f.InsertWithTx(tx)
	case Modified:
		err = f.UpdateWithTx(tx)
	case Deleted:
		err = f.DeleteWithTx(tx)
	default:
		err = f.CheckWithTx(tx)
	}
	if err != nil {
		return err
	}
	f.new = false
	f.stale = false
	return nil
}

func (f *File) InsertWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Inserting file %s...", f.RelativePath)
	query := `
		INSERT INTO file(
			oid,
			relative_path,
			wikilink,
			front_matter,
			content,
			content_line,
			created_at,
			updated_at,
			last_checked_at,
			mtime,
			size,
			hashsum,
			mode
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	frontMatter, err := f.FrontMatterString()
	if err != nil {
		return err
	}
	_, err = tx.Exec(query,
		f.OID,
		f.RelativePath,
		f.Wikilink,
		frontMatter,
		f.Content,
		f.ContentLine,
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

func (f *File) UpdateWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Updating file %s...", f.RelativePath)
	query := `
		UPDATE file
		SET
			relative_path = ?,
			wikilink = ?,
			front_matter = ?,
			content = ?,
			content_line = ?,
			updated_at = ?,
			last_checked_at = ?,
			mtime = ?,
			size = ?,
			hashsum = ?,
			mode = ?
		WHERE oid = ?;
	`
	frontMatter, err := f.FrontMatterString()
	if err != nil {
		return err
	}
	_, err = tx.Exec(query,
		f.RelativePath,
		f.Wikilink,
		frontMatter,
		f.Content,
		f.ContentLine,
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
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = f.DeleteWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (f *File) DeleteWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Deleting file %s...", f.RelativePath)
	query := `DELETE FROM file WHERE oid = ?;`
	_, err := tx.Exec(query, f.OID)
	return err
}

func LoadFileByPath(relativePath string) (*File, error) {
	return QueryFile(`WHERE relative_path = ?`, relativePath)
}

func LoadFileByOID(oid string) (*File, error) {
	return QueryFile(`WHERE oid = ?`, oid)
}

func LoadFilesByRelativePathPrefix(relativePathPrefix string) ([]*File, error) {
	return QueryFiles(`WHERE relative_path LIKE ?`, relativePathPrefix+"%")
}

func FindFilesByWikilink(wikilink string) ([]*File, error) {
	return QueryFiles(`WHERE wikilink LIKE ?`, "%"+wikilink)
}

func FindFilesLastCheckedBefore(point time.Time, path string) ([]*File, error) {
	return QueryFiles(`WHERE last_checked_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

// CountFiles returns the total number of files.
func CountFiles() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM file`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

/* SQL Helpers */

func QueryFile(whereClause string, args ...any) (*File, error) {
	db := CurrentDB().Client()

	var f File
	var rawFrontMatter string
	var createdAt string
	var updatedAt string
	var lastCheckedAt string
	var mTime string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			relative_path,
			wikilink,
			front_matter,
			content,
			content_line,
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
			&f.RelativePath,
			&f.Wikilink,
			&rawFrontMatter,
			&f.Content,
			&f.ContentLine,
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

	var frontMatter yaml.Node
	err := yaml.Unmarshal([]byte(rawFrontMatter), &frontMatter)
	if err != nil {
		return nil, err
	}

	if frontMatter.Kind > 0 { // Happen when no Front Matter is present
		f.frontMatter = frontMatter.Content[0]
	}
	f.CreatedAt = timeFromSQL(createdAt)
	f.UpdatedAt = timeFromSQL(updatedAt)
	f.LastCheckedAt = timeFromSQL(lastCheckedAt)
	f.MTime = timeFromSQL(mTime)

	return &f, nil
}

func QueryFiles(whereClause string, args ...any) ([]*File, error) {
	db := CurrentDB().Client()

	var files []*File

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			relative_path,
			wikilink,
			front_matter,
			content,
			content_line,
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
		var rawFrontMatter string
		var createdAt string
		var updatedAt string
		var lastCheckedAt string
		var mTime string

		err = rows.Scan(
			&f.OID,
			&f.RelativePath,
			&f.Wikilink,
			&rawFrontMatter,
			&f.Content,
			&f.ContentLine,
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

		var frontMatter yaml.Node
		err := yaml.Unmarshal([]byte(rawFrontMatter), &frontMatter)
		if err != nil {
			return nil, err
		}

		if frontMatter.Kind > 0 { // Happen when no Front Matter is present
			f.frontMatter = frontMatter.Content[0]
		}
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
