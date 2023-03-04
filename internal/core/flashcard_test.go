package core

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlashcard(t *testing.T) {
	// Make tests reproductible
	UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
	FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))

	t.Run("YAML", func(t *testing.T) {
		fileSrc := NewEmptyFile("example.md")
		noteSrc := NewNote(fileSrc, "Flashcard: Syntax", "Question\n---\nAnswer", 1)
		flashcardSrc := NewFlashcard(fileSrc, noteSrc)

		// Marshall
		buf := new(bytes.Buffer)
		err := flashcardSrc.Write(buf)
		require.NoError(t, err)
		flashcardYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
short_title: Syntax
file_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
note_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: example.md
type: 0
queue: 0
due: 0
interval: 1
ease_factor: 2500
repetitions: 0
lapses: 0
left: 0
front_markdown: Question
back_markdown: Answer
front_html: <p>Question</p>
back_html: <p>Answer</p>
front_text: Question
back_text: Answer
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`), strings.TrimSpace(flashcardYAML))

		// Unmarshall
		flashcardDest := new(Flashcard)
		err = flashcardDest.Read(buf)
		require.NoError(t, err)

		// Compare ignore certain attributes
		flashcardSrc.File = nil
		flashcardSrc.Note = nil
		flashcardSrc.new = false
		flashcardSrc.stale = false
		assert.EqualValues(t, flashcardSrc, flashcardDest)
	})

}
