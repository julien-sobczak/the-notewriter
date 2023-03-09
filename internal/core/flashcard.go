package core

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
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
	OID string `yaml:"oid"`

	// Short title of the note (denormalized field)
	ShortTitle string `yaml:"short_title"`

	// File
	FileOID string `yaml:"file_oid"`
	File    *File  `yaml:"-"` // Lazy-loaded

	// Note representing the flashcard
	NoteOID string `yaml:"note_oid"`
	Note    *Note  `yaml:"-"` // Lazy-loaded

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path"`

	// List of tags
	Tags []string `yaml:"tags,omitempty"`

	// 0=new, 1=learning, 2=review, 3=relearning
	Type CardType `yaml:"type"`

	// Queue types
	Queue QueueType `yaml:"queue"`

	// Due is used differently for different card types:
	//   - new: note id or random int
	//   - due: integer day, relative to the collection's creation time
	//   - learning: integer timestamp in second
	Due int `yaml:"due"`

	// The interval. Negative = seconds, positive = days
	Interval int `yaml:"interval"`

	// The ease factor in permille (ex: 2500 = the interval will be multiplied by 2.5 the next time you press "Good").
	EaseFactor int `yaml:"ease_factor"`

	// Number of reviews
	Repetitions int `yaml:"repetitions"`

	// The number of times the card went from a "was answered correctly" to "was answered incorrectly" state.
	Lapses int `yaml:"lapses"`

	// Of the form a*1000+b, with:
	//   - a the number of reps left today
	//   - b the number of reps left till graduation
	// For example: '2004' means 2 reps left today and 4 reps till graduation
	Left int `yaml:"left"`

	// Fields in Markdown (best for editing)
	FrontMarkdown string `yaml:"front_markdown"`
	BackMarkdown  string `yaml:"back_markdown"`
	// Fields in HTML (best for rendering)
	FrontHTML string `yaml:"front_html"`
	BackHTML  string `yaml:"back_html"`
	// Fields in raw text (best for indexing)
	FrontText string `yaml:"front_text"`
	BackText  string `yaml:"back_text"`

	// Timestamps to track changes
	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	DeletedAt     time.Time `yaml:"-"`
	LastCheckedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}

type Study struct {
	Answers []*Answer
}

type Feedback string

const (
	FeedbackEasy        Feedback = "easy"
	FeedbackGood        Feedback = "easy"
	FeedbackAgain       Feedback = "easy"
	FeedbackHard        Feedback = "easy"
	FeedbackAssimilated Feedback = "assimilated"
)

type Answer struct {
	OID          string
	Feedback     Feedback
	DurationInMs int
	// New EaseFactor? etc.
}

func NewOrExistingFlashcard(file *File, note *Note) *Flashcard {
	if note.new {
		return NewFlashcard(file, note)
	}

	// Flashcard may already exists
	flashcard, err := LoadFlashcardByNoteOID(note.OID)
	if err != nil {
		log.Fatal(err)
	}
	// or not if the note just have been saved now
	if flashcard == nil {
		return NewFlashcard(file, note)
	}
	if note.stale {
		flashcard.Update(file, note)
	}
	return flashcard
}

// NewFlashcard initializes a new flashcard.
func NewFlashcard(file *File, note *Note) *Flashcard {

	frontMarkdown, backMarkdown := splitFrontBack(note.ContentMarkdown)
	// FIXME if front => invalid flashcard (lint)

	f := &Flashcard{
		OID:          NewOID(),
		ShortTitle:   note.ShortTitle,
		FileOID:      file.OID,
		File:         file,
		NoteOID:      note.OID,
		Note:         note,
		RelativePath: note.RelativePath,
		Tags:         note.GetTags(),

		// SRS
		Type:        CardNew,
		Queue:       QueueNew,
		Due:         0,
		Interval:    DefaultFirstInterval,
		EaseFactor:  DefaultEaseFactor * 1000,
		Repetitions: 0, // never reviewed
		Lapses:      0, // never forgotten
		Left:        0,

		// Timestamps
		CreatedAt: clock.Now(),
		UpdatedAt: clock.Now(),

		new:   true,
		stale: true,
	}

	f.updateContent(frontMarkdown, backMarkdown)

	return f
}

/* Object */

func (f *Flashcard) Kind() string {
	return "flashcard"
}

func (f *Flashcard) UniqueOID() string {
	return f.OID
}

func (f *Flashcard) ModificationTime() time.Time {
	return f.UpdatedAt
}

func (f *Flashcard) State() State {
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

func (f *Flashcard) ForceState(state State) {
	switch state {
	case Added:
		f.new = true
	case Deleted:
		f.DeletedAt = clock.Now()
	}
	f.stale = true
}

func (f *Flashcard) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(f)
	if err != nil {
		return err
	}
	return nil
}

func (f *Flashcard) Write(w io.Writer) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (f *Flashcard) SubObjects() []StatefulObject {
	return nil
}

func (f *Flashcard) Blobs() []BlobRef {
	// Use Media.Blobs() instead
	return nil
}

func (f Flashcard) String() string {
	return fmt.Sprintf("flashcard %q [%s]", f.ShortTitle, f.OID)
}

/* Update */

func (f *Flashcard) updateContent(frontMarkdown, backMarkdown string) {
	f.FrontMarkdown = frontMarkdown
	f.BackMarkdown = backMarkdown
	f.FrontHTML = markdown.ToHTML(frontMarkdown)
	f.BackHTML = markdown.ToHTML(backMarkdown)
	f.FrontText = markdown.ToText(frontMarkdown)
	f.BackText = markdown.ToText(backMarkdown)
}

func (f *Flashcard) Update(file *File, note *Note) {
	if f.ShortTitle != note.ShortTitle {
		f.ShortTitle = note.ShortTitle
		f.stale = true
	}

	if f.FileOID != file.OID {
		f.FileOID = file.OID
		f.File = file
		f.stale = true
	}

	if f.NoteOID != note.OID {
		f.NoteOID = note.OID
		f.Note = note
		f.stale = true
	}

	if !reflect.DeepEqual(f.Tags, note.GetTags()) {
		f.Tags = note.GetTags()
		f.stale = true
	}

	frontMarkdown, backMarkdown := splitFrontBack(note.ContentMarkdown)
	if f.FrontMarkdown != frontMarkdown || f.BackMarkdown != backMarkdown {
		f.updateContent(frontMarkdown, backMarkdown)
		f.stale = true
	}
}

/* State Management */

func (f *Flashcard) New() bool {
	return f.new
}

func (f *Flashcard) Updated() bool {
	return f.stale
}

/* Parsing */

func splitFrontBack(content string) (string, string) {
	front := true
	var frontContent bytes.Buffer
	var backContent bytes.Buffer
	for _, line := range strings.Split(content, "\n") {
		if line == "---" {
			front = false
			continue
		}
		if front {
			frontContent.WriteString(line)
			frontContent.WriteString("\n")
		} else {
			backContent.WriteString(line)
			backContent.WriteString("\n")
		}
	}
	return strings.TrimSpace(frontContent.String()), strings.TrimSpace(backContent.String())
}

// GetMedias extracts medias from the flashcard.
func (f *Flashcard) GetMedias() []*Media {
	return extractMediasFromMarkdown(f.File.RelativePath, f.FrontMarkdown+f.BackMarkdown)
}

func (f *Flashcard) Check() error {
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

func (f *Flashcard) CheckWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Checking flashcard %s...", f.ShortTitle)
	f.LastCheckedAt = clock.Now()
	query := `
		UPDATE flashcard
		SET last_checked_at = ?
		WHERE oid = ?;`
	_, err := tx.Exec(query,
		timeToSQL(f.LastCheckedAt),
		f.OID,
	)

	return err
}

func (f *Flashcard) Save(tx *sql.Tx) error {
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

func (f *Flashcard) InsertWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Inserting flashcard %s...", f.ShortTitle)
	query := `
		INSERT INTO flashcard(
			oid,
			file_oid,
			note_oid,
			relative_path,
			short_title,
			tags,
			"type",
			queue,
			due,
			ivl,
			ease_factor,
			repetitions,
			lapses,
			left,
			front_markdown,
			back_markdown,
			front_html,
			back_html,
			front_text,
			back_text,
			created_at,
			updated_at,
			last_checked_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`
	_, err := tx.Exec(query,
		f.OID,
		f.FileOID,
		f.NoteOID,
		f.RelativePath,
		f.ShortTitle,
		strings.Join(f.Tags, ","),
		f.Type,
		f.Queue,
		f.Due,
		f.Interval,
		f.EaseFactor,
		f.Repetitions,
		f.Lapses,
		f.Left,
		f.FrontMarkdown,
		f.BackMarkdown,
		f.FrontHTML,
		f.BackHTML,
		f.FrontText,
		f.BackText,
		timeToSQL(f.CreatedAt),
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.LastCheckedAt))
	if err != nil {
		return err
	}

	return nil
}

func (f *Flashcard) UpdateWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Updating flashcard %s...", f.ShortTitle)
	query := `
		UPDATE flashcard
		SET
			file_oid = ?,
			note_oid = ?,
			relative_path = ?,
			short_title = ?,
			tags = ?,
			"type" = ?,
			queue = ?,
			due = ?,
			ivl = ?,
			ease_factor = ?,
			repetitions = ?,
			lapses = ?,
			left = ?,
			front_markdown = ?,
			back_markdown = ?,
			front_html = ?,
			back_html = ?,
			front_text = ?,
			back_text = ?,
			updated_at = ?,
			last_checked_at = ?
		WHERE oid = ?;
		`
	_, err := tx.Exec(query,
		f.FileOID,
		f.NoteOID,
		f.RelativePath,
		f.ShortTitle,
		strings.Join(f.Tags, ","),
		f.Type,
		f.Queue,
		f.Due,
		f.Interval,
		f.EaseFactor,
		f.Repetitions,
		f.Lapses,
		f.Left,
		f.FrontMarkdown,
		f.BackMarkdown,
		f.FrontHTML,
		f.BackHTML,
		f.FrontText,
		f.BackText,
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.LastCheckedAt),
		f.OID)

	return err
}

func (f *Flashcard) Delete() error {
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

func (f *Flashcard) DeleteWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Deleting flashcard %s...", f.ShortTitle)
	query := `DELETE FROM flashcard WHERE oid = ?;`
	_, err := tx.Exec(query, f.OID)
	return err
}

// CountFlashcards returns the total number of flashcards.
func CountFlashcards() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM flashcard`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func LoadFlashcardByOID(oid string) (*Flashcard, error) {
	return QueryFlashcard(`WHERE oid = ?`, oid)
}

func LoadFlashcardByNoteOID(noteID string) (*Flashcard, error) {
	return QueryFlashcard(`WHERE note_oid = ?`, noteID)
}

func FindFlashcardByShortTitle(shortTitle string) (*Flashcard, error) {
	return QueryFlashcard(`WHERE short_title = ?`, shortTitle)
}

func FindFlashcardByHash(hash string) (*Flashcard, error) {
	return QueryFlashcard(`WHERE hash = ?`, hash)
}

func FindFlashcardsLastCheckedBefore(point time.Time, path string) ([]*Flashcard, error) {
	return QueryFlashcards(`WHERE last_checked_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

/* SQL Helpers */

func QueryFlashcard(whereClause string, args ...any) (*Flashcard, error) {
	db := CurrentDB().Client()

	var f Flashcard
	var tagsRaw string
	var createdAt string
	var updatedAt string
	var lastCheckedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			note_oid,
			relative_path,
			short_title,
			tags,
			"type",
			queue,
			due,
			ivl,
			ease_factor,
			repetitions,
			lapses,
			left,
			front_markdown,
			back_markdown,
			front_html,
			back_html,
			front_text,
			back_text,
			created_at,
			updated_at,
			last_checked_at
		FROM flashcard
		%s;`, whereClause), args...).
		Scan(
			&f.OID,
			&f.FileOID,
			&f.NoteOID,
			&f.RelativePath,
			&f.ShortTitle,
			&tagsRaw,
			&f.Type,
			&f.Queue,
			&f.Due,
			&f.Interval,
			&f.EaseFactor,
			&f.Repetitions,
			&f.Lapses,
			&f.Left,
			&f.FrontMarkdown,
			&f.BackMarkdown,
			&f.FrontHTML,
			&f.BackHTML,
			&f.FrontText,
			&f.BackText,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	f.Tags = strings.Split(tagsRaw, ",")
	f.CreatedAt = timeFromSQL(createdAt)
	f.UpdatedAt = timeFromSQL(updatedAt)
	f.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &f, nil
}

func QueryFlashcards(whereClause string, args ...any) ([]*Flashcard, error) {
	db := CurrentDB().Client()

	var flashcards []*Flashcard

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			note_oid,
			relative_path,
			short_title,
			tags,
			"type",
			queue,
			due,
			ivl,
			ease_factor,
			repetitions,
			lapses,
			left,
			front_markdown,
			back_markdown,
			front_html,
			back_html,
			front_text,
			back_text,
			created_at,
			updated_at,
			last_checked_at
		FROM flashcard
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f Flashcard
		var tagsRaw string
		var createdAt string
		var updatedAt string
		var lastCheckedAt string

		err = rows.Scan(
			&f.OID,
			&f.FileOID,
			&f.NoteOID,
			&f.RelativePath,
			&f.ShortTitle,
			&tagsRaw,
			&f.Type,
			&f.Queue,
			&f.Due,
			&f.Interval,
			&f.EaseFactor,
			&f.Repetitions,
			&f.Lapses,
			&f.Left,
			&f.FrontMarkdown,
			&f.BackMarkdown,
			&f.FrontHTML,
			&f.BackHTML,
			&f.FrontText,
			&f.BackText,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		)
		if err != nil {
			return nil, err
		}
		f.Tags = strings.Split(tagsRaw, ",")
		f.CreatedAt = timeFromSQL(createdAt)
		f.UpdatedAt = timeFromSQL(updatedAt)
		f.LastCheckedAt = timeFromSQL(lastCheckedAt)
		flashcards = append(flashcards, &f)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return flashcards, err
}
