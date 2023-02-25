package core

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"gopkg.in/yaml.v3"
)

type Link struct {
	OID string `yaml:"oid"`

	NoteOID string `yaml:"note_oid"`

	// The link text
	Text string `yaml:"text"`

	// The link destination
	URL string `yaml:"url"`

	// The optional link title
	Title string `yaml:"title"`

	// The optional GO name
	GoName string `yaml:"go_name"`

	// Timestamps to track changes
	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	DeletedAt     time.Time `yaml:"-"`
	LastCheckedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}

func NewOrExistingLink(note *Note, text, url, title, goName string) *Link {
	link, err := FindLinkByGoName(goName)
	if err != nil {
		log.Fatal(err)
	}
	if link != nil {
		link.Update(note, text, url, title, goName)
		return link
	}
	return NewLink(note, text, url, title, goName)
}

func NewLink(note *Note, text, url, title, goName string) *Link {
	return &Link{
		OID:     NewOID(),
		NoteOID: note.OID,
		Text:    text,
		URL:     url,
		Title:   title,
		GoName:  goName,

		CreatedAt: clock.Now(),
		UpdatedAt: clock.Now(),

		new:   true,
		stale: true,
	}
}

// NewLinkFromObject instantiates a new link from an object file.
func NewLinkFromObject(r io.Reader) *Link {
	// TODO
	return &Link{}
}

/* Object */

func (l *Link) Kind() string {
	return "link"
}

func (l *Link) UniqueOID() string {
	return l.OID
}

func (l *Link) ModificationTime() time.Time {
	return l.UpdatedAt
}

func (l *Link) State() State {
	if !l.DeletedAt.IsZero() {
		return Deleted
	}
	if l.new {
		return Added
	}
	if l.stale {
		return Modified
	}
	return None
}

func (l *Link) SetTombstone() {
	l.DeletedAt = clock.Now()
	l.stale = true
}

func (l *Link) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(l)
	if err != nil {
		return err
	}
	return nil
}

func (l *Link) Write(w io.Writer) error {
	data, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (l *Link) SubObjects() []Object {
	return nil
}


func (l *Link) Blobs() []Blob {
	// Use Media.Blobs() instead
	return nil
}

func (l Link) String() string {
	return fmt.Sprintf("link %q [%s]", l.URL, l.OID)
}

/* Update */

func (l *Link) Update(note *Note, text, url, title, goName string) {
	if l.NoteOID != note.OID {
		l.NoteOID = note.OID
		l.stale = true
	}
	if l.Text != text {
		l.Text = text
		l.stale = true
	}
	if l.URL != url {
		l.URL = url
		l.stale = true
	}
	if l.Title != title {
		l.Title = title
		l.stale = true
	}
	if l.GoName != goName {
		l.GoName = goName
		l.stale = true
	}
}

/* State Management */

func (l *Link) New() bool {
	return l.new
}

func (l *Link) Updated() bool {
	return l.stale
}

/* Database Management */

func (l *Link) Check() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = l.CheckWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil

}

func (l *Link) CheckWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Checking link %s...", l.GoName)
	l.LastCheckedAt = clock.Now()
	query := `
		UPDATE link
		SET last_checked_at = ?
		WHERE oid = ?;`
	_, err := tx.Exec(query,
		timeToSQL(l.LastCheckedAt),
		l.OID,
	)

	return err
}

func (l *Link) Save(tx *sql.Tx) error {
	switch l.State() {
	case Added:
		return l.InsertWithTx(tx)
	case Modified:
		return l.UpdateWithTx(tx)
	case Deleted:
		return l.DeleteWithTx(tx)
	default:
		return l.CheckWithTx(tx)
	}
}

func (l *Link) OldSave() error { // FIXME remove deprecated
	if !l.stale {
		return nil
	}

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

	l.new = false
	l.stale = false

	return nil
}

func (l *Link) SaveWithTx(tx *sql.Tx) error { // FIXME remove deprecated
	if !l.stale {
		return nil
	}

	now := clock.Now()
	l.UpdatedAt = now
	l.LastCheckedAt = now

	if !l.new {
		if err := l.UpdateWithTx(tx); err != nil {
			return err
		}
	} else {
		l.CreatedAt = now
		if err := l.InsertWithTx(tx); err != nil {
			return err
		}
	}

	l.new = false
	l.stale = false

	return nil
}

func (l *Link) InsertWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Creating link %s...", l.GoName)
	query := `
		INSERT INTO link(
			oid,
			note_oid,
			"text",
			url,
			title,
			go_name,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`
	_, err := tx.Exec(query,
		l.OID,
		l.NoteOID,
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

	return nil
}

func (l *Link) UpdateWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Updating link %s...", l.GoName)
	query := `
		UPDATE link
		SET
			note_oid = ?,
			"text" = ?,
			url = ?,
			title = ?,
			go_name = ?,
			updated_at = ?,
			deleted_at = ?,
			last_checked_at = ?
		)
		WHERE oid = ?;
		`
	_, err := tx.Exec(query,
		l.NoteOID,
		l.Text,
		l.URL,
		l.Title,
		l.GoName,
		timeToSQL(l.UpdatedAt),
		timeToSQL(l.DeletedAt),
		timeToSQL(l.LastCheckedAt),
		l.OID,
	)

	return err
}

func (l *Link) Delete() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = l.DeleteWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (l *Link) DeleteWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Deleting link %s...", l.GoName)
	query := `DELETE FROM link WHERE oid = ?;`
	_, err := tx.Exec(query, l.OID)
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

func LoadLinkByOID(oid string) (*Link, error) {
	return QueryLink("WHERE oid = ?", oid)
}

func FindLinkByGoName(goName string) (*Link, error) {
	return QueryLink("WHERE go_name = ?", goName)
}

func FindLinksByText(text string) ([]*Link, error) {
	return QueryLinks("WHERE text = ?", text)
}

func FindLinksLastCheckedBefore(point time.Time) ([]*Link, error) {
	return QueryLinks(`WHERE last_checked_at < ?`, timeToSQL(point))
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
			oid,
			note_oid,
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
			&l.OID,
			&l.NoteOID,
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
			return nil, nil
		}
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
			oid,
			note_oid,
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
			&l.OID,
			&l.NoteOID,
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
