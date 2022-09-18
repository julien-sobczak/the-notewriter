package core

import (
	"fmt"
	"os"
)

const ReferenceKindBook = "book"
const ReferenceKindAuthor = "author"

type Collection struct {
	path          string
	bookManager   ReferenceManager
	personManager ReferenceManager
}

func NewCollection(path string, bookManager ReferenceManager, personManager ReferenceManager) (*Collection, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("collection %q doesn't exists", path)
	}
	c := &Collection{
		path:          path,
		bookManager:   bookManager,
		personManager: personManager,
	}
	return c, nil
}

func (c *Collection) createNewReferenceNote(identifier string, kind string) (*Note, error) {
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

	return &Note{
		ID:               "XXX", // TODO add stable ID on notes?
		Kind:             KindReference,
		FrontMatter:      reference.Attributes(),
		FrontMatterOrder: reference.AttributesOrder(),
		Content:          "",
	}, nil
}

func (c *Collection) AddNewReferenceNote(identifier string, kind string) error {
	note, err := c.createNewReferenceNote(identifier, kind)
	if err != nil {
		return err
	}
	return note.Save()
}
