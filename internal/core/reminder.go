package core

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
)

type Reminder struct {
	ID int64

	// File
	FileID int64
	File   *File // Lazy-loaded

	// Note representing the flashcard
	NoteID int64
	Note   *Note // Lazy-loaded

	// Description
	DescriptionRaw      string
	DescriptionMarkdown string
	DescriptionHTML     string
	DescriptionText     string

	// Tag value containig the formula to determine the next occurence
	Tag string

	// Timestamps to track progress
	LastPerformedAt time.Time
	NextPerformedAt time.Time

	// Timestamps to track changes
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     time.Time
	LastCheckedAt time.Time
}

// NewReminder instantiates a new reminder.
func NewReminder(n *Note, descriptionRaw, tag string) (*Reminder, error) {
	r := &Reminder{
		ID:             0,
		FileID:         n.FileID,
		NoteID:         n.ID,
		DescriptionRaw: descriptionRaw,
		Tag:            tag,
	}

	r.DescriptionMarkdown = markdown.ToMarkdown(r.DescriptionRaw)
	r.DescriptionHTML = markdown.ToHTML(r.DescriptionMarkdown)
	r.DescriptionText = markdown.ToText(r.DescriptionMarkdown)

	err := r.Next()
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Reminder) Next() error {
	if clock.Now().Before(r.NextPerformedAt) {
		// already OK
		return nil
	}

	expression := strings.TrimPrefix(r.Tag, "#reminder-")

	r.LastPerformedAt = r.NextPerformedAt
	r.NextPerformedAt = evaluateTimeExpression(expression)[0]
	return nil
}

func evaluateTimeExpression(expr string) []time.Time {
	// TODO implement logic
	t, err := time.Parse("2006-01", expr)
	if err != nil {
		// log.Fatalf("Unable to parse reminder expression: %v", err)
		return []time.Time{{}}
	}

	// Step 1: Replace variables
	// Step 2: Evaluation

	return []time.Time{t}
}

func (r *Reminder) Save() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = r.SaveWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (r *Reminder) SaveWithTx(tx *sql.Tx) error {
	now := clock.Now()
	r.UpdatedAt = now
	r.LastCheckedAt = now

	if r.ID != 0 {
		return r.UpdateWithTx(tx)
	} else {
		r.CreatedAt = now
		return r.InsertWithTx(tx)
	}
}

func (r *Reminder) InsertWithTx(tx *sql.Tx) error {
	query := `
		INSERT INTO reminder(
			id,
			file_id,
			note_id,
			description_raw,
			description_markdown,
			description_html,
			description_text,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at
		)
		VALUES (NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	res, err := tx.Exec(query,
		r.FileID,
		r.NoteID,
		r.DescriptionRaw,
		r.DescriptionMarkdown,
		r.DescriptionHTML,
		r.DescriptionText,
		r.Tag,
		timeToSQL(r.LastPerformedAt),
		timeToSQL(r.NextPerformedAt),
		timeToSQL(r.CreatedAt),
		timeToSQL(r.UpdatedAt),
		timeToSQL(r.DeletedAt),
		timeToSQL(r.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return err
	}
	r.ID = id

	return nil
}

func (r *Reminder) UpdateWithTx(tx *sql.Tx) error {
	query := `
		UPDATE reminder
		SET
			file_id = ?,
			note_id = ?,
			description_raw = ?,
			description_markdown = ?,
			description_html = ?,
			description_text = ?,
			tag = ?,
			last_performed_at = ?,
			next_performed_at = ?,
			updated_at = ?,
			deleted_at = ?,
			last_checked_at = ?
		)
		WHERE id = ?;
	`
	_, err := tx.Exec(query,
		r.FileID,
		r.NoteID,
		r.DescriptionRaw,
		r.DescriptionMarkdown,
		r.DescriptionHTML,
		r.DescriptionText,
		r.Tag,
		timeToSQL(r.LastPerformedAt),
		timeToSQL(r.NextPerformedAt),
		timeToSQL(r.UpdatedAt),
		timeToSQL(r.DeletedAt),
		timeToSQL(r.LastCheckedAt),
		r.ID,
	)

	return err
}

// CountReminders returns the total number of reminders.
func CountReminders() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM reminder WHERE deleted_at = ''`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func FindReminders() ([]*Reminder, error) {
	return QueryReminders("")
}

func LoadReminderByID(id int64) (*Reminder, error) {
	return QueryReminder(`WHERE id = ?`, id)
}

func FindRemindersByUpcomingDate(deadline time.Time) ([]*Reminder, error) {
	return QueryReminders(`WHERE next_performed_at > ?`, timeToSQL(deadline))
}

/* SQL Helpers */

func QueryReminder(whereClause string, args ...any) (*Reminder, error) {
	db := CurrentDB().Client()

	var r Reminder
	var lastPerformedAt string
	var nextPerformedAt string
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
			description_raw,
			description_markdown,
			description_html,
			description_text,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at
		FROM reminder
		%s;`, whereClause), args...).
		Scan(
			&r.ID,
			&r.FileID,
			&r.NoteID,
			&r.DescriptionRaw,
			&r.DescriptionMarkdown,
			&r.DescriptionHTML,
			&r.DescriptionText,
			&r.Tag,
			&lastPerformedAt,
			&nextPerformedAt,
			&createdAt,
			&updatedAt,
			&deletedAt,
			&lastCheckedAt,
		); err != nil {

		return nil, err
	}

	r.LastPerformedAt = timeFromSQL(lastPerformedAt)
	r.NextPerformedAt = timeFromSQL(nextPerformedAt)
	r.CreatedAt = timeFromSQL(createdAt)
	r.UpdatedAt = timeFromSQL(updatedAt)
	r.DeletedAt = timeFromSQL(deletedAt)
	r.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &r, nil
}

func QueryReminders(whereClause string, args ...any) ([]*Reminder, error) {
	db := CurrentDB().Client()

	var reminders []*Reminder

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			id,
			file_id,
			note_id,
			description_raw,
			description_markdown,
			description_html,
			description_text,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at
		FROM reminder
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r Reminder
		var lastPerformedAt string
		var nextPerformedAt string
		var createdAt string
		var updatedAt string
		var deletedAt string
		var lastCheckedAt string

		err = rows.Scan(
			&r.ID,
			&r.FileID,
			&r.NoteID,
			&r.DescriptionRaw,
			&r.DescriptionMarkdown,
			&r.DescriptionHTML,
			&r.DescriptionText,
			&r.Tag,
			&lastPerformedAt,
			&nextPerformedAt,
			&createdAt,
			&updatedAt,
			&deletedAt,
			&lastCheckedAt,
		)
		if err != nil {
			return nil, err
		}

		r.LastPerformedAt = timeFromSQL(lastPerformedAt)
		r.NextPerformedAt = timeFromSQL(nextPerformedAt)
		r.CreatedAt = timeFromSQL(createdAt)
		r.UpdatedAt = timeFromSQL(updatedAt)
		r.DeletedAt = timeFromSQL(deletedAt)
		r.LastCheckedAt = timeFromSQL(lastCheckedAt)
		reminders = append(reminders, &r)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return reminders, err
}
