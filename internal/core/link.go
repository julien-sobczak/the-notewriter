package core

import (
	"database/sql"
	"time"
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
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

func (l *Link) Save() error {
	// TODO
	return nil
}

func (l *Link) SaveWithTx(tx *sql.Tx) error {
	// TODO
	return nil
}
