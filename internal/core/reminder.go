package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"
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
	// There is no common interface between sql.DB and sql.Txt
	// See https://github.com/golang/go/issues/14468

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

func LoadReminderByID(id int64) (*Reminder, error) {
	db := CurrentDB().Client()

	var r Reminder
	var lastPerformedAt string
	var nextPerformedAt string
	var createdAt string
	var updatedAt string
	var deletedAt string
	var lastCheckedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(`
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
		WHERE id = ?`, id).
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
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("unknown reminder %v", id)
		}
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

// TODO Add FindRemindersByUpcomingDate
