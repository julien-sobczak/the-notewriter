package core

import (
	"database/sql"
	"fmt"
	"os"
)

const ReferenceKindBook = "book"
const ReferenceKindAuthor = "author"

type Collection struct {
	db            *sql.DB
	path          string
	bookManager   ReferenceManager
	personManager ReferenceManager
}

func NewCollection(path string, bookManager ReferenceManager, personManager ReferenceManager) (*Collection, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("collection %q doesn't exists", path)
	}
	db, err := sql.Open("sqlite3", "database.db")
	if err != nil {
		return nil, err
	}
	c := &Collection{
		db:            db,
		path:          path,
		bookManager:   bookManager,
		personManager: personManager,
	}
	return c, nil
}

func (c *Collection) createNewReferenceFile(identifier string, kind string) (*File, error) {
	var reference Reference
	var err error

	switch kind {
	case ReferenceKindBook:
		reference, err = c.bookManager.Search(identifier)
	case ReferenceKindAuthor:
		reference, err = c.personManager.Search(identifier)
	}
	if err != nil {
		return nil, err
	}

	return &File{
		Attributes: reference.Attributes(),
		Content:    "",
	}, nil
}

func (c *Collection) AddNewReferenceFile(identifier string, kind string) error {
	note, err := c.createNewReferenceFile(identifier, kind)
	if err != nil {
		return err
	}
	return note.Save()
}

func (c *Collection) Close() {
	c.db.Close()
}
