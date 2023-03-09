package medias

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomConverter(t *testing.T) {
	mediasDir := filepath.Join("testdata", "TestMedias/medias")
	outputDir := t.TempDir()

	converter := NewRandomConverter()

	src := filepath.Join(mediasDir, "waterfall.flac")
	dest := filepath.Join(outputDir, "out.avif")

	err := converter.ToAVIF(src, dest, OriginalSize())
	require.NoError(t, err)
	assert.FileExists(t, dest)
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`[a-z0-9]{32,}`), string(data))
}
