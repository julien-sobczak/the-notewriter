package core

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

const DefaultEaseFactor = 2.5  // Same as Anki
const MinEaseFactor = 1.3      // Same as Anki
const DefaultFirstInterval = 1 // day

type CardType int

const (
	CardNew        CardType = 0
	CardLearning   CardType = 1
	CardReview     CardType = 2
	CardRelearning CardType = 3
)

type QueueType int

const (
	QueueSuspend  QueueType = -1 // leeches as manual suspension is not supported
	QueueNew      QueueType = 0  // new (never shown)
	QueueLearn    QueueType = 1  // learning/relearning
	QueueReview   QueueType = 2  // review (as for type)
	QueueDayLearn QueueType = 3  // in learning, next review in at least a day after the previous review
)

type Flashcard struct {
	OID oid.OID `yaml:"oid" json:"oid"`

	// Pack file where this object belongs
	PackFileOID oid.OID `yaml:"packfile_oid" json:"packfile_oid"`

	// File
	FileOID oid.OID `yaml:"file_oid" json:"file_oid"`
	File    *File   `yaml:"-" json:"-"` // Lazy-loaded

	// Note representing the flashcard
	NoteOID oid.OID `yaml:"note_oid" json:"note_oid"`
	Note    *Note   `yaml:"-" json:"-"` // Lazy-loaded

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path" json:"relative_path"`

	// The slug of the note (denornalized field)
	Slug string `yaml:"slug" json:"slug"`

	// Short title of the note (denormalized field)
	ShortTitle markdown.Document `yaml:"short_title" json:"short_title"`

	// List of tags
	Tags TagSet `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Fields in Markdown (best for editing)
	Front markdown.Document `yaml:"front" json:"front"`
	Back  markdown.Document `yaml:"back" json:"back"`

	// Timestamps to track changes
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
	IndexedAt time.Time `yaml:"indexed_at,omitempty" json:"indexed_at,omitempty"`

	// SRS
	DueAt     time.Time      `yaml:"-" json:"-"`
	StudiedAt time.Time      `yaml:"-" json:"-"`
	Settings  map[string]any `yaml:"-" json:"-"`
}

type Review struct {
	FlashcardOID oid.OID        `yaml:"flashcard_oid" json:"flashcard_oid"`
	Confidence   int            `yaml:"confidence" json:"confidence"` // Value between 0 and 100
	DurationInMs int            `yaml:"duration_ms" json:"duration_ms"`
	CompletedAt  time.Time      `yaml:"completed_at" json:"completed_at"`
	DueAt        time.Time      `yaml:"due_at" json:"due_at"`
	Algorithm    string         `yaml:"algorithm" json:"algorithm"`
	Settings     map[string]any `yaml:"settings" json:"settings"` // Include algorithm-specific attributes (like the e-factor in SM-2)
}

func NewOrExistingFlashcard(packFile *PackFile, file *File, note *Note, parsedFlashcard *ParsedFlashcard) (*Flashcard, error) {
	// Try to find an existing note (instead of recreating it from scratch after every change)
	existingFlashcard, err := CurrentRepository().FindMatchingFlashcard(note, parsedFlashcard)
	if err != nil {
		return nil, err
	}
	if existingFlashcard != nil {
		existingFlashcard.update(packFile, file, note, parsedFlashcard)
		return existingFlashcard, nil
	}
	return NewFlashcard(packFile, file, note, parsedFlashcard)
}

// NewFlashcard initializes a new flashcard.
func NewFlashcard(packFile *PackFile, file *File, note *Note, parsedFlashcard *ParsedFlashcard) (*Flashcard, error) {
	f := &Flashcard{
		OID: oid.NewFromString(parsedFlashcard.Slug),

		PackFileOID: packFile.OID,

		// File-specific attributes
		FileOID:      file.OID,
		File:         file,
		RelativePath: note.RelativePath,

		// Note-specific attributes
		NoteOID:    note.OID,
		Note:       note,
		Slug:       note.Slug,
		ShortTitle: note.ShortTitle,
		Tags:       note.GetTags(),

		// Flashcard-specific attributes
		Front: parsedFlashcard.Front,
		Back:  parsedFlashcard.Back,

		// SRS-specific attributes
		// Wait for first study to initialize SRS fields

		// Timestamps
		CreatedAt: packFile.CTime,
		UpdatedAt: packFile.CTime,
		IndexedAt: packFile.CTime,
	}

	return f, nil
}

/* Object */

func (f *Flashcard) FileRelativePath() string {
	return f.RelativePath
}

func (f *Flashcard) Kind() string {
	return "flashcard"
}

func (f *Flashcard) UniqueOID() oid.OID {
	return f.OID
}

func (f *Flashcard) ModificationTime() time.Time {
	return f.UpdatedAt
}

func (f *Flashcard) Read(r io.Reader) error {
	return yaml.NewDecoder(r).Decode(f)
}

func (f *Flashcard) Write(w io.Writer) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (f *Flashcard) Links() []*Link {
	return nil
}

func (f Flashcard) String() string {
	return fmt.Sprintf("flashcard %q [%s]", f.ShortTitle, f.OID)
}

/* Format */

func (f *Flashcard) ToYAML() string {
	return ToBeautifulYAML(f)
}

func (f *Flashcard) ToJSON() string {
	return ToBeautifulJSON(f)
}

func (f *Flashcard) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString(string(f.Front))
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(string(f.Back))
	return sb.String()
}

/* Update */

func (f *Flashcard) update(packFile *PackFile, file *File, note *Note, parsedFlashcard *ParsedFlashcard) {
	stale := false

	if f.ShortTitle != note.ShortTitle {
		f.ShortTitle = note.ShortTitle
		stale = true
	}

	if f.FileOID != file.OID {
		f.FileOID = file.OID
		f.File = file
		stale = true
	}

	if f.NoteOID != note.OID {
		f.NoteOID = note.OID
		f.Note = note
		stale = true
	}

	if f.Slug != note.Slug {
		f.Slug = note.Slug
		stale = true
	}

	if !reflect.DeepEqual(f.Tags, note.GetTags()) {
		f.Tags = note.GetTags()
		stale = true
	}

	if f.Front != parsedFlashcard.Front {
		f.Front = parsedFlashcard.Front
		stale = true
	}

	if f.Back != parsedFlashcard.Back {
		f.Back = parsedFlashcard.Back
		stale = true
	}

	f.PackFileOID = packFile.OID
	f.IndexedAt = packFile.CTime

	if stale {
		f.UpdatedAt = packFile.CTime
	}
}

func (f *Flashcard) Save() error {
	CurrentLogger().Debugf("Saving flashcard %s...", f.ShortTitle)
	query := `
		INSERT INTO flashcard(
			oid,
			packfile_oid,
			file_oid,
			note_oid,
			relative_path,
			short_title,
			slug,
			tags,
			front,
			back,
			created_at,
			updated_at,
			indexed_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			file_oid = ?,
			note_oid = ?,
			relative_path = ?,
			short_title = ?,
			slug = ?,
			tags = ?,
			front = ?,
			back = ?,
			updated_at = ?,
			indexed_at = ?
		;
		`

	_, err := CurrentDB().Client().Exec(query,
		// Insert
		f.OID,
		f.PackFileOID,
		f.FileOID,
		f.NoteOID,
		f.RelativePath,
		f.ShortTitle,
		f.Slug,
		strings.Join(f.Tags, ","),
		f.Front,
		f.Back,
		timeToSQL(f.CreatedAt),
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.IndexedAt),
		// Update
		f.PackFileOID,
		f.FileOID,
		f.NoteOID,
		f.RelativePath,
		f.ShortTitle,
		f.Slug,
		strings.Join(f.Tags, ","),
		f.Front,
		f.Back,
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.IndexedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (f *Flashcard) SaveMetadata() error {
	CurrentLogger().Debugf("Saving flashcard %s...", f.ShortTitle)
	query := `
		UPDATE flashcard
		SET
			due_at = ?,
			studied_at = ?,
			settings = ?
		WHERE oid = ?
		;
		`

	settingsJSON, err := json.MarshalIndent(f.Settings, "", "  ")
	if err != nil {
		return err
	}

	_, err = CurrentDB().Client().Exec(query,
		timeToSQL(f.DueAt),
		timeToSQL(f.StudiedAt),
		settingsJSON,
		f.OID,
	)
	if err != nil {
		return err
	}

	return nil
}

func (f *Flashcard) Delete() error {
	CurrentLogger().Debugf("Deleting flashcard %s...", f.ShortTitle)
	query := `DELETE FROM flashcard WHERE oid = ? AND packfile_oid = ?;`
	_, err := CurrentDB().Client().Exec(query, f.OID, f.PackFileOID)
	return err
}

func SettingsJSON(settings map[string]any) (string, error) {
	var buf bytes.Buffer
	bufEncoder := json.NewEncoder(&buf)
	err := bufEncoder.Encode(settings)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

/* SQL Queries */

// CountFlashcards returns the total number of flashcards.
func (r *Repository) CountFlashcards() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM flashcard`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) FindMatchingFlashcard(note *Note, parsedFlashcard *ParsedFlashcard) (*Flashcard, error) {
	// Search by slug
	flashcard, err := r.LoadFlashcardBySlug(parsedFlashcard.Slug)
	if err != nil {
		return nil, err
	}
	if flashcard != nil {
		return flashcard, nil
	}

	// Search by note OID
	flashcard, err = r.LoadFlashcardByNoteOID(note.OID)
	if err != nil {
		log.Fatal(err)
	}
	if flashcard != nil {
		return flashcard, nil
	}

	return nil, nil
}

func (r *Repository) LoadFlashcardByOID(oid oid.OID) (*Flashcard, error) {
	return QueryFlashcard(CurrentDB().Client(), `WHERE oid = ?`, oid)
}

func (r *Repository) LoadFlashcardBySlug(slug string) (*Flashcard, error) {
	return QueryFlashcard(CurrentDB().Client(), `WHERE slug = ?`, slug)
}

func (r *Repository) LoadFlashcardByNoteOID(noteID oid.OID) (*Flashcard, error) {
	return QueryFlashcard(CurrentDB().Client(), `WHERE note_oid = ?`, noteID)
}

func (r *Repository) LoadFlashcards() ([]*Flashcard, error) {
	return QueryFlashcards(CurrentDB().Client(), ``)
}

func (r *Repository) FindFlashcardByShortTitle(shortTitle string) (*Flashcard, error) {
	return QueryFlashcard(CurrentDB().Client(), `WHERE short_title = ?`, shortTitle)
}

func (r *Repository) FindFlashcardByHash(hash string) (*Flashcard, error) {
	return QueryFlashcard(CurrentDB().Client(), `WHERE hash = ?`, hash)
}

/* SQL Helpers */

func QueryFlashcard(db SQLClient, whereClause string, args ...any) (*Flashcard, error) {
	var f Flashcard
	var tagsRaw string
	var settingsRaw sql.NullString
	var dueAt sql.NullString
	var studiedAt sql.NullString
	var createdAt string
	var updatedAt string
	var lastIndexedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			file_oid,
			note_oid,
			relative_path,
			short_title,
			slug,
			tags,
			front,
			back,
			due_at,
			studied_at,
			settings,
			created_at,
			updated_at,
			indexed_at
		FROM flashcard
		%s;`, whereClause), args...).
		Scan(
			&f.OID,
			&f.PackFileOID,
			&f.FileOID,
			&f.NoteOID,
			&f.RelativePath,
			&f.ShortTitle,
			&f.Slug,
			&tagsRaw,
			&f.Front,
			&f.Back,
			&dueAt,
			&studiedAt,
			&settingsRaw,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var settings map[string]any
	if settingsRaw.Valid {
		err := yaml.Unmarshal([]byte(settingsRaw.String), &settings)
		if err != nil {
			return nil, err
		}
	}

	if tagsRaw != "" {
		f.Tags = strings.Split(tagsRaw, ",")
	}
	f.Settings = settings
	f.DueAt = timeFromNullableSQL(dueAt)
	f.StudiedAt = timeFromNullableSQL(studiedAt)
	f.CreatedAt = timeFromSQL(createdAt)
	f.UpdatedAt = timeFromSQL(updatedAt)
	f.IndexedAt = timeFromSQL(lastIndexedAt)

	return &f, nil
}

func QueryFlashcards(db SQLClient, whereClause string, args ...any) ([]*Flashcard, error) {
	var flashcards []*Flashcard

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			file_oid,
			note_oid,
			relative_path,
			short_title,
			slug,
			tags,
			front,
			back,
			due_at,
			studied_at,
			settings,
			created_at,
			updated_at,
			indexed_at
		FROM flashcard
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f Flashcard
		var tagsRaw string
		var settingsRaw sql.NullString
		var dueAt sql.NullString
		var studiedAt sql.NullString
		var createdAt string
		var updatedAt string
		var lastIndexedAt string

		err = rows.Scan(
			&f.OID,
			&f.PackFileOID,
			&f.FileOID,
			&f.NoteOID,
			&f.RelativePath,
			&f.ShortTitle,
			&f.Slug,
			&tagsRaw,
			&f.Front,
			&f.Back,
			&dueAt,
			&studiedAt,
			&settingsRaw,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
		)
		if err != nil {
			return nil, err
		}

		var settings map[string]any
		if settingsRaw.Valid {
			err := yaml.Unmarshal([]byte(settingsRaw.String), &settings)
			if err != nil {
				return nil, err
			}
		}

		if tagsRaw != "" {
			f.Tags = strings.Split(tagsRaw, ",")
		}
		f.Settings = settings
		f.DueAt = timeFromNullableSQL(dueAt)
		f.StudiedAt = timeFromNullableSQL(studiedAt)
		f.CreatedAt = timeFromSQL(createdAt)
		f.UpdatedAt = timeFromSQL(updatedAt)
		f.IndexedAt = timeFromSQL(lastIndexedAt)
		flashcards = append(flashcards, &f)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return flashcards, err
}

/* Operations */

type FlashcardReview struct {
	Confidence int            `yaml:"confidence" json:"confidence"` // Value between 0 and 100
	Duration   time.Duration  `duration:"duration" json:"duration"`
	DueAt      time.Time      `yaml:"due_at" json:"due_at"`
	Algorithm  string         `yaml:"algorithm" json:"algorithm"`
	Settings   map[string]any `yaml:"settings" json:"settings"`
}

// Review updates the flashcard following a review.
func (f *Flashcard) Review(studiedAt time.Time, review *FlashcardReview) {
	if !f.StudiedAt.IsZero() && f.StudiedAt.After(studiedAt) {
		// The studiedAt timestamp is in the past. Ignore this review.
		return
	}
	f.StudiedAt = studiedAt
	f.DueAt = review.DueAt
	f.Settings = review.Settings
}
