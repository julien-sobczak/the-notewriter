package medias

import (
	"os"
	"path/filepath"

	"github.com/julien-sobczak/the-notetaker/internal/helpers"
)

// RandomConverter generates files containing fake data.
// Useful in tests to avoid waiting for a command like ffmpeg to finish.
type RandomConverter struct{}

func NewRandomConverter() *RandomConverter {
	return &RandomConverter{}
}

func (c *RandomConverter) ToAVIF(src, dest string, dimensions Dimensions) error {
	return c.toFakeFile(dest)
}

func (c *RandomConverter) ToMP3(src, dest string) error {
	return c.toFakeFile(dest)
}

func (c *RandomConverter) ToWebM(src, dest string) error {
	return c.toFakeFile(dest)
}

func (c *RandomConverter) toFakeFile(dest string) error {
	hash := helpers.HashFromFileName(filepath.Base(dest)) // Ignore Dir as tests often uses t.TempDir()
	return os.WriteFile(dest, []byte(hash), 0644)
}
