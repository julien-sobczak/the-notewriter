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
		l.CreatedAt = now
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

// CountLinks returns the total number of links.
func CountLinks() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM link WHERE deleted_at = ''`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func LoadLinkByID(id int64) (*Link, error) {
	return QueryLink("WHERE id = ?", id)
}

func FindLinkByGoName(goName string) (*Link, error) {
	return QueryLink("WHERE go_name = ?", goName)
}

func FindLinksByText(text string) ([]*Link, error) {
	return QueryLinks("WHERE text = ?", text)
}

/* SQL Helpers */

func QueryLink(whereClause string, args ...any) (*Link, error) {
	db := CurrentDB().Client()

	var l Link
	var createdAt string
	var updatedAt string
	var deletedAt string
	var lastCheckedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
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
		%s;`, whereClause), args...).
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
		return nil, err
	}

	l.CreatedAt = timeFromSQL(createdAt)
	l.UpdatedAt = timeFromSQL(updatedAt)
	l.DeletedAt = timeFromSQL(deletedAt)
	l.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &l, nil
}

func QueryLinks(whereClause string, args ...any) ([]*Link, error) {
	db := CurrentDB().Client()

	var links []*Link

	rows, err := db.Query(fmt.Sprintf(`
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
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var l Link
		var createdAt string
		var updatedAt string
		var deletedAt string
		var lastCheckedAt string

		err = rows.Scan(
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
		)
		if err != nil {
			return nil, err
		}

		l.CreatedAt = timeFromSQL(createdAt)
		l.UpdatedAt = timeFromSQL(updatedAt)
		l.DeletedAt = timeFromSQL(deletedAt)
		l.LastCheckedAt = timeFromSQL(lastCheckedAt)
		links = append(links, &l)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return links, err
}
