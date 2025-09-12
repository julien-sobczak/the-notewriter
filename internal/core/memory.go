package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

type Memory struct {
	OID oid.OID `yaml:"oid" json:"oid"`

	// Pack file where this object belongs
	PackFileOID oid.OID `yaml:"packfile_oid" json:"packfile_oid"`

	NoteOID oid.OID `yaml:"note_oid" json:"note_oid"`

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path" json:"relative_path"`

	// The text content of the memory
	Text markdown.Document `yaml:"text" json:"text"`

	// When the memory occurred
	OccurredAt time.Time `yaml:"occurred_at" json:"occurred_at"`

	// Timestamps to track changes
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
	IndexedAt time.Time `yaml:"indexed_at,omitempty" json:"indexed_at,omitempty"`
}

func NewOrExistingMemory(packFile *PackFile, note *Note, parsedMemory *ParsedMemory) (*Memory, error) {
	// Try to find an existing object (instead of recreating it from scratch after every change)
	existingMemory, err := CurrentRepository().FindMatchingMemory(parsedMemory, note.OID)
	if err != nil {
		return nil, err
	}

	var memory *Memory
	if existingMemory != nil {
		// Update existing memory
		memory = existingMemory
		memory.PackFileOID = packFile.OID
		memory.Text = parsedMemory.Text
		memory.UpdatedAt = clock.Now()
	} else {
		// Create new memory
		memory = &Memory{
			OID:          oid.New(),
			PackFileOID:  packFile.OID,
			NoteOID:      note.OID,
			RelativePath: note.RelativePath,
			Text:         parsedMemory.Text,
			OccurredAt:   parsedMemory.OccurredAt,
			CreatedAt:    clock.Now(),
			UpdatedAt:    clock.Now(),
		}
	}

	return memory, nil
}

func (m Memory) String() string {
	return fmt.Sprintf("memory %q occurred at %s", m.Text, m.OccurredAt.Format("2006-01-02"))
}

/* Object interface */

func (m *Memory) UniqueOID() oid.OID {
	return m.OID
}

func (m *Memory) Kind() string {
	return "memory"
}

func (m *Memory) ModificationTime() time.Time {
	return m.UpdatedAt
}

func (m *Memory) FileRelativePath() string {
	return m.RelativePath
}

func (m *Memory) PackFileRef() PackFileRef {
	return PackFileRef{
		OID: m.PackFileOID,
	}
}

func (m *Memory) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(m)
	return err
}

func (m *Memory) Write(w io.Writer) error {
	content, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}

func (m *Memory) Relations() []*Relation {
	// Memories don't create relations by default
	return nil
}

func (m *Memory) ToYAML() string {
	content, _ := yaml.Marshal(m)
	return string(content)
}

func (m *Memory) ToJSON() string {
	content, _ := json.Marshal(m)
	return string(content)
}

func (m *Memory) ToMarkdown() string {
	return fmt.Sprintf("Memory: %s (occurred on %s)", m.Text.String(), m.OccurredAt.Format("2006-01-02"))
}

/* StatefulObject interface */

func (m *Memory) Save() error {
	query := `
		INSERT INTO memory (
			oid,
			packfile_oid,
			note_oid,
			relative_path,
			text,
			occurred_at,
			created_at,
			updated_at,
			indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			note_oid = ?,
			relative_path = ?,
			text = ?,
			occurred_at = ?,
			updated_at = ?,
			indexed_at = ?
		;
	`
	_, err := CurrentDB().Client().Exec(query,
		// Insert
		string(m.OID),
		string(m.PackFileOID),
		string(m.NoteOID),
		m.RelativePath,
		m.Text.String(),
		timeToSQL(m.OccurredAt),
		timeToSQL(m.CreatedAt),
		timeToSQL(m.UpdatedAt),
		timeToSQL(m.IndexedAt),
		// Update
		string(m.PackFileOID),
		string(m.NoteOID),
		m.RelativePath,
		m.Text.String(),
		timeToSQL(m.OccurredAt),
		timeToSQL(m.UpdatedAt),
		timeToSQL(m.IndexedAt),
	)
	if err != nil {
		return err
	}

	CurrentLogger().Infof("Saving memory %s...", m.Text.TrimSpace().String())
	return nil
}

func (m *Memory) SaveMetadata() error {
	// No operation-related fields for now
	return nil
}

func (m *Memory) Delete() error {
	db := CurrentDB().Client()

	stmt, err := db.Prepare("DELETE FROM memory WHERE oid = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(string(m.OID))
	return err
}

/* Repository methods */

func (r *Repository) FindMatchingMemory(parsedMemory *ParsedMemory, noteOID oid.OID) (*Memory, error) {
	return QueryMemory(CurrentDB().Client(), `WHERE note_oid = ? AND occurred_at = ? AND text = ?`, noteOID, timeToSQL(parsedMemory.OccurredAt), parsedMemory.Text)
}

/* SQL Queries */

// CountMemories returns the total number of memories.
func (r *Repository) CountMemories() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM memory`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) FindMemories() ([]*Memory, error) {
	return QueryMemories(CurrentDB().Client(), "")
}

func (r *Repository) LoadMemoryByOID(oid oid.OID) (*Memory, error) {
	return QueryMemory(CurrentDB().Client(), `WHERE oid = ?`, oid)
}

func (r *Repository) LoadMemories() ([]*Memory, error) {
	return QueryMemories(CurrentDB().Client(), ``)
}

/* SQL Helpers */

func QueryMemory(db SQLClient, whereClause string, args ...any) (*Memory, error) {
	var m Memory

	var occurredAt string
	var createdAt string
	var updatedAt string
	var indexedAt sql.NullString

	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			note_oid,
			relative_path,
			text,
			occurred_at,
			created_at,
			updated_at,
			indexed_at
		FROM memory
		%s;`, whereClause), args...).
		Scan(
			&m.OID,
			&m.PackFileOID,
			&m.NoteOID,
			&m.RelativePath,
			&m.Text,
			&m.OccurredAt,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.IndexedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	m.OccurredAt = timeFromSQL(occurredAt)
	m.CreatedAt = timeFromSQL(createdAt)
	m.UpdatedAt = timeFromSQL(updatedAt)
	m.IndexedAt = timeFromNullableSQL(indexedAt)

	return &m, nil
}

func QueryMemories(db SQLClient, whereClause string, args ...any) ([]*Memory, error) {
	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			note_oid,
			relative_path,
			text,
			occurred_at,
			created_at,
			updated_at,
			indexed_at
		FROM memory
		%s
		ORDER BY occurred_at DESC;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []*Memory
	for rows.Next() {
		var memory Memory

		var occurredAt string
		var createdAt string
		var updatedAt string
		var indexedAt sql.NullString

		err := rows.Scan(
			&memory.OID,
			&memory.PackFileOID,
			&memory.NoteOID,
			&memory.RelativePath,
			&memory.Text,
			&occurredAt,
			&createdAt,
			&updatedAt,
			&indexedAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse timestamps
		memory.OccurredAt = timeFromSQL(occurredAt)
		memory.CreatedAt = timeFromSQL(createdAt)
		memory.UpdatedAt = timeFromSQL(updatedAt)
		memory.IndexedAt = timeFromNullableSQL(indexedAt)

		memories = append(memories, &memory)
	}

	return memories, nil
}
