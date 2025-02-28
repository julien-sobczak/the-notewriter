package medias

import (
	"os"
	"path/filepath"

	"github.com/julien-sobczak/the-notewriter/internal/helpers"
)

// RandomConverter generates files containing fake data.
// Useful in tests to avoid waiting for a command like ffmpeg to finish.
type RandomConverter struct {
	listeners []func(cmd string, args ...string)
}

func NewRandomConverter() *RandomConverter {
	return &RandomConverter{}
}

func (c *RandomConverter) OnPreGeneration(fn func(cmd string, args ...string)) {
	c.listeners = append(c.listeners, fn)
}

func (c *RandomConverter) notifyListeners(cmd string, args ...string) {
	for _, fn := range c.listeners {
		fn(cmd, args...)
	}
}

func (c *RandomConverter) ToAVIF(src, dest string, dimensions Dimensions) error {
	return c.toFakeFile(src, dest)
}

func (c *RandomConverter) ToMP3(src, dest string) error {
	return c.toFakeFile(src, dest)
}

func (c *RandomConverter) ToWebM(src, dest string) error {
	return c.toFakeFile(src, dest)
}

func (c *RandomConverter) toFakeFile(src, dest string) error {
	c.notifyListeners("convert", src, dest)
	hash := helpers.HashFromFileName(filepath.Base(dest)) // Ignore Dir as tests often uses t.TempDir()
	return os.WriteFile(dest, []byte(hash), 0644)
}
