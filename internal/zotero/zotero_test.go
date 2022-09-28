package zotero

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	score "github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.nhat.io/surveyexpect"
)

func init() {
	// disable color output for all prompts to simplify testing
	score.DisableColor = true
}

func TestSearch(t *testing.T) {
	// Ex: https://en.wikipedia.org/wiki/Nelson_Mandela

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search" {
			// Ex: curl -XPOST http://localhost:1969/search -H 'Content-Type: text/plain' -d '0525538836'
			fmt.Fprintln(w, `
[
  {
    "key": "6SM7FTLC",
    "version": 0,
    "itemType": "book",
    "creators": [
      {
        "firstName": "Simon",
        "lastName": "Sinek",
        "creatorType": "author"
      }
    ],
    "tags": [
      {
        "tag": "Leadership",
        "type": 1
      }
    ],
    "ISBN": "9780735213500 9780525538837",
    "abstractNote": "\"In finite games, like football or chess, the players are known, the rules are fixed, and the endpoint is clear. The winners and losers are easily identified. In infinite games, like business or politics or life itself, the players come and go, the rules are changeable, and there is no defined endpoint. There are no winners or losers in an infinite game; there is only ahead and behind.",
    "title": "The infinite game",
    "numPages": "251",
    "callNumber": "HD57.7 .S54866 2019",
    "place": "New York",
    "publisher": "Portfolio/Penguin",
    "date": "2019",
    "libraryCatalog": "Library of Congress ISBN"
  }
]`)
		}
	}))
	defer ts.Close()

	s := surveyexpect.Expect(func(s *surveyexpect.Survey) {
		s.ExpectSelect("Which reference?  [Use arrows to move, type to filter]").
			ExpectOptions(
				"> The infinite game",
			).
			Enter()
	})(t)
	s.Start(func(stdio terminal.Stdio) {
		manager := NewReferenceManager()
		manager.BaseURL = ts.URL
		manager.Stdio = &stdio

		reference, err := manager.Search("Nelson Mandela")
		require.NoError(t, err)
		file := core.NewFileFromAttributes(reference.Attributes())
		require.NoError(t, err)
		frontMatter, err := file.FrontMatterString()
		require.NoError(t, err)
		assert.Equal(t,
			strings.TrimSpace(`
creators:
- creatorType: author
  firstName: Simon
  lastName: Sinek
title: The infinite game
place: New York
publisher: Portfolio/Penguin
date: "2019"
numPages: "251"
ISBN: 9780735213500 9780525538837
`),
			strings.TrimSpace(frontMatter))
	})

}

/* Kind tests */

func TestZoteroReferenceOnBook(t *testing.T) {
	referenceJSON := `  {
    "key": "TW7KH7DX",
    "version": 0,
    "itemType": "book",
    "creators": [
      {
        "firstName": "Patrick",
        "lastName": "Lencioni",
        "creatorType": "author"
      }
    ],
    "tags": [
      {
        "tag": "Teams in the workplace",
        "type": 1
      }
    ],
    "ISBN": "9780787960759",
    "title": "The five dysfunctions of a team: a leadership fable",
    "edition": "1st ed",
    "place": "San Francisco",
    "publisher": "Jossey-Bass",
    "date": "2002",
    "numPages": "229",
    "callNumber": "HD66 .L456 2002",
    "libraryCatalog": "Library of Congress ISBN",
    "shortTitle": "The five dysfunctions of a team"
  }`
	var reference map[string]interface{}
	err := json.Unmarshal([]byte(referenceJSON), &reference)
	require.NoError(t, err)

	ref := ZoteroReference{
		fields: reference,
	}
	assert.Equal(t, "The five dysfunctions of a team: a leadership fable", ref.Title())
	assert.Equal(t, "The five dysfunctions of a team", ref.ShortTitle())
	assert.Equal(t, []string{"Patrick Lencioni"}, ref.Authors())
	assert.Equal(t, "2002", ref.PublicationYear())
	assert.Contains(t, ref.Attributes(), "ISBN")
}
