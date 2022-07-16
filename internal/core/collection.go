package core

import (
	"fmt"
	"os"
	"strings"
)

type Collection struct {
	path             string
	referenceManager ReferenceManager
}

func NewCollection(path string, referenceManager ReferenceManager) (*Collection, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("collection %q doesn't exists", path)
	}
	c := &Collection{
		path:             path,
		referenceManager: referenceManager,
	}
	return c, nil
}

func (c *Collection) createNewReferenceNote(identifier string) (*Note, error) {
	reference, err := c.referenceManager.Search(identifier)
	if err != nil {
		return nil, err
	}

	id := strings.ReplaceAll(fmt.Sprintf("%s-%s-%s", reference.PublicationYear(), reference.Authors()[0], reference.ShortTitle()), " ", "-") // TODO use all authors and add unit test
	return &Note{
		ID:   id,
		Kind: KindReference,
		FrontMatter: map[string]interface{}{
			"type":       reference.Type(),
			"title":      reference.Title(),
			"shortTitle": reference.ShortTitle(),
			"date":       reference.PublicationYear(),
			"details":    reference.Attributes(),
		},
		Content: reference.Bibliography(),
	}, nil
}

func (c *Collection) AddNewReferenceNote(identifier string) error {
	note, err := c.createNewReferenceNote(identifier)
	if err != nil {
		return err
	}
	return note.Save()
}
