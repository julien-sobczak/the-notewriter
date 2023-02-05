package core

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

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
	ID int64

	// A relative path to the collection directory
	RelativePath string

	// The FrontMatter for the note file
	frontMatter *yaml.Node

	Content string
	notes   []*Note

	// Permission of the file (required to save back)
	Mode fs.FileMode
	// Size of the file (can be useful to detect changes)
	Size int64
	// Hash of the content (can be useful to detect changes too)
	Hash string
	// Content last modification date
	MTime time.Time

	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     time.Time
	LastCheckedAt time.Time
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

// GetNotes extracts the typed notes present in the file.
func (f *File) GetNotes() []*Note {
	if f.notes != nil {
		return f.notes
	}

	// All typed notes collected until now
	var notes []*Note

	// Current line number during the parsing
	var lineNumber int

	// Keep some information about the current note that
	// will be added when finding the next one (or end of file)
	var currentNote bytes.Buffer
	var currentNoteTitle string
	var currentLineNumber int
	var currentLevel int
	var linesCountInCurrentNote int = -1 // 0 = the title has been found

	// Keep parent notes to create the hierarchy
	lastNotePerLevel := make(map[int]*Note)
	lastNotePerLevel[-1] = nil
	lastNotePerLevel[0] = nil
	lastNotePerLevel[1] = nil
	lastNotePerLevel[2] = nil
	lastNotePerLevel[3] = nil
	lastNotePerLevel[4] = nil
	lastNotePerLevel[5] = nil
	lastNotePerLevel[6] = nil
	lastNotePerLevel[7] = nil

	for _, line := range strings.Split(f.Content, "\n") {
		lineNumber++

		// New section = new potential note?
		if ok, text, level := markdown.IsHeading(line); ok {
			ok, _, _ := isSupportedNote(text)
			if ok || level <= currentLevel {

				// Add previous note
				if linesCountInCurrentNote > 0 {
					note := NewNote(f, currentNoteTitle, currentNote.String(), currentLineNumber)
					note.ParentNote = lastNotePerLevel[currentLevel-1]
					notes = append(notes, note)
					lastNotePerLevel[currentLevel] = note
					// Reset
					currentNote.Reset()
					linesCountInCurrentNote = -1
				}
			}

			if ok {
				// New note
				currentNote.Reset()
				currentLineNumber = lineNumber
				currentNoteTitle = text
				currentLevel = level
				linesCountInCurrentNote = 0
				continue
			}

			// Just a subsection
			if linesCountInCurrentNote >= 0 {
				currentNote.WriteString(line + "\n")
				linesCountInCurrentNote++
			}
		}

		// Just another line in note content
		if linesCountInCurrentNote >= 0 {
			currentNote.WriteString(line + "\n")
			linesCountInCurrentNote++
		}
	}

	// Add last note
	if linesCountInCurrentNote > 0 {
		note := NewNote(f, currentNoteTitle, currentNote.String(), lineNumber)
		note.ParentNote = lastNotePerLevel[currentLevel-1]
		notes = append(notes, note)
	}

	if len(notes) > 0 {
		f.notes = notes
	}
	return f.notes
}

// FindNoteByKindAndShortTitle searches for a given note based on its kind and title.
func (f *File) FindNoteByKindAndShortTitle(kind NoteKind, shortTitle string) *Note {
	for _, note := range f.GetNotes() {
		if note.Kind == kind && note.ShortTitle == shortTitle {
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
		if note.Kind == KindFlashcard {
			flashcard := NewFlashcard(f, note)
			flashcards = append(flashcards, flashcard)
		}
	}
	return flashcards
}

// GetMedias extracts medias from the file.
func (f *File) GetMedias() ([]*Media, error) {
	return extractMediasFromMarkdown(f.RelativePath, f.Content)
}

/* Creation */

func NewEmptyFile() *File {
	return &File{}
}

func NewFileFromAttributes(attributes []Attribute) *File {
	file := &File{}
	for _, attribute := range attributes {
		file.SetAttribute(attribute.Key, attribute.Value)
	}
	return file
}

func NewFileFromPath(filepath string) (*File, error) {
	contentBytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var rawFrontMatter bytes.Buffer
	var rawContent bytes.Buffer
	frontMatterStarted := false
	frontMatterEnded := false
	bodyStarted := false
	for _, line := range strings.Split(strings.TrimSuffix(string(contentBytes), "\n"), "\n") {
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
			if !text.IsBlank(line) {
				bodyStarted = true
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

	stat, err := os.Lstat(filepath)
	if err != nil {
		return nil, err
	}

	relativePath, err := CurrentCollection().GetFileRelativePath(filepath)
	if err != nil {
		return nil, err
	}

	file := &File{
		// We ignore if the file already exists in database
		ID: 0,
		// Reread the file
		RelativePath: relativePath,
		Mode:         stat.Mode(),
		Size:         stat.Size(),
		Hash:         hash(contentBytes),
		MTime:        stat.ModTime(),
		Content:      strings.TrimSpace(rawContent.String()),
	}
	if frontMatter.Kind > 0 { // Happen when no Front Matter is present
		file.frontMatter = frontMatter.Content[0]
	}

	return file, nil
}

/* Data Management */

// hash is an utility to determine a MD5 hash (acceptable as not used for security reasons).
func hash(bytes []byte) string {
	h := md5.New()
	h.Write(bytes)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// hashFromFile reads the file content to determine the hash.
func hashFromFile(path string) (string, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return hash(contentBytes), nil
}

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
	f.Hash = hash(rawContent)

	return nil
}

func (f *File) Save() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = f.SaveWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (f *File) SaveWithTx(tx *sql.Tx) error {
	now := clock.Now()
	f.UpdatedAt = now
	f.LastCheckedAt = now

	if f.ID != 0 {
		if err := f.UpdateWithTx(tx); err != nil {
			return err
		}
	} else {
		f.CreatedAt = now
		if err := f.InsertWithTx(tx); err != nil {
			return err
		}
	}

	// Save the notes
	for _, note := range f.GetNotes() {
		if err := note.SaveWithTx(tx); err != nil {
			return err
		}
	}

	// Ssve the flashcards
	for _, flashcard := range f.GetFlashcards() {
		if err := flashcard.SaveWithTx(tx); err != nil {
			return err
		}
	}

	// Save the medias
	medias, err := f.GetMedias()
	if err != nil {
		return err
	}
	for _, media := range medias {
		if err := media.SaveWithTx(tx); err != nil {
			return err
		}
	}

	return nil
}

func (f *File) InsertWithTx(tx *sql.Tx) error {
	query := `
		INSERT INTO file(
			id,
			relative_path,
			front_matter,
			content,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at,
			mtime,
			size,
			hashsum,
			mode
		)
		VALUES (NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	frontMatter, err := f.FrontMatterString()
	if err != nil {
		return err
	}
	res, err := tx.Exec(query,
		f.RelativePath,
		frontMatter,
		f.Content,
		timeToSQL(f.CreatedAt),
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.DeletedAt),
		timeToSQL(f.LastCheckedAt),
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
		f.Mode,
	)
	if err != nil {
		return err
	}

	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return err
	}
	f.ID = id

	return nil
}

func (f *File) UpdateWithTx(tx *sql.Tx) error {
	query := `
		UPDATE file
		SET
			relative_path = ?,
			front_matter = ?,
			content = ?,
			updated_at = ?,
			deleted_at = ?,
			last_checked_at = ?,
			mtime = ?,
			size = ?,
			hashsum = ?,
			mode = ?
		)
		WHERE id = ?;
	`
	frontMatter, err := f.FrontMatterString()
	if err != nil {
		return err
	}
	_, err = tx.Exec(query,
		f.RelativePath,
		frontMatter,
		f.Content,
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.DeletedAt),
		timeToSQL(f.LastCheckedAt),
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
		f.Mode,
		f.ID,
	)
	return err
}

func LoadFileByPath(relativePath string) (*File, error) {
	return QueryFile(`WHERE relative_path = ?`, relativePath)
}

func LoadFileByID(id int64) (*File, error) {
	return QueryFile(`WHERE id = ?`, id)
}

func LoadFilesByRelativePathPrefix(relativePathPrefix string) ([]*File, error) {
	return QueryFiles(`WHERE relative_path LIKE ?`, relativePathPrefix+"%")
}

// CountFiles returns the total number of files.
func CountFiles() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM file WHERE deleted_at = ''`).Scan(&count); err != nil {
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
	var deletedAt string
	var lastCheckedAt string
	var mTime string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			id,
			relative_path,
			front_matter,
			content,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at,
			mtime,
			size,
			hashsum,
			mode
		FROM file
		%s;`, whereClause), args...).
		Scan(
			&f.ID,
			&f.RelativePath,
			&rawFrontMatter,
			&f.Content,
			&createdAt,
			&updatedAt,
			&deletedAt,
			&lastCheckedAt,
			&mTime,
			&f.Size,
			&f.Hash,
			&f.Mode,
		); err != nil {

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
	f.DeletedAt = timeFromSQL(deletedAt)
	f.LastCheckedAt = timeFromSQL(lastCheckedAt)
	f.MTime = timeFromSQL(mTime)

	return &f, nil
}

func QueryFiles(whereClause string, args ...any) ([]*File, error) {
	db := CurrentDB().Client()

	var files []*File

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			id,
			relative_path,
			front_matter,
			content,
			created_at,
			updated_at,
			deleted_at,
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
		var deletedAt string
		var lastCheckedAt string
		var mTime string

		err = rows.Scan(
			&f.ID,
			&f.RelativePath,
			&rawFrontMatter,
			&f.Content,
			&createdAt,
			&updatedAt,
			&deletedAt,
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
		f.DeletedAt = timeFromSQL(deletedAt)
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
