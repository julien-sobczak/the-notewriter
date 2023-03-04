package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/internal/reference"
	"github.com/julien-sobczak/the-notetaker/internal/reference/wikipedia"
	"github.com/julien-sobczak/the-notetaker/internal/reference/zotero"
	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/julien-sobczak/the-notetaker/pkg/resync"
	"gopkg.in/yaml.v3"
)

const ReferenceKindBook = "book"
const ReferenceKindAuthor = "author"

var (
	// Lazy-load configuration and ensure a single read
	collectionOnce      resync.Once
	collectionSingleton *Collection
)

type Collection struct {
	OID string `yaml:"oid"`

	Path          string `yaml:"path"`
	bookManager   reference.Manager
	personManager reference.Manager

	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	LastCheckedAt time.Time `yaml:"-"`

	new bool
}

func CurrentCollection() *Collection {
	collectionOnce.Do(func() {
		var err error
		zoteroManager := zotero.NewReferenceManager()
		wikipediaManager := wikipedia.NewReferenceManager()
		collectionSingleton, err = NewCollection(zoteroManager, wikipediaManager)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to init current collection: %v\n", err)
			os.Exit(1)
		}
	})
	return collectionSingleton
}

func NewCollection(bookManager reference.Manager, personManager reference.Manager) (*Collection, error) {
	config := CurrentConfig()

	absolutePath, err := filepath.Abs(config.RootDirectory)
	if err != nil {
		return nil, err
	}

	c := &Collection{
		OID:           NewOID(),
		Path:          absolutePath,
		bookManager:   bookManager,
		personManager: personManager,
		CreatedAt:     clock.Now(),
		UpdatedAt:     clock.Now(),
		new:           true,
	}
	return c, nil
}

func (c *Collection) CreateNewReferenceFile(identifier string, kind string) (*File, error) {
	var ref reference.Reference
	var err error

	switch kind {
	case ReferenceKindBook:
		ref, err = c.bookManager.Search(identifier)
	case ReferenceKindAuthor:
		ref, err = c.personManager.Search(identifier)
	}
	if err != nil {
		return nil, err
	}

	var attributes []Attribute
	for _, refAttribute := range ref.Attributes() {
		attributes = append(attributes, Attribute{
			Key:   refAttribute.Key,
			Value: refAttribute.Value,
		})
	}

	return NewFileFromAttributes("", attributes), nil // FIXME use a name
}

/* Object */

func (c *Collection) Kind() string {
	return "collection"
}

func (c *Collection) UniqueOID() string {
	return c.OID
}

func (c *Collection) ModificationTime() time.Time {
	return c.UpdatedAt
}

func (c *Collection) State() State {
	return Modified
}

func (c *Collection) ForceState(state State) {
}

func (c *Collection) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(c)
	if err != nil {
		return err
	}
	return nil
}

func (c *Collection) Write(w io.Writer) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (c *Collection) SubObjects() []StatefulObject {
	return nil
}

func (c *Collection) Blobs() []BlobRef {
	// Use Media.Blobs() instead
	return nil
}

func (c Collection) String() string {
	return fmt.Sprintf("collection [%s]", c.OID)
}

/* Reference Management */

func (c *Collection) AddNewReferenceFile(identifier string, kind string) error {
	f, err := c.CreateNewReferenceFile(identifier, kind)
	if err != nil {
		return err
	}
	return f.SaveOnDisk()
}

func (c *Collection) Close() {
	CurrentDB().Close()
}

// GetNoteRelativePath converts a relative path from a note to a relative path from the collection root directory.
func (c *Collection) GetNoteRelativePath(fileRelativePath string, srcPath string) (string, error) {
	return filepath.Rel(c.Path, filepath.Join(filepath.Dir(c.GetAbsolutePath(fileRelativePath)), srcPath))
}

// GetFileRelativePath converts a relative path of a file to a relative path from the collection.
func (c *Collection) GetFileRelativePath(fileAbsolutePath string) (string, error) {
	return filepath.Rel(c.Path, fileAbsolutePath)
}

// GetAbsolutePath converts a relative path from the collection to an absolute path on disk.
func (c *Collection) GetAbsolutePath(path string) string {
	if strings.HasPrefix(path, c.Path) {
		return path
	}
	return filepath.Join(c.Path, path)
}

func (c *Collection) Save(tx *sql.Tx) error {
	var err error
	switch c.State() {
	case Added:
		err = c.InsertWithTx(tx)
	case Modified:
		err = c.UpdateWithTx(tx)
	}
	c.new = false
	return err
}

func (c *Collection) OldSave() error { // FIXME remove deprecated
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = c.SaveWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (c *Collection) SaveWithTx(tx *sql.Tx) error { // FIXME remove deprecated
	now := clock.Now()
	c.UpdatedAt = now
	c.LastCheckedAt = now

	if !c.new {
		return c.UpdateWithTx(tx)
	} else {
		return c.InsertWithTx(tx)
	}
}

func (c *Collection) InsertWithTx(tx *sql.Tx) error {
	query := `
		INSERT INTO collection(
			oid,
			created_at,
			updated_at,
			last_checked_at)
		VALUES (?, ?, ?, ?);
	`

	_, err := tx.Exec(query,
		c.OID,
		timeToSQL(c.CreatedAt),
		timeToSQL(c.UpdatedAt),
		timeToSQL(c.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *Collection) UpdateWithTx(tx *sql.Tx) error {
	query := `
		UPDATE collection
		SET
			updated_at = ?,
			last_checked_at = ?
		WHERE oid = ?;
	`

	_, err := tx.Exec(query,
		timeToSQL(c.UpdatedAt),
		timeToSQL(c.LastCheckedAt),
		c.OID,
	)
	return err
}

func LoadCollection() (*Collection, error) {
	c, err := QueryCollection("")
	if err == sql.ErrNoRows {
		return nil, errors.New("unknown collection")
	}

	return c, nil
}

/* SQL Helpers */

func QueryCollection(whereClause string, args ...any) (*Collection, error) {
	db := CurrentDB().Client()

	var c Collection
	var createdAt string
	var updatedAt string
	var lastCheckedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			created_at,
			updated_at,
			last_checked_at
		FROM file
		%s;`, whereClause), args...).
		Scan(&c.OID, &createdAt, &updatedAt, &lastCheckedAt); err != nil {

		return nil, err
	}

	c.CreatedAt = timeFromSQL(createdAt)
	c.UpdatedAt = timeFromSQL(updatedAt)
	c.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &c, nil
}

func QueryCollections(whereClause string, args ...any) ([]*Collection, error) {
	db := CurrentDB().Client()

	var collections []*Collection

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			created_at,
			updated_at,
			last_checked_at
		FROM file
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c Collection
		var createdAt string
		var updatedAt string
		var lastCheckedAt string

		err = rows.Scan(&c.OID, &createdAt, &updatedAt, &lastCheckedAt)
		if err != nil {
			return nil, err
		}
		c.CreatedAt = timeFromSQL(createdAt)
		c.UpdatedAt = timeFromSQL(updatedAt)
		c.LastCheckedAt = timeFromSQL(lastCheckedAt)
		collections = append(collections, &c)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return collections, err
}
