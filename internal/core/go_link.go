package core

import (
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

type GoLink struct {
	OID oid.OID `yaml:"oid" json:"oid"`

	// Pack file where this object belongs
	PackFileOID oid.OID `yaml:"packfile_oid" json:"packfile_oid"`

	NoteOID oid.OID `yaml:"note_oid" json:"note_oid"`

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path" json:"relative_path"`

	// The link text
	Text markdown.Document `yaml:"text" json:"text"`

	// The link destination
	URL string `yaml:"url" json:"url"`

	// The optional link title
	Title string `yaml:"title" json:"title"`

	// The optional GO name
	GoName string `yaml:"go_name" json:"go_name"`

	// Timestamps to track changes
	CreatedAt     time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at" json:"updated_at"`
	LastIndexedAt time.Time `yaml:"last_indexed_at,omitempty" json:"last_indexed_at,omitempty"`
}

func NewOrExistingGoLink(packFile *PackFile, note *Note, parsedGoLink *ParsedGoLink) (*GoLink, error) {
	// Try to find an existing object (instead of recreating it from scratch after every change)
	existingGoLink, err := CurrentRepository().FindGoLinkByGoName(string(parsedGoLink.GoName))
	if err != nil {
		return nil, err
	}
	if existingGoLink != nil {
		existingGoLink.update(packFile, note, parsedGoLink)
		return existingGoLink, nil
	}
	return NewGoLink(packFile, note, parsedGoLink), nil
}

func NewGoLink(packFile *PackFile, note *Note, parsedLink *ParsedGoLink) *GoLink {
	return &GoLink{
		OID:          oid.New(),
		PackFileOID:  packFile.OID,
		NoteOID:      note.OID,
		RelativePath: note.RelativePath,
		Text:         parsedLink.Text,
		URL:          parsedLink.URL,
		Title:        parsedLink.Title,
		GoName:       parsedLink.GoName,

		CreatedAt:     packFile.CTime,
		UpdatedAt:     packFile.CTime,
		LastIndexedAt: packFile.CTime,
	}
}

/* Object */

func (l *GoLink) Kind() string {
	return "link"
}

func (l *GoLink) UniqueOID() oid.OID {
	return l.OID
}

func (l *GoLink) ModificationTime() time.Time {
	return l.UpdatedAt
}

func (l *GoLink) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(l)
	if err != nil {
		return err
	}
	return nil
}

func (l *GoLink) Write(w io.Writer) error {
	data, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (l *GoLink) Relations() []*Relation {
	return nil
}

func (l GoLink) String() string {
	return fmt.Sprintf("link %q [%s]", l.URL, l.OID)
}

/* Format */

func (l *GoLink) ToYAML() string {
	return ToBeautifulYAML(l)
}

func (l *GoLink) ToJSON() string {
	return ToBeautifulJSON(l)
}

func (l *GoLink) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(string(l.Text))
	sb.WriteString("](")
	sb.WriteString(string(l.URL))
	sb.WriteString(")")
	return sb.String()
}

/* Update */

func (l *GoLink) update(packFile *PackFile, note *Note, parsedLink *ParsedGoLink) {
	stale := false

	if l.NoteOID != note.OID {
		l.NoteOID = note.OID
		stale = true
	}
	if l.Text != parsedLink.Text {
		l.Text = parsedLink.Text
		stale = true
	}
	if l.URL != parsedLink.URL {
		l.URL = parsedLink.URL
		stale = true
	}
	if l.Title != parsedLink.Title {
		l.Title = parsedLink.Title
		stale = true
	}
	if l.GoName != parsedLink.GoName {
		l.GoName = parsedLink.GoName
		stale = true
	}

	l.PackFileOID = packFile.OID
	l.LastIndexedAt = packFile.CTime

	if stale {
		l.UpdatedAt = packFile.CTime
	}
}

/* Database Management */

func (l *GoLink) Save() error {
	CurrentLogger().Debugf("Saving go link %s...", l.GoName)
	query := `
		INSERT INTO link(
			oid,
			packfile_oid,
			note_oid,
			relative_path,
			"text",
			url,
			title,
			go_name,
			created_at,
			updated_at,
			last_indexed_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			note_oid = ?,
			relative_path = ?,
			"text" = ?,
			url = ?,
			title = ?,
			go_name = ?,
			updated_at = ?,
			last_indexed_at = ?
		;
		`
	_, err := CurrentDB().Client().Exec(query,
		// Insert
		l.OID,
		l.PackFileOID,
		l.NoteOID,
		l.RelativePath,
		l.Text,
		l.URL,
		l.Title,
		l.GoName,
		timeToSQL(l.CreatedAt),
		timeToSQL(l.UpdatedAt),
		timeToSQL(l.LastIndexedAt),
		// Update
		l.PackFileOID,
		l.NoteOID,
		l.RelativePath,
		l.Text,
		l.URL,
		l.Title,
		l.GoName,
		timeToSQL(l.UpdatedAt),
		timeToSQL(l.LastIndexedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (l *GoLink) Delete() error {
	CurrentLogger().Debugf("Deleting link %s...", l.GoName)
	query := `DELETE FROM link WHERE oid = ? AND packfile_oid = ?;`
	_, err := CurrentDB().Client().Exec(query, l.OID, l.PackFileOID)
	return err
}

/* SQL Queries */

// CountGoLinks returns the total number of links.
func (r *Repository) CountGoLinks() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM link`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) LoadGoLinkByOID(oid oid.OID) (*GoLink, error) {
	return QueryGoLink(CurrentDB().Client(), "WHERE oid = ?", oid)
}

func (r *Repository) FindGoLinkByGoName(goName string) (*GoLink, error) {
	return QueryGoLink(CurrentDB().Client(), "WHERE go_name = ?", goName)
}

func (r *Repository) FindGoLinksByText(text string) ([]*GoLink, error) {
	return QueryGoLinks(CurrentDB().Client(), "WHERE text = ?", text)
}

func (r *Repository) FindGoLinksLastCheckedBefore(point time.Time, path string) ([]*GoLink, error) {
	if path == "." {
		path = ""
	}
	return QueryGoLinks(CurrentDB().Client(), `WHERE last_indexed_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

/* SQL Helpers */

func QueryGoLink(db SQLClient, whereClause string, args ...any) (*GoLink, error) {
	var l GoLink
	var createdAt string
	var updatedAt string
	var lastIndexedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			note_oid,
			relative_path,
			"text",
			url,
			title,
			go_name,
			created_at,
			updated_at,
			last_indexed_at
		FROM link
		%s;`, whereClause), args...).
		Scan(
			&l.OID,
			&l.PackFileOID,
			&l.NoteOID,
			&l.RelativePath,
			&l.Text,
			&l.URL,
			&l.Title,
			&l.GoName,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	l.CreatedAt = timeFromSQL(createdAt)
	l.UpdatedAt = timeFromSQL(updatedAt)
	l.LastIndexedAt = timeFromSQL(lastIndexedAt)

	return &l, nil
}

func QueryGoLinks(db SQLClient, whereClause string, args ...any) ([]*GoLink, error) {
	var links []*GoLink

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			note_oid,
			relative_path,
			"text",
			url,
			title,
			go_name,
			created_at,
			updated_at,
			last_indexed_at
		FROM link
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var l GoLink
		var createdAt string
		var updatedAt string
		var lastIndexedAt string

		err = rows.Scan(
			&l.OID,
			&l.PackFileOID,
			&l.NoteOID,
			&l.RelativePath,
			&l.Text,
			&l.URL,
			&l.Title,
			&l.GoName,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
		)
		if err != nil {
			return nil, err
		}

		l.CreatedAt = timeFromSQL(createdAt)
		l.UpdatedAt = timeFromSQL(updatedAt)
		l.LastIndexedAt = timeFromSQL(lastIndexedAt)
		links = append(links, &l)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return links, err
}
