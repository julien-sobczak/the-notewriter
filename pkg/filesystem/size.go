package filesystem

import (
	"log"
	"os"
	"path/filepath"
)

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

// FileSize returns the size for a single file in bytes.
func FileSize(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

// DirSize returns the total size for a directory in bytes
// by recursively iterating on files.
func DirSize(path string) (int64, error) {
	var size int64
	entries, err := os.ReadDir(path)
	if err != nil {
		return size, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			subDirSize, err := DirSize(path + "/" + entry.Name())
			if err != nil {
				log.Printf("failed to calculate size of directory %s: %v\n", entry.Name(), err)
				continue
			}
			size += subDirSize
		} else {
			fileInfo, err := entry.Info()
			if err != nil {
				log.Printf("failed to get info of file %s: %v\n", entry.Name(), err)
				continue
			}
			size += fileInfo.Size()
		}
	}
	return size, nil
}

// ListFiles lists all files present in a directory recursively.
func ListFiles(path string) ([]string, error) {
	var paths []string
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}
