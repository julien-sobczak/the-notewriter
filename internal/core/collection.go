package core

import (
	"fmt"
	"os"
)

type Collection struct {
	path string
}

func NewCollection(path string) (*Collection, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("collection %q doesn't exists", path)
	}
	c := &Collection{
		path: path,
	}
	return c, nil
}

func (c *Collection) createNewReferenceNote(identifier string) (*Note, error) {
	return &Note{
		ID:   "1", // TODO generate unique ID
		Kind: KindReference,
		FrontMatter: map[string]interface{}{
			"DOI": "TODO",
			// complete with every possible attributes
		},
		Content: "Title by Author",
	}, nil
}

func (c *Collection) AddNewReferenceNote(identifier string) error {
	note, err := c.createNewReferenceNote(identifier)
	if err != nil {
		return err
	}
	return note.Save()
}
