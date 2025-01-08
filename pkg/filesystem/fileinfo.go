package filesystem

import (
	"os"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/resync"
)

var (
	// Lazy-load
	fileInfoReaderOnce      resync.Once
	fileInfoReaderSingleton FileInfoReader
)

type FileInfoReader interface {
	Lstat(name string) (os.FileInfo, error)
	Stat(name string) (os.FileInfo, error)
}

type StandardFileInfoReader struct{}

func (r StandardFileInfoReader) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (r StandardFileInfoReader) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

// A FileInfo describes a file and is returned by [Stat].
type testFileInfo struct {
	stat os.FileInfo
}

func (fi testFileInfo) Name() string {
	return fi.stat.Name()
}
func (fi testFileInfo) Size() int64 {
	if fi.stat.Size() == 0 {
		return 0
	}
	return 1 // Reproducible
}
func (fi testFileInfo) Mode() os.FileMode {
	return fi.stat.Mode()
}
func (fi testFileInfo) ModTime() time.Time {
	// Return now
	return clock.Now()
}
func (fi testFileInfo) IsDir() bool {
	return fi.stat.IsDir()
}
func (fi testFileInfo) Sys() any {
	return fi.stat.Sys()
}

// ClockBasedFileInfoReader is a FileInfoReader implementation that can be used to have reproductible tests.
// It relies on Clock to control the time and uses a default size 0 for empty file and 1 for any non-empty file.
type ClockBasedFileInfoReader struct{}

func NewClockBasedFileInfoReader() *ClockBasedFileInfoReader {
	return &ClockBasedFileInfoReader{}
}

func (r ClockBasedFileInfoReader) Stat(name string) (os.FileInfo, error) {
	// Execute the real Stat function to reproduce errors
	stat, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	// Override non-reproducible fields
	return testFileInfo{stat}, nil
}

func (r ClockBasedFileInfoReader) Lstat(name string) (os.FileInfo, error) {
	// Execute the real Lstat function to reproduce errors
	stat, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	// Override non-reproducible fields
	return testFileInfo{stat}, nil
}

func CurrentFileInfoReader() FileInfoReader {
	if fileInfoReaderSingleton != nil {
		return fileInfoReaderSingleton
	}
	fileInfoReaderOnce.Do(func() {
		fileInfoReaderSingleton = StandardFileInfoReader{}
	})
	return fileInfoReaderSingleton
}

// Same as os.Stat() but makes possible to control time from unit tests.
func Stat(name string) (os.FileInfo, error) {
	return CurrentFileInfoReader().Stat(name)
}

// Same as os.Lstat() but makes possible to control time from unit tests.
func Lstat(name string) (os.FileInfo, error) {
	return CurrentFileInfoReader().Lstat(name)
}

func OverrideFileInfoReader(reader FileInfoReader) {
	fileInfoReaderSingleton = reader
}

func RestoreFileInfoReader() {
	fileInfoReaderSingleton = nil
	fileInfoReaderOnce.Reset()
}
