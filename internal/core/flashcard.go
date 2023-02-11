package core

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
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
	ID int64

	// Short title of the note (denormalized field)
	ShortTitle string

	// File
	FileID int64
	File   *File // Lazy-loaded

	// Note representing the flashcard
	NoteID int64
	Note   *Note // Lazy-loaded

	// List of tags
	Tags []string

	// 0=new, 1=learning, 2=review, 3=relearning
	Type CardType

	// Queue types
	Queue QueueType

	// Due is used differently for different card types:
	//   - new: note id or random int
	//   - due: integer day, relative to the collection's creation time
	//   - learning: integer timestamp in second
	Due int

	// The interval. Negative = seconds, positive = days
	Interval int

	// The ease factor in permille (ex: 2500 = the interval will be multiplied by 2.5 the next time you press "Good").
	EaseFactor int

	// Number of reviews
	Repetitions int

	// The number of times the card went from a "was answered correctly" to "was answered incorrectly" state.
	Lapses int

	// Of the form a*1000+b, with:
	//   - a the number of reps left today
	//   - b the number of reps left till graduation
	// For example: '2004' means 2 reps left today and 4 reps till graduation
	Left int

	// Fields in Markdown (best for editing)
	FrontMarkdown string
	BackMarkdown  string
	// Fields in HTML (best for rendering)
	FrontHTML string
	BackHTML  string
	// Fields in raw text (best for indexing)
	FrontText string
	BackText  string

	// Timestamps to track changes
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     time.Time
	LastCheckedAt time.Time

	stale bool
}

func NewOrExistingFlashcard(file *File, note *Note) *Flashcard {
	if note.ID == 0 {
		return NewFlashcard(file, note)
	}

	// Flashcard may already exists
	flashcard, err := LoadFlashcardByNoteID(note.ID)
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
	// FIXME if front => invalid flashcard

	f := &Flashcard{
		ShortTitle: note.ShortTitle,
		FileID:     file.ID,
		File:       file,
		NoteID:     note.ID,
		Note:       note,
		Tags:       note.GetTags(),

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
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),

		stale: true,
	}

	f.updateContent(frontMarkdown, backMarkdown)

	return f
}

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

	if f.FileID != file.ID {
		f.FileID = file.ID
		f.File = file
		f.stale = true
	}

	if f.NoteID != note.ID {
		f.NoteID = note.ID
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
	return f.ID == 0
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
func (f *Flashcard) GetMedias() ([]*Media, error) {
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
		WHERE id = ?;`
	_, err := tx.Exec(query,
		timeToSQL(f.LastCheckedAt),
		f.ID,
	)

	return err
}

func (f *Flashcard) Save() error {
	if !f.stale {
		return f.Check()
	}

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

	f.stale = false

	return nil
}

func (f *Flashcard) SaveWithTx(tx *sql.Tx) error {
	if !f.stale {
		return f.CheckWithTx(tx)
	}

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

	f.stale = false

	return nil
}

func (f *Flashcard) InsertWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Inserting file %s...", f.ShortTitle)
	query := `
		INSERT INTO flashcard(
			id,
			file_id,
			note_id,
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
			deleted_at,
			last_checked_at
		)
		VALUES (NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`
	res, err := tx.Exec(query,
		f.FileID,
		f.NoteID,
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
		timeToSQL(f.DeletedAt),
		timeToSQL(f.LastCheckedAt))
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

func (f *Flashcard) UpdateWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Updating flashcard %s...", f.ShortTitle)
	query := `
		UPDATE flashcard
		SET
			file_id = ?,
			note_id = ?,
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
			deleted_at = ?,
			last_checked_at = ?
		WHERE id = ?;
		`
	_, err := tx.Exec(query,
		f.FileID,
		f.NoteID,
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
		timeToSQL(f.DeletedAt),
		timeToSQL(f.LastCheckedAt),
		f.ID)

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
	query := `DELETE FROM flashcard WHERE id = ?;`
	_, err := tx.Exec(query, f.ID)
	return err
}

// CountFlashcards returns the total number of flashcards.
func CountFlashcards() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM flashcard WHERE deleted_at = ''`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func LoadFlashcardByID(id int64) (*Flashcard, error) {
	return QueryFlashcard(`WHERE id = ?`, id)
}

func LoadFlashcardByNoteID(noteID int64) (*Flashcard, error) {
	return QueryFlashcard(`WHERE note_id = ?`, noteID)
}

func FindFlashcardByShortTitle(shortTitle string) (*Flashcard, error) {
	return QueryFlashcard(`WHERE short_title = ?`, shortTitle)
}

func FindFlashcardByHash(hash string) (*Flashcard, error) {
	return QueryFlashcard(`WHERE hash = ?`, hash)
}

func FindFlashcardsLastCheckedBefore(point time.Time) ([]*Flashcard, error) {
	return QueryFlashcards(`WHERE last_checked_at < ?`, timeToSQL(point))
}

/* SQL Helpers */

func QueryFlashcard(whereClause string, args ...any) (*Flashcard, error) {
	db := CurrentDB().Client()

	var f Flashcard
	var tagsRaw string
	var createdAt string
	var updatedAt string
	var deletedAt string
	var lastCheckedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			id,
			file_id,
			note_id,
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
			deleted_at,
			last_checked_at
		FROM flashcard
		%s;`, whereClause), args...).
		Scan(
			&f.ID,
			&f.FileID,
			&f.NoteID,
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
			&deletedAt,
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
	f.DeletedAt = timeFromSQL(deletedAt)
	f.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &f, nil
}

func QueryFlashcards(whereClause string, args ...any) ([]*Flashcard, error) {
	db := CurrentDB().Client()

	var flashcards []*Flashcard

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			id,
			file_id,
			note_id,
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
			deleted_at,
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
		var deletedAt string
		var lastCheckedAt string

		err = rows.Scan(
			&f.ID,
			&f.FileID,
			&f.NoteID,
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
			&deletedAt,
			&lastCheckedAt,
		)
		if err != nil {
			return nil, err
		}
		f.Tags = strings.Split(tagsRaw, ",")
		f.CreatedAt = timeFromSQL(createdAt)
		f.UpdatedAt = timeFromSQL(updatedAt)
		f.DeletedAt = timeFromSQL(deletedAt)
		f.LastCheckedAt = timeFromSQL(lastCheckedAt)
		flashcards = append(flashcards, &f)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return flashcards, err
}
