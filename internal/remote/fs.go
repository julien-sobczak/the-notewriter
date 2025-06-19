package remote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

/* FS */

type FS struct {
	path string
	// Use classic FS APIs to satisfy interface
}

func NewFS(dirpath string) (*FS, error) {
	stat, err := os.Stat(dirpath)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dirpath)
	}

	return &FS{
		path: dirpath,
	}, nil
}

func (r *FS) GetObject(key string) ([]byte, error) {
	path := filepath.Join(r.path, key)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrObjectNotExist
	}
	return data, err
}

func (r *FS) PutObject(key string, data []byte) error {
	dirPath := filepath.Join(r.path, filepath.Dir(key))
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return err
	}
	filePath := filepath.Join(r.path, key)
	return os.WriteFile(filePath, data, 0644)
}

func (r *FS) DeleteObject(key string) error {
	path := filepath.Join(r.path, key)
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrObjectNotExist
	}
	return os.Remove(path)
}

func (r *FS) GC() error {
	// Not implemented
	return nil
}
