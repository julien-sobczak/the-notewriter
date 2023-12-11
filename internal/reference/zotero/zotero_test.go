package zotero

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	manager, err := NewManager()
	require.NoError(t, err)
	manager.BaseURL = ts.URL

	time.Sleep(2 * time.Second)
	ready, err := manager.Ready()
	require.NoError(t, err)
	require.True(t, ready)

	results, err := manager.Search("Nelson Mandela")
	require.NoError(t, err)
	require.Len(t, results, 1)
	actual := results[0]
	expected := &Result{
		title: "The infinite game",
		attributes: map[string]any{
			"itemType":       "book",
			"title":          "The infinite game",
			"key":            "6SM7FTLC",
			"libraryCatalog": "Library of Congress ISBN",
			"place":          "New York",
			"publisher":      "Portfolio/Penguin",
			"date":           "2019",
			"numPages":       "251",
			"ISBN":           "9780735213500 9780525538837",
			"version":        float64(0),
			"tags": []any{
				map[string]any{
					"tag":  "Leadership",
					"type": float64(1),
				},
			},
			"abstractNote": "\"In finite games, like football or chess, the players are known, the rules are fixed, and the endpoint is clear. The winners and losers are easily identified. In infinite games, like business or politics or life itself, the players come and go, the rules are changeable, and there is no defined endpoint. There are no winners or losers in an infinite game; there is only ahead and behind.",
			"callNumber":   "HD57.7 .S54866 2019",
			"creators": []any{
				map[string]any{
					"creatorType": "author",
					"firstName":   "Simon",
					"lastName":    "Sinek",
				},
			},
		},
	}
	assert.Equal(t, expected, actual)
}
