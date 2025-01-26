package core

import (
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"gopkg.in/yaml.v3"
)

type GoLink struct {
	OID OID `yaml:"oid" json:"oid"`

	// Pack file where this object belongs
	PackFileOID OID `yaml:"packfile_oid" json:"packfile_oid"`

	NoteOID OID `yaml:"note_oid" json:"note_oid"`

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
	DeletedAt     time.Time `yaml:"deleted_at,omitempty" json:"deleted_at,omitempty"`
	LastCheckedAt time.Time `yaml:"-" json:"-"`

	new   bool
	stale bool
}

func NewOrExistingGoLink(packFileOID OID, note *Note, parsedGoLink *ParsedGoLink) (*GoLink, error) {
	existingGoLink, err := CurrentRepository().FindGoLinkByGoName(string(parsedGoLink.GoName))
	if err != nil {
		return nil, err
	}
	if existingGoLink != nil {
		existingGoLink.update(packFileOID, note, parsedGoLink)
		return existingGoLink, nil
	}
	return NewGoLink(packFileOID, note, parsedGoLink), nil
}

func NewGoLink(packFileOID OID, note *Note, parsedLink *ParsedGoLink) *GoLink {
	return &GoLink{
		OID:          NewOID(),
		PackFileOID:  packFileOID,
		NoteOID:      note.OID,
		RelativePath: note.RelativePath,
		Text:         parsedLink.Text,
		URL:          parsedLink.URL,
		Title:        parsedLink.Title,
		GoName:       parsedLink.GoName,

		CreatedAt: clock.Now(),
		UpdatedAt: clock.Now(),

		new:   true,
		stale: true,
	}
}

/* Object */

func (l *GoLink) Kind() string {
	return "link"
}

func (l *GoLink) UniqueOID() OID {
	return l.OID
}

func (l *GoLink) ModificationTime() time.Time {
	return l.UpdatedAt
}

func (l *GoLink) Refresh() (bool, error) {
	// No dependencies = no need to refresh
	return false, nil
}

func (l *GoLink) Stale() bool {
	return l.stale
}

func (l *GoLink) State() State {
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

func (l *GoLink) ForceState(state State) {
	switch state {
	case Added:
		l.new = true
	case Deleted:
		l.DeletedAt = clock.Now()
	}
	l.stale = true
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

func (l *GoLink) update(packFileOID OID, note *Note, parsedLink *ParsedGoLink) {
	if l.NoteOID != note.OID {
		l.NoteOID = note.OID
		l.stale = true
	}
	if l.Text != parsedLink.Text {
		l.Text = parsedLink.Text
		l.stale = true
	}
	if l.URL != parsedLink.URL {
		l.URL = parsedLink.URL
		l.stale = true
	}
	if l.Title != parsedLink.Title {
		l.Title = parsedLink.Title
		l.stale = true
	}
	if l.GoName != parsedLink.GoName {
		l.GoName = parsedLink.GoName
		l.stale = true
	}

	l.PackFileOID = packFileOID
	// Do not set the stale flag. An object can be unchanged when a new pack file is created (ex: new note appended at the end)
	// FIXME always update and marks as stale? If we end up here, it means the Markdown file has been modified
}

/* State Management */

func (l *GoLink) New() bool {
	return l.new
}

func (l *GoLink) Updated() bool {
	return l.stale
}

/* Database Management */

func (l *GoLink) Check() error {
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

func (l *GoLink) Save() error {
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

func (l *GoLink) Insert() error {
	CurrentLogger().Debugf("Creating go link %s...", l.GoName)
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
			last_checked_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`
	_, err := CurrentDB().Client().Exec(query,
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
		timeToSQL(l.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (l *GoLink) Update() error {
	CurrentLogger().Debugf("Updating link %s...", l.GoName)
	query := `
		UPDATE link
		SET
			packfile_oid = ?,
			note_oid = ?,
			relative_path = ?,
			"text" = ?,
			url = ?,
			title = ?,
			go_name = ?,
			updated_at = ?,
			last_checked_at = ?
		WHERE oid = ?;
		`
	_, err := CurrentDB().Client().Exec(query,
		l.PackFileOID,
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

func (l *GoLink) Delete() error {
	l.ForceState(Deleted)
	CurrentLogger().Debugf("Deleting link %s...", l.GoName)
	query := `DELETE FROM link WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query, l.OID)
	return err
}

// CountGoLinks returns the total number of links.
func (r *Repository) CountGoLinks() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM link`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) LoadGoLinkByOID(oid OID) (*GoLink, error) {
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
	return QueryGoLinks(CurrentDB().Client(), `WHERE last_checked_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

/* SQL Helpers */

func QueryGoLink(db SQLClient, whereClause string, args ...any) (*GoLink, error) {
	var l GoLink
	var createdAt string
	var updatedAt string
	var lastCheckedAt string

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
			last_checked_at
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
			last_checked_at
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
		var lastCheckedAt string

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
