package core

import (
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

type Goto struct {
	OID oid.OID `yaml:"oid" json:"oid"`

	// Pack file where this object belongs
	PackFileOID oid.OID `yaml:"packfile_oid" json:"packfile_oid"`

	NoteOID oid.OID `yaml:"note_oid" json:"note_oid"`

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path" json:"relative_path"`

	// The link text
	Text markdown.Document `yaml:"text" json:"text"`

	// The link destination
	URL ParameterizedURL `yaml:"url" json:"url"`

	// The optional link title
	Title string `yaml:"title" json:"title"`

	// The optional GO name
	Name string `yaml:"name" json:"name"`

	// Timestamps to track changes
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
	IndexedAt time.Time `yaml:"indexed_at,omitempty" json:"indexed_at,omitempty"`
}

func NewOrExistingGoto(packFile *PackFile, note *Note, parsedGoto *ParsedGoto) (*Goto, error) {
	// Try to find an existing object (instead of recreating it from scratch after every change)
	existingGoto, err := CurrentRepository().FindGotoByName(string(parsedGoto.Name))
	if err != nil {
		return nil, err
	}
	if existingGoto != nil {
		existingGoto.update(packFile, note, parsedGoto)
		return existingGoto, nil
	}
	return NewGoto(packFile, note, parsedGoto), nil
}

func NewGoto(packFile *PackFile, note *Note, parsedLink *ParsedGoto) *Goto {
	return &Goto{
		OID:          oid.New(),
		PackFileOID:  packFile.OID,
		NoteOID:      note.OID,
		RelativePath: note.RelativePath,
		Text:         parsedLink.Text,
		URL:          ParameterizedURL(parsedLink.URL),
		Title:        parsedLink.Title,
		Name:         parsedLink.Name,

		CreatedAt: packFile.CTime,
		UpdatedAt: packFile.CTime,
		IndexedAt: packFile.CTime,
	}
}

/* Object */

func (l *Goto) FileRelativePath() string {
	return l.RelativePath
}

func (l *Goto) Kind() string {
	return "link"
}

func (l *Goto) UniqueOID() oid.OID {
	return l.OID
}

func (l *Goto) ModificationTime() time.Time {
	return l.UpdatedAt
}

func (l *Goto) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(l)
	if err != nil {
		return err
	}
	return nil
}

func (l *Goto) Write(w io.Writer) error {
	data, err := yaml.Marshal(l)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (l *Goto) Relations() []*Relation {
	return nil
}

func (l Goto) String() string {
	return fmt.Sprintf("link %q [%s]", l.URL, l.OID)
}

// Placeholders extracts all placeholders from the goto URL
func (l *Goto) Placeholders() []Placeholder {
	return l.URL.Placeholders()
}

// Expand replaces placeholders in the goto URL with provided values
func (l *Goto) Expand(values map[string]string) ParameterizedURL {
	return l.URL.Expand(values)
}

/* Format */

func (l *Goto) ToYAML() string {
	return ToBeautifulYAML(l)
}

func (l *Goto) ToJSON() string {
	return ToBeautifulJSON(l)
}

func (l *Goto) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(string(l.Text))
	sb.WriteString("](")
	sb.WriteString(string(l.URL))
	sb.WriteString(")")
	return sb.String()
}

/* Update */

func (l *Goto) update(packFile *PackFile, note *Note, parsedLink *ParsedGoto) {
	stale := false

	if l.NoteOID != note.OID {
		l.NoteOID = note.OID
		stale = true
	}
	if l.Text != parsedLink.Text {
		l.Text = parsedLink.Text
		stale = true
	}
	if l.URL != ParameterizedURL(parsedLink.URL) {
		l.URL = ParameterizedURL(parsedLink.URL)
		stale = true
	}
	if l.Title != parsedLink.Title {
		l.Title = parsedLink.Title
		stale = true
	}
	if l.Name != parsedLink.Name {
		l.Name = parsedLink.Name
		stale = true
	}

	l.PackFileOID = packFile.OID
	l.IndexedAt = packFile.CTime

	if stale {
		l.UpdatedAt = packFile.CTime
	}
}

/* Database Management */

func (l *Goto) Save() error {
	CurrentLogger().Debugf("Saving goto %s...", l.Name)
	query := `
		INSERT INTO goto(
			oid,
			packfile_oid,
			note_oid,
			relative_path,
			"text",
			url,
			title,
			name,
			created_at,
			updated_at,
			indexed_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			note_oid = ?,
			relative_path = ?,
			"text" = ?,
			url = ?,
			title = ?,
			name = ?,
			updated_at = ?,
			indexed_at = ?
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
		l.Name,
		timeToSQL(l.CreatedAt),
		timeToSQL(l.UpdatedAt),
		timeToSQL(l.IndexedAt),
		// Update
		l.PackFileOID,
		l.NoteOID,
		l.RelativePath,
		l.Text,
		l.URL,
		l.Title,
		l.Name,
		timeToSQL(l.UpdatedAt),
		timeToSQL(l.IndexedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (l *Goto) SaveMetadata() error {
	// No operation-related fields for now
	return nil
}

func (l *Goto) Delete() error {
	CurrentLogger().Debugf("Deleting goto %s...", l.Name)
	query := `DELETE FROM goto WHERE oid = ? AND packfile_oid = ?;`
	_, err := CurrentDB().Client().Exec(query, l.OID, l.PackFileOID)
	return err
}

/* SQL Queries */

// CountGotos returns the total number of gotos.
func (r *Repository) CountGotos() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM goto`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) LoadGotoByOID(oid oid.OID) (*Goto, error) {
	return QueryGoto(CurrentDB().Client(), "WHERE oid = ?", oid)
}

func (r *Repository) LoadGotos() ([]*Goto, error) {
	return QueryGotos(CurrentDB().Client(), "")
}

func (r *Repository) FindGotoByName(name string) (*Goto, error) {
	return QueryGoto(CurrentDB().Client(), "WHERE name = ?", name)
}

func (r *Repository) FindGotosByText(text string) ([]*Goto, error) {
	return QueryGotos(CurrentDB().Client(), "WHERE text = ?", text)
}

/* SQL Helpers */

func QueryGoto(db SQLClient, whereClause string, args ...any) (*Goto, error) {
	var l Goto
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
			name,
			created_at,
			updated_at,
			indexed_at
		FROM goto
		%s;`, whereClause), args...).
		Scan(
			&l.OID,
			&l.PackFileOID,
			&l.NoteOID,
			&l.RelativePath,
			&l.Text,
			&l.URL,
			&l.Title,
			&l.Name,
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
	l.IndexedAt = timeFromSQL(lastIndexedAt)

	return &l, nil
}

func QueryGotos(db SQLClient, whereClause string, args ...any) ([]*Goto, error) {
	var links []*Goto

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			note_oid,
			relative_path,
			"text",
			url,
			title,
			name,
			created_at,
			updated_at,
			indexed_at
		FROM goto
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var l Goto
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
			&l.Name,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
		)
		if err != nil {
			return nil, err
		}

		l.CreatedAt = timeFromSQL(createdAt)
		l.UpdatedAt = timeFromSQL(updatedAt)
		l.IndexedAt = timeFromSQL(lastIndexedAt)
		links = append(links, &l)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return links, err
}

/* ParameterizedURL */

// ParameterizedURL represents a URL with optional placeholders. Ex:
// http://${domain}
// https://www.github.com/${owner}/${repo}/${section:[issues,pulls,...]}
type ParameterizedURL string

// Short removes optional allowed values from the URL to have a shorter representation
// Ex: https://www.github.com/${owner}/${repo}/${section:[issues,pulls,...]} => https://www.github.com/${owner}/${repo}/${section}
func (u ParameterizedURL) Short() ParameterizedURL {
	result := string(u)
	// Remove all allowed values from the URL
	re := regexp.MustCompile(`\$\{([^}:]+):\[[^\]]*\]\}`)
	result = re.ReplaceAllString(result, "${$1}")
	return ParameterizedURL(result)
}

// Expand replaces placeholders in the URL with the corresponding values from the map.
func (u ParameterizedURL) Expand(values map[string]string) ParameterizedURL {
	result := string(u)
	for name, value := range values {
		// Replace all occurrences of this placeholder
		re := regexp.MustCompile(`\$\{` + regexp.QuoteMeta(name) + `(?::\[[^\]]*\])?\}`)
		result = re.ReplaceAllString(result, value)
	}
	return ParameterizedURL(result)
}

// Placeholders extracts all placeholders from the URL.
func (u ParameterizedURL) Placeholders() []Placeholder {
	return ExtractPlaceholders(string(u))
}

// ToANSI highlights placeholders in red using ANSI escape codes.
// Ex: https://github.com/${user} => https://github.com/\033[31m${user}\033[0m
func (u ParameterizedURL) ToANSI() string {
	re := regexp.MustCompile(`(\$\{[^}]+\})`)
	return re.ReplaceAllStringFunc(string(u), func(m string) string {
		return fmt.Sprintf("\033[31m%s\033[0m", m)
	})
}

// Placeholder represents a URL placeholder variable
type Placeholder struct {
	Name     string   // Variable name (e.g., "page")
	Raw      string   // Full placeholder text (e.g., "${page:[issues,pulls]}")
	Options  []string // Specific allowed values, empty if no value provided
	Ellipsis bool     // Additionanl values are allowed
}

// String returns a human-readable description of the placeholder
func (p Placeholder) String() string {
	if len(p.Options) > 0 && !p.Ellipsis {
		return fmt.Sprintf("%s (choose from: %s)", p.Name, strings.Join(p.Options, ", "))
	} else if len(p.Options) > 0 && p.Ellipsis {
		return fmt.Sprintf("%s (suggestions: %s, or enter custom value)", p.Name, strings.Join(p.Options, ", "))
	}
	return fmt.Sprintf("%s (enter any value)", p.Name)
}

// ExtractPlaceholders extracts placeholders from a text string.
func ExtractPlaceholders(text string) []Placeholder {
	// Regex to match ${variable} or ${variable:[value1,value2,...]}
	re := regexp.MustCompile(`\$\{([^}:]+)(?::?\[([^\]]*)\])?\}`)
	matches := re.FindAllStringSubmatch(text, -1)

	if len(matches) == 0 {
		return nil
	}

	var placeholders []Placeholder
	for _, match := range matches {
		placeholder := Placeholder{
			Name: match[1],
			Raw:  match[0],
		}

		// Parse allowed values if present
		if len(match) > 2 && match[2] != "" {
			var options []string
			values := strings.Split(match[2], ",")
			for _, value := range values {
				value = strings.TrimSpace(value)
				if value == "..." || value == "…" {
					placeholder.Ellipsis = true
				} else {
					options = append(options, value)
				}
			}

			placeholder.Options = options
		}

		placeholders = append(placeholders, placeholder)
	}

	return placeholders
}
