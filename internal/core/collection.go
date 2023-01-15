package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/julien-sobczak/the-notetaker/internal/reference"
	"github.com/julien-sobczak/the-notetaker/internal/reference/wikipedia"
	"github.com/julien-sobczak/the-notetaker/internal/reference/zotero"
	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/julien-sobczak/the-notetaker/pkg/resync"
)

const ReferenceKindBook = "book"
const ReferenceKindAuthor = "author"

var (
	// Lazy-load configuration and ensure a single read
	collectionOnce      resync.Once
	collectionSingleton *Collection
)

type Collection struct {
	ID int64

	Path          string
	bookManager   reference.Manager
	personManager reference.Manager

	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastCheckedAt time.Time
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
		Path:          absolutePath,
		bookManager:   bookManager,
		personManager: personManager,
	}
	return c, nil
}

func (c *Collection) createNewReferenceFile(identifier string, kind string) (*File, error) {
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

	return NewFileFromAttributes(attributes), nil
}

func (c *Collection) AddNewReferenceFile(identifier string, kind string) error {
	f, err := c.createNewReferenceFile(identifier, kind)
	if err != nil {
		return err
	}
	return f.SaveOnDisk()
}

func (c *Collection) Close() {
	CurrentDB().Close()
}

// GetRelativePath converts a relative path from a note to a relative path from the collection.
func (c *Collection) GetRelativePath(referencePath string, srcPath string) (string, error) {
	return filepath.Rel(c.Path, filepath.Join(filepath.Dir(referencePath), srcPath))
}

// GetAbsolutePath converts a relative path from the collection to an absoluate path on disk.
func (c *Collection) GetAbsolutePath(relativePath string) string {
	return filepath.Join(c.Path, relativePath)
}

func (c *Collection) Save() error {
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

func (c *Collection) SaveWithTx(tx *sql.Tx) error {
	// TODO walk the file system to find stale files

	now := clock.Now()
	c.UpdatedAt = now
	c.LastCheckedAt = now

	if c.ID != 0 {
		return c.UpdateWithTx(tx)
	} else {
		return c.InsertWithTx(tx)
	}
}

func (c *Collection) InsertWithTx(tx *sql.Tx) error {
	query := `
		INSERT INTO collection(
			id,
			created_at,
			updated_at,
			last_checked_at)
		VALUES (NULL, ?, ?, ?);
	`

	res, err := tx.Exec(query,
		c.ID,
		timeToSQL(c.CreatedAt),
		timeToSQL(c.UpdatedAt),
		timeToSQL(c.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return err
	}
	c.ID = id

	return nil
}

func (c *Collection) UpdateWithTx(tx *sql.Tx) error {
	query := `
		UPDATE collection
		SET
			updated_at = ?,
			last_checked_at = ?
		WHERE id = ?;
	`

	_, err := tx.Exec(query,
		timeToSQL(c.UpdatedAt),
		timeToSQL(c.LastCheckedAt),
		c.ID,
	)
	return err
}

func LoadCollection() (*Collection, error) {
	c, err := querySingleCollection("")
	if err == sql.ErrNoRows {
		return nil, errors.New("unknown collection")
	}

	return c, nil
}

func querySingleCollection(whereClause string, args ...any) (*Collection, error) {
	db := CurrentDB().Client()

	var c Collection
	var createdAt string
	var updatedAt string
	var lastCheckedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			id,
			created_at,
			updated_at,
			last_checked_at
		FROM file
		%s;`, whereClause), args).
		Scan(&c.ID, &createdAt, &updatedAt, &lastCheckedAt); err != nil {

		return nil, err
	}

	c.CreatedAt = timeFromSQL(createdAt)
	c.UpdatedAt = timeFromSQL(updatedAt)
	c.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &c, nil
}
