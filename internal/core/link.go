package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
)

type Link struct {
	ID int64

	NoteID int64

	// The link text
	Text string

	// The link destination
	URL string

	// The optional link title
	Title string

	// The optional GO name
	GoName string

	// Timestamps to track changes
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     time.Time
	LastCheckedAt time.Time
}

func (l *Link) Save() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = l.SaveWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (l *Link) SaveWithTx(tx *sql.Tx) error {
	now := clock.Now()
	l.UpdatedAt = now
	l.LastCheckedAt = now

	if l.ID != 0 {
		return l.UpdateWithTx(tx)
	} else {
		return l.InsertWithTx(tx)
	}
}

func (l *Link) InsertWithTx(tx *sql.Tx) error {
	query := `
		INSERT INTO link(
			id,
			note_id,
			"text",
			url,
			title,
			go_name,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at
		)
		VALUES (NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`
	res, err := tx.Exec(query,
		l.NoteID,
		l.Text,
		l.URL,
		l.Title,
		l.GoName,
		timeToSQL(l.CreatedAt),
		timeToSQL(l.UpdatedAt),
		timeToSQL(l.DeletedAt),
		timeToSQL(l.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return err
	}
	l.ID = id
	return nil
}

func (l *Link) UpdateWithTx(tx *sql.Tx) error {
	query := `
		UPDATE link
		SET
			note_id = ?,
			"text" = ?,
			url = ?,
			title = ?,
			go_name = ?,
			updated_at = ?,
			deleted_at = ?,
			last_checked_at = ?
		)
		WHERE id = ?;
		`
	_, err := tx.Exec(query,
		l.NoteID,
		l.Text,
		l.URL,
		l.Title,
		l.GoName,
		timeToSQL(l.UpdatedAt),
		timeToSQL(l.DeletedAt),
		timeToSQL(l.LastCheckedAt),
		l.ID,
	)

	return err
}

func LoadLinkByID(id int64) (*Link, error) {
	db := CurrentDB().Client()

	var l Link
	var createdAt string
	var updatedAt string
	var deletedAt string
	var lastCheckedAt string

	if err := db.QueryRow(`
		SELECT
			id,
			note_id,
			"text",
			url,
			title,
			go_name,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at
		FROM link
		WHERE id = ?`, id).
		Scan(
			&l.ID,
			&l.NoteID,
			&l.Text,
			&l.URL,
			&l.Title,
			&l.GoName,
			&createdAt,
			&updatedAt,
			&deletedAt,
			&lastCheckedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("unknown link %v", id)
		}
		return nil, err
	}

	l.CreatedAt = timeFromSQL(createdAt)
	l.UpdatedAt = timeFromSQL(updatedAt)
	l.DeletedAt = timeFromSQL(deletedAt)
	l.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &l, nil
}

// TODO Add FindLinkByGoName
// TODO Add FindLinksByText
