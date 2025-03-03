package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
)

// Lite version of internal/core/repository.go

var (
	repositoryOnce      sync.Once
	repositorySingleton *Repository
)

type Repository struct {
	Path string
}

func CurrentRepository() *Repository {
	repositoryOnce.Do(func() {
		var root string
		// Useful in tests when working with repositories in tmp directories
		if path, ok := os.LookupEnv("NT_HOME"); ok {
			root = path
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
			root = cwd
		}
		repositorySingleton = &Repository{
			Path: root,
		}
	})
	return repositorySingleton
}

func (r *Repository) walk(fn func(md *markdown.File) error) error {
	return filepath.WalkDir(r.Path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "." || path == ".." {
			return nil
		}

		dirname := filepath.Base(path)
		if dirname == ".nt" {
			return fs.SkipDir // NB fs.SkipDir skip the parent dir when path is a file
		}

		// We look for Markdown files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// A file found to process!
		md, err := markdown.ParseFile(path)
		if err != nil {
			return err
		}

		if err := fn(md); err != nil {
			return err
		}

		return nil
	})
}

// Add implements the command `nt add`.`
func (r *Repository) Add() error {
	db := CurrentDB()
	index := CurrentIndex()

	var traversedPaths []string
	var packFilesToUpsert []*PackFile

	// Traverse all given paths to detected updated medias/files
	err := r.walk(func(mdFile *markdown.File) error {
		relativePath, err := filepath.Rel(r.Path, mdFile.AbsolutePath)
		if err != nil {
			log.Fatalf("Unable to determine relative path: %v", err)
		}

		traversedPaths = append(traversedPaths, relativePath)

		if !index.Modified(relativePath, mdFile.MTime) {
			// Nothing changed = Nothing to parse
			return nil
		}

		// Reparse the new version
		parsedFile, err := ParseFile(relativePath, mdFile)
		if err != nil {
			return err
		}

		packFile, err := NewPackFileFromParsedFile(parsedFile)
		if err != nil {
			return err
		}
		packFilesToUpsert = append(packFilesToUpsert, packFile)

		return nil
	})
	if err != nil {
		return err
	}

	// We saved pack files on disk before starting a new transaction to keep it short
	if err := db.BeginTransaction(); err != nil {
		return err
	}
	db.UpsertPackFiles(packFilesToUpsert...)
	index.Stage(packFilesToUpsert...)

	// Don't forget to commit
	if err := db.CommitTransaction(); err != nil {
		return err
	}
	// And to persist the index
	if err := index.Save(); err != nil {
		return err
	}

	return nil
}

// Commit implements the command `nt commit`
func (r *Repository) Commit() error {
	return CurrentIndex().Commit()
}
