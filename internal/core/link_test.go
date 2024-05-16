package core

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLink(t *testing.T) {

	t.Run("YAML", func(t *testing.T) {
		SetUpRepositoryFromTempDir(t)
		// Make tests reproductible
		UseFixedOID(t, "42d74d967d9b4e989502647ac510777ca1e22f4a")
		FreezeAt(t, time.Date(2023, time.Month(1), 1, 1, 12, 30, 0, time.UTC))

		fileSrc := NewEmptyFile("example.md")
		parsedNoteSrc := MustParseNote("## TODO: Backlog\n\n* [ ] Test", "")
		noteSrc := NewNote(fileSrc, nil, parsedNoteSrc)
		linkSrc := NewLink(noteSrc, "click here", "https://www.google.com", "", "g")

		// Marshall
		buf := new(bytes.Buffer)
		err := linkSrc.Write(buf)
		require.NoError(t, err)
		linkYAML := buf.String()
		assert.Equal(t, strings.TrimSpace(`
oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
note_oid: 42d74d967d9b4e989502647ac510777ca1e22f4a
relative_path: example.md
text: click here
url: https://www.google.com
title: ""
go_name: g
created_at: 2023-01-01T01:12:30Z
updated_at: 2023-01-01T01:12:30Z
`), strings.TrimSpace(linkYAML))

		// Unmarshall
		linkDest := new(Link)
		err = linkDest.Read(buf)
		require.NoError(t, err)
		linkSrc.new = false
		linkSrc.stale = false
		assert.EqualValues(t, linkSrc, linkDest)
	})

}
