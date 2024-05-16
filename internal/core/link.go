package core

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"gopkg.in/yaml.v3"
)

type Link struct {
	OID string `yaml:"oid"`

	NoteOID string `yaml:"note_oid"`

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path"`

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
	DeletedAt     time.Time `yaml:"deleted_at,omitempty"`
	LastCheckedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}

func NewOrExistingLink(note *Note, text, url, title, goName string) *Link {
	link, err := CurrentRepository().FindLinkByGoName(goName)
	if err != nil {
		log.Fatal(err)
	}
	if link != nil {
		link.update(note, text, url, title, goName)
		return link
	}
	return NewLink(note, text, url, title, goName)
}

func NewLink(note *Note, text, url, title, goName string) *Link {
	return &Link{
		OID:          NewOID(),
		NoteOID:      note.OID,
		RelativePath: note.RelativePath,
		Text:         text,
		URL:          url,
		Title:        title,
		GoName:       goName,

		CreatedAt: clock.Now(),
		UpdatedAt: clock.Now(),

		new:   true,
		stale: true,
	}
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

func (l *Link) Refresh() (bool, error) {
	// No dependencies = no need to refresh
	return false, nil
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

func (l *Link) ForceState(state State) {
	switch state {
	case Added:
		l.new = true
	case Deleted:
		l.DeletedAt = clock.Now()
	}
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

func (l *Link) SubObjects() []StatefulObject {
	return nil
}

func (l *Link) Blobs() []*BlobRef {
	return nil
}

func (l *Link) Relations() []*Relation {
	return nil
}

func (l Link) String() string {
	return fmt.Sprintf("link %q [%s]", l.URL, l.OID)
}

/* Update */

func (l *Link) update(note *Note, text, url, title, goName string) {
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
	CurrentLogger().Debugf("Checking link %s...", l.GoName)
	l.LastCheckedAt = clock.Now()
	query := `
		UPDATE link
		SET last_checked_at = ?
		WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query,
		timeToSQL(l.LastCheckedAt),
		l.OID,
	)

	return err
}

func (l *Link) Save() error {
	var err error
	l.UpdatedAt = clock.Now()
	l.LastCheckedAt = clock.Now()
	switch l.State() {
	case Added:
		err = l.Insert()
	case Modified:
		err = l.Update()
	case Deleted:
		err = l.Delete()
	default:
		err = l.Check()
	}
	if err != nil {
		return err
	}
	l.new = false
	l.stale = false
	return nil
}

func (l *Link) Insert() error {
	CurrentLogger().Debugf("Creating link %s...", l.GoName)
	query := `
		INSERT INTO link(
			oid,
			note_oid,
			relative_path,
			"text",
			url,
			title,
			go_name,
			created_at,
			updated_at,
			last_checked_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`
	_, err := CurrentDB().Client().Exec(query,
		l.OID,
		l.NoteOID,
		l.RelativePath,
		l.Text,
		l.URL,
		l.Title,
		l.GoName,
		timeToSQL(l.CreatedAt),
		timeToSQL(l.UpdatedAt),
		timeToSQL(l.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (l *Link) Update() error {
	CurrentLogger().Debugf("Updating link %s...", l.GoName)
	query := `
		UPDATE link
		SET
			note_oid = ?,
			relative_path = ?,
			"text" = ?,
			url = ?,
			title = ?,
			go_name = ?,
			updated_at = ?,
			last_checked_at = ?
		)
		WHERE oid = ?;
		`
	_, err := CurrentDB().Client().Exec(query,
		l.NoteOID,
		l.RelativePath,
		l.Text,
		l.URL,
		l.Title,
		l.GoName,
		timeToSQL(l.UpdatedAt),
		timeToSQL(l.LastCheckedAt),
		l.OID,
	)

	return err
}

func (l *Link) Delete() error {
	CurrentLogger().Debugf("Deleting link %s...", l.GoName)
	query := `DELETE FROM link WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query, l.OID)
	return err
}

// CountLinks returns the total number of links.
func (r *Repository) CountLinks() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM link`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) LoadLinkByOID(oid string) (*Link, error) {
	return QueryLink(CurrentDB().Client(), "WHERE oid = ?", oid)
}

func (r *Repository) FindLinkByGoName(goName string) (*Link, error) {
	return QueryLink(CurrentDB().Client(), "WHERE go_name = ?", goName)
}

func (r *Repository) FindLinksByText(text string) ([]*Link, error) {
	return QueryLinks(CurrentDB().Client(), "WHERE text = ?", text)
}

func (r *Repository) FindLinksLastCheckedBefore(point time.Time, path string) ([]*Link, error) {
	if path == "." {
		path = ""
	}
	return QueryLinks(CurrentDB().Client(), `WHERE last_checked_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

/* SQL Helpers */

func QueryLink(db SQLClient, whereClause string, args ...any) (*Link, error) {
	var l Link
	var createdAt string
	var updatedAt string
	var lastCheckedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			note_oid,
			relative_path,
			"text",
			url,
			title,
			go_name,
			created_at,
			updated_at,
			last_checked_at
		FROM link
		%s;`, whereClause), args...).
		Scan(
			&l.OID,
			&l.NoteOID,
			&l.RelativePath,
			&l.Text,
			&l.URL,
			&l.Title,
			&l.GoName,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	l.CreatedAt = timeFromSQL(createdAt)
	l.UpdatedAt = timeFromSQL(updatedAt)
	l.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &l, nil
}

func QueryLinks(db SQLClient, whereClause string, args ...any) ([]*Link, error) {
	var links []*Link

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			note_oid,
			relative_path,
			"text",
			url,
			title,
			go_name,
			created_at,
			updated_at,
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
		var lastCheckedAt string

		err = rows.Scan(
			&l.OID,
			&l.NoteOID,
			&l.RelativePath,
			&l.Text,
			&l.URL,
			&l.Title,
			&l.GoName,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		)
		if err != nil {
			return nil, err
		}

		l.CreatedAt = timeFromSQL(createdAt)
		l.UpdatedAt = timeFromSQL(updatedAt)
		l.LastCheckedAt = timeFromSQL(lastCheckedAt)
		links = append(links, &l)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return links, err
}
