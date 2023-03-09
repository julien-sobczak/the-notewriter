package medias

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMimeType(t *testing.T) {
	tests := []struct {
		extension string // input
		mimeType  string // output
	}{
		{".mp3", "audio/mpeg"},
		{".MP3", "audio/mpeg"},
		{".mp4", "video/mp4"},
		{".mp9999", "application/octet-stream"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.mimeType, MimeType(tt.extension))
	}
}
