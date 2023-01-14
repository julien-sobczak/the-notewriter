package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/julien-sobczak/the-notetaker/internal/reference"
	"github.com/julien-sobczak/the-notetaker/internal/reference/wikipedia"
	"github.com/julien-sobczak/the-notetaker/internal/reference/zotero"
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
	Path          string
	bookManager   reference.Manager
	personManager reference.Manager
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
	// TODD
	// walk the file system to find stale files
	return nil
}
