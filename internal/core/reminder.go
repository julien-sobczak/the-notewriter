package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
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

func NewReminder(description, tag string) *Reminder {
	// TODO
	return nil
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
		timeToSQL(r.LastCheckedAt),
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
		timeToSQL(r.LastCheckedAt),
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
