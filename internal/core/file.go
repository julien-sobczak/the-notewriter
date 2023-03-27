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
	"reflect"
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

	// Optional parent file (= index.md)
	ParentFileOID string `yaml:"file_oid,omitempty"`
	ParentFile    *File  `yaml:"-"` // Lazy-loaded

	// A relative path to the collection directory
	RelativePath string `yaml:"relative_path"`
	// The full wikilink to this file (without the extension)
	Wikilink string `yaml:"wikilink"`

	// The FrontMatter for the note file
	FrontMatter *yaml.Node `yaml:"front_matter"`

	// Merged attributes
	Attributes map[string]interface{} `yaml:"attributes,omitempty"`

	Body     string  `yaml:"body"`
	BodyLine int     `yaml:"body_line"`
	notes    []*Note `yaml:"-"`

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

func NewOrExistingFile(parent *File, path string) (*File, error) {
	relpath, err := CurrentCollection().GetFileRelativePath(path)
	if err != nil {
		log.Fatal(err)
	}
	existingFile, err := LoadFileByPath(CurrentDB().Client(), relpath)
	if err != nil {
		log.Fatal(err)
	}

	if existingFile != nil {
		existingFile.Update(parent)
		return existingFile, nil
	}

	return NewFileFromPath(parent, path)
}

/* Creation */

func NewEmptyFile(name string) *File {
	return &File{
		OID:          NewOID(),
		stale:        true,
		new:          true,
		Wikilink:     name,
		RelativePath: name,
		Attributes:   make(map[string]interface{}),
	}
}

func NewFileFromAttributes(parent *File, name string, attributes []Attribute) *File {
	file := NewEmptyFile(name)
	if parent != nil {
		file.ParentFile = parent
		file.ParentFileOID = parent.OID
		file.Attributes = parent.GetAttributes()
	}
	for _, attribute := range attributes {
		file.SetAttribute(attribute.Key, attribute.Value)
	}
	return file
}

func NewFileFromPath(parent *File, filepath string) (*File, error) {
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
		Attributes:   make(map[string]interface{}),
		FrontMatter:  parsedFile.FrontMatter,
		Body:         parsedFile.Body,
		BodyLine:     parsedFile.BodyLine,
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
		newAttributes = file.mergeAttributes(parent.GetAttributes(), newAttributes)
	}
	file.Attributes = newAttributes

	return file, nil
}

func (f *File) mergeAttributes(attributes ...map[string]interface{}) map[string]interface{} {
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
	return MergeAttributes(attributes...)
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

func (f *File) Update(parent *File) error {
	abspath := CurrentCollection().GetAbsolutePath(f.RelativePath)
	parsedFile, err := ParseFile(abspath)
	if err != nil {
		return err
	}

	newAttributes := parsedFile.FileAttributes
	if parent != nil {
		newAttributes = f.mergeAttributes(parent.GetAttributes(), f.Attributes)
	}

	// Check if parent attributes have changed
	if !reflect.DeepEqual(newAttributes, f.Attributes) {
		f.stale = true
		f.Attributes = newAttributes
	}

	// Check if local file has changed
	if f.MTime != parsedFile.LStat.ModTime() || f.Size != parsedFile.LStat.Size() {
		// file change
		f.stale = true

		f.Mode = parsedFile.LStat.Mode()
		f.Size = parsedFile.LStat.Size()
		f.Hash = helpers.Hash(parsedFile.Bytes)
		if parsedFile.FrontMatter.Kind > 0 {
			f.FrontMatter = parsedFile.FrontMatter
		} else {
			f.FrontMatter = nil
		}
		f.Attributes = parsedFile.FileAttributes
		if parent != nil {
			f.Attributes = f.mergeAttributes(parent.GetAttributes(), f.Attributes)
		}
		f.MTime = parsedFile.LStat.ModTime()
		f.Body = parsedFile.Body
		f.BodyLine = parsedFile.BodyLine
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

// FrontMatterString formats the current attributes to the YAML front matter format.
func (f *File) FrontMatterString() (string, error) {
	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(Indent)
	err := bufEncoder.Encode(f.FrontMatter)
	if err != nil {
		return "", err
	}
	return CompactYAML(buf.String()), nil
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

// SetAttribute overrides or defines a single attribute.
func (f *File) SetAttribute(key string, value interface{}) {
	if f.FrontMatter == nil {
		var frontMatterContent []*yaml.Node
		f.FrontMatter = &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: frontMatterContent,
		}
	}

	found := false
	for i := 0; i < len(f.FrontMatter.Content)/2; i++ {
		keyNode := f.FrontMatter.Content[i*2]
		valueNode := f.FrontMatter.Content[i*2+1]
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
		f.FrontMatter.Content = append(f.FrontMatter.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
		})
		newValueNode := toSafeYAMLNode(value)
		switch newValueNode.Kind {
		case yaml.DocumentNode:
			f.FrontMatter.Content = append(f.FrontMatter.Content, newValueNode.Content[0])
		case yaml.ScalarNode:
			f.FrontMatter.Content = append(f.FrontMatter.Content, newValueNode)
		default:
			fmt.Printf("Unexcepted type %v\n", newValueNode.Kind)
			os.Exit(1)
		}
	}

	// Don't forget to append in parsed attributes too
	newAttributes := map[string]interface{}{key: value}
	newAttributes = CastAttributes(newAttributes, GetSchemaAttributeTypes())
	f.Attributes = MergeAttributes(f.Attributes, newAttributes)
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

	parsedNotes := ParseNotes(f.Body)

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

		noteLine := f.BodyLine + currentNote.Line - 1
		var parent *Note
		if parentNoteIndex != -1 {
			parent = notes[parentNoteIndex]
		}
		note := NewOrExistingNote(f, parent, currentNote.LongTitle, currentNote.Body, noteLine)
		notes = append(notes, note)
	}

	if len(notes) > 0 {
		f.notes = notes
	}
	return f.notes
}

// ParsedNote represents a single raw note inside a file.
type ParsedNote struct {
	Level          int
	Kind           NoteKind
	LongTitle      string
	ShortTitle     string
	Line           int
	Body           string
	NoteAttributes map[string]interface{}
	NoteTags       []string
}

// ParseNotes extracts the notes from a file body.
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

		tags, attributes := ExtractBlockTagsAndAttributes(noteContent)

		notes = append(notes, &ParsedNote{
			Level:          section.level,
			Kind:           section.kind,
			LongTitle:      section.longTitle,
			ShortTitle:     section.shortTitle,
			Line:           section.lineNumber,
			NoteAttributes: CastAttributes(attributes, GetSchemaAttributeTypes()),
			NoteTags:       tags,
			Body:           noteContent,
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
	return extractMediasFromMarkdown(f.RelativePath, f.Body)
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
	// File attributes extracted from the Front Matter
	FileAttributes map[string]interface{}

	// The body (= content minus the front matter)
	Body     string
	BodyLine int
}

// ParseFile contains the main logic to parse a raw note file.
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

	var frontMatter = new(yaml.Node)
	err = yaml.Unmarshal(rawFrontMatter.Bytes(), frontMatter)
	if err != nil {
		return nil, err
	}
	if frontMatter.Kind > 0 { // Happen when no Front Matter is present
		frontMatter = frontMatter.Content[0]
	}

	var attributes = make(map[string]interface{})
	err = yaml.Unmarshal(rawFrontMatter.Bytes(), attributes)
	if err != nil {
		return nil, err
	}

	// i := 0
	// for i < len(frontMatter.Content)-1 {
	// 	keyNode := frontMatter.Content[i]
	// 	valueNode := frontMatter.Content[i+1]
	// 	valueSafe := toSafeYAMLValue(valueNode)
	// 	attributes[keyNode.Value] = valueSafe
	// 	i += 2
	// }

	return &ParsedFile{
		AbsolutePath:   absolutePath,
		RelativePath:   relativePath,
		Stat:           stat,
		LStat:          lstat,
		Bytes:          contentBytes,
		FrontMatter:    frontMatter,
		FileAttributes: CastAttributes(attributes, GetSchemaAttributeTypes()),
		Body:           strings.TrimSpace(rawContent.String()),
		BodyLine:       bodyStartLineNumber,
	}, nil
}

// Content returns the raw file content.
func (f *ParsedFile) Content() string {
	return string(f.Bytes)
}

// GetFileAttributesProcessed returns attributes with the right type when applicable.
// Run linter first to ensure all attributes are correctly typed.
// FIXME remove
func (f *ParsedFile) GetFileAttributesProcessedTOREMOVE() map[string]interface{} {
	lintFile := CurrentConfig().LintFile
	result := make(map[string]interface{})
	for attributeName, attributeValueRaw := range f.FileAttributes {
		definition := lintFile.GetAttributeDefinition(attributeName, func(schema ConfigLintSchema) bool {
			if schema.Path == "" {
				return true
			}
			return strings.HasPrefix(f.RelativePath, schema.Path)
		})
		if definition == nil {
			// No processing
			result[attributeName] = attributeValueRaw
			continue
		}

		switch definition.Type {
		case "array":
			if !IsArray(attributeValueRaw) {
				result[attributeName] = []interface{}{attributeValueRaw}
			} else {
				// Processing not possible
				result[attributeName] = attributeValueRaw
			}
		case "string":
			if IsPrimitive(attributeValueRaw) {
				typedValue := fmt.Sprintf("%s", attributeValueRaw)
				result[attributeName] = typedValue
			} else {
				// Processing not possible
				result[attributeName] = attributeValueRaw
			}
		case "object":
			// Nothing can be done
			result[attributeName] = attributeValueRaw
		case "number":
			// Nothing can be done
			result[attributeName] = attributeValueRaw
		case "bool":
			// Nothing can be done
			result[attributeName] = attributeValueRaw
		}
	}

	return result
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
	sb.WriteString(f.Body)

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
			file_oid,
			relative_path,
			wikilink,
			front_matter,
			attributes,
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	frontMatter, err := f.FrontMatterString()
	if err != nil {
		return err
	}
	attributesJSON, err := AttributesJSON(f.Attributes)
	if err != nil {
		return err
	}

	_, err = tx.Exec(query,
		f.OID,
		f.ParentFileOID,
		f.RelativePath,
		f.Wikilink,
		frontMatter,
		attributesJSON,
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

func (f *File) UpdateWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Updating file %s...", f.RelativePath)
	query := `
		UPDATE file
		SET
		    file_oid = ?,
			relative_path = ?,
			wikilink = ?,
			front_matter = ?,
			attributes = ?,
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
	frontMatter, err := f.FrontMatterString()
	if err != nil {
		return err
	}
	attributesJSON, err := AttributesJSON(f.Attributes)
	if err != nil {
		return err
	}
	_, err = tx.Exec(query,
		f.ParentFileOID,
		f.RelativePath,
		f.Wikilink,
		frontMatter,
		attributesJSON,
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

func LoadFileByPath(db Queryable, relativePath string) (*File, error) {
	return QueryFile(db, `WHERE relative_path = ?`, relativePath)
}

func LoadFileByOID(db Queryable, oid string) (*File, error) {
	return QueryFile(db, `WHERE oid = ?`, oid)
}

func LoadFilesByRelativePathPrefix(relativePathPrefix string) ([]*File, error) {
	return QueryFiles(CurrentDB().Client(), `WHERE relative_path LIKE ?`, relativePathPrefix+"%")
}

func FindFilesByWikilink(wikilink string) ([]*File, error) {
	return QueryFiles(CurrentDB().Client(), `WHERE wikilink LIKE ?`, "%"+wikilink)
}

func FindFilesLastCheckedBefore(point time.Time, path string) ([]*File, error) {
	return QueryFiles(CurrentDB().Client(), `WHERE last_checked_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
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

type Queryable interface {
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}

func QueryFile(db Queryable, whereClause string, args ...any) (*File, error) {
	var f File
	var rawFrontMatter string
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
			relative_path,
			wikilink,
			front_matter,
			attributes,
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
			&f.RelativePath,
			&f.Wikilink,
			&rawFrontMatter,
			&attributesRaw,
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

	var frontMatter yaml.Node
	err := yaml.Unmarshal([]byte(rawFrontMatter), &frontMatter)
	if err != nil {
		return nil, err
	}
	if frontMatter.Kind > 0 { // Happen when no Front Matter is present
		f.FrontMatter = frontMatter.Content[0]
	}

	attributes, err := UnmarshalAttributes(attributesRaw)
	if err != nil {
		return nil, err
	}

	f.Attributes = attributes
	f.CreatedAt = timeFromSQL(createdAt)
	f.UpdatedAt = timeFromSQL(updatedAt)
	f.LastCheckedAt = timeFromSQL(lastCheckedAt)
	f.MTime = timeFromSQL(mTime)

	return &f, nil
}

func QueryFiles(db Queryable, whereClause string, args ...any) ([]*File, error) {
	var files []*File

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			relative_path,
			wikilink,
			front_matter,
			attributes,
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
		var rawFrontMatter string
		var createdAt string
		var updatedAt string
		var lastCheckedAt string
		var mTime string
		var attributesRaw string

		err = rows.Scan(
			&f.OID,
			&f.ParentFileOID,
			&f.RelativePath,
			&f.Wikilink,
			&rawFrontMatter,
			&attributesRaw,
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

		var frontMatter yaml.Node
		err := yaml.Unmarshal([]byte(rawFrontMatter), &frontMatter)
		if err != nil {
			return nil, err
		}
		if frontMatter.Kind > 0 { // Happen when no Front Matter is present
			f.FrontMatter = frontMatter.Content[0]
		}

		attributes, err := UnmarshalAttributes(attributesRaw)
		if err != nil {
			return nil, err
		}

		f.Attributes = attributes
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
