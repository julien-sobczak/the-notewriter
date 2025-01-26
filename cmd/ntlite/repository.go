package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func (r *Repository) walk(fn func(path string, stat fs.FileInfo) error) error {
	filepath.WalkDir(r.Path, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		dirname := filepath.Base(path)
		if dirname == ".nt" {
			return fs.SkipDir // NB fs.SkipDir skip the parent dir when path is a file
		}

		relativePath, err := filepath.Rel(r.Path, path)
		if err != nil {
			// ignore the file
			return nil
		}

		// We look for only specific extension
		if !info.IsDir() && !strings.HasSuffix(relativePath, ".md") {
			// Nothing to do
			return nil
		}

		// Ignore certain file modes like symlinks
		fileInfo, err := os.Lstat(path) // NB: os.Stat follows symlinks
		if err != nil {
			// Ignore the file
			return nil
		}
		if !fileInfo.Mode().IsRegular() {
			// Exclude any file with a mode bit set (device, socket, named pipe, ...)
			// See https://pkg.go.dev/io/fs#FileMode
			return nil
		}

		// A file found to process using the callback
		err = fn(relativePath, fileInfo)
		if err != nil {
			return err
		}

		return nil
	})

	return nil
}

// Add implements the command `nt add`.`
func (r *Repository) Add() error {
	db := CurrentDB()

	// Run all queries inside the same transaction
	err := db.BeginTransaction()
	if err != nil {
		return err
	}
	defer db.RollbackTransaction()

	// Traverse all files
	err = r.walk(func(relativePath string, stat fs.FileInfo) error {
		file, err := NewOrExistingFile(relativePath)
		if err != nil {
			return err
		}

		if file.State() != None {
			if err := db.StageObject(file); err != nil {
				return fmt.Errorf("unable to stage modified object %s: %v", file.RelativePath, err)
			}
		}
		if err := file.Save(); err != nil {
			return nil
		}

		for _, object := range file.SubObjects() {
			if object.State() != None {
				if err := db.StageObject(object); err != nil {
					return fmt.Errorf("unable to stage modified object %s: %v", object, err)
				}
			}
			if err := object.Save(); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// (Not implemented) Find objects to delete by querying
	// the different tables for rows with last_indexed_at < :execution_time

	// Don't forget to commit
	if err := db.CommitTransaction(); err != nil {
		return err
	}
	// And to persist the index
	if err := db.index.Save(); err != nil {
		return err
	}

	return nil
}
