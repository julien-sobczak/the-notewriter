package googlebooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/reference"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {

	t.Run("ISBN", func(t *testing.T) {
		// Setup mock server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ex: https://www.googleapis.com/books/v1/volumes?q=isbn:9780008505547
			if r.URL.Path == "/volumes" && r.URL.Query().Get("q") == "ibsn:9780008505547" {
				responseBody := testutil.GoldenFileNamed(t, "isbn-9780008505547.json")
				fmt.Fprintln(w, string(responseBody))
			}
		}))
		defer ts.Close()

		manager := NewManager()
		manager.BaseURL = ts.URL

		results, err := manager.Search("978-0008505547")
		require.NoError(t, err)
		require.Len(t, results, 1)

		// Check first result
		firstResult := results[0]
		assert.Equal(t, "Good Inside: A Guide to Becoming the Parent You Want to Be", firstResult.Description())
		actual := firstResult.Attributes()
		expected := map[string]any{
			"allowAnonLogging": false,
			"authors": []any{
				"Becky Kennedy",
			},
			"contentVersion": "preview-1.0.0",
			"description":    "Wildly popular parenting expert Dr Becky Kenned shares...",
			"id":             "b6GxzgEACAAJ",
			"imageLinks": map[string]any{
				"smallThumbnail": "http://books.google.com/books/content?id=b6GxzgEACAAJ&printsec=frontcover&img=1&zoom=5&source=gbs_api",
				"thumbnail":      "http://books.google.com/books/content?id=b6GxzgEACAAJ&printsec=frontcover&img=1&zoom=1&source=gbs_api",
			},
			"industryIdentifiers": []any{
				map[string]any{
					"identifier": "0008505543", "type": "ISBN_10",
				},
				map[string]any{
					"identifier": "9780008505547", "type": "ISBN_13",
				},
			},
			"kind":           "books#volume",
			"language":       "en",
			"maturityRating": "NOT_MATURE",
			"pageCount":      float64(0),
			"printType":      "BOOK",
			"publishedDate":  "2022-09-15",
			"publisher":      "HarperThorsons",
			"subtitle":       "A Guide to Becoming the Parent You Want to Be",
			"title":          "Good Inside",
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("Title", func(t *testing.T) {
		// Setup mock server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ex: https://www.googleapis.com/books/v1/volumes?q=good+inside
			if r.URL.Path == "/volumes" && r.URL.Query().Get("q") == "good inside" {
				responseBody := testutil.GoldenFileNamed(t, "good+inside.json")
				fmt.Fprintln(w, string(responseBody))
			}
		}))
		defer ts.Close()

		manager := NewManager()
		manager.BaseURL = ts.URL

		results, err := manager.Search("good inside")
		require.NoError(t, err)
		require.Len(t, results, maxResults)
		assert.Equal(t, "Good Inside: A Guide to Becoming the Parent You Want to Be", results[0].Description())
		assert.Equal(t, "Changing Business from the Inside Out: A Treehugger’s Guide to Working in Corporations", results[1].Description())
		assert.Equal(t, "Inside Science: Stories from the Field in Human and Animal Science", results[2].Description())
		assert.Equal(t, "Gardeners' Chronicle", results[3].Description())
		assert.Equal(t, "The Good Inside Best Outside Parent: A Practical Guide to Excellent Parenting", results[4].Description())
	})

}

func TestTemplates(t *testing.T) {
	template := `---
title: "{{index . "title" | title}}{{ if index . "subtitle"}}:{{index . "subtitle" | title}}{{end}}"
short_title: "{{index . "title" | title}}"
name: {{index . "authors" | join ", "}}
occupation: Unknown
nationality: Unknown
{{- if index . "publishedDate"}}
date: "{{index . "publishedDate"}}"
{{- end -}}
{{- if index . "publisher"}}
publisher: {{index . "publisher"}}
{{- end -}}
{{- if index . "pageCount"}}
numPages: {{index . "pageCount"}}
{{- end -}}
{{- if index . "unknown"}}
unknown: {{index . "unknown"}}
{{- end -}}
{{- if index . "industryIdentifiers"}}
isbn: "{{index . "industryIdentifiers" | jq ". | first | .identifier"}}"
{{- end }}
---

# {{index . "title" | title}}
`
	results := LoadResultsFromGoldenFileNamed(t, "isbn-9780008505547.json")
	result := results[0]
	actual, err := reference.EvaluateTemplate(template, result)
	require.NoError(t, err)
	expected := `---
title: "Good Inside:A Guide to Becoming the Parent You Want to Be"
short_title: "Good Inside"
name: Becky Kennedy
occupation: Unknown
nationality: Unknown
date: "2022-09-15"
publisher: HarperThorsons
isbn: "0008505543"
---

# Good Inside
`
	assert.Equal(t, expected, actual)
}

/* Test Helpers */

// LoadResultsFromGoldenFileNamed parses a JSON Google Book response to convert them to Result.
func LoadResultsFromGoldenFileNamed(t *testing.T, filename string) []*Result {
	content := testutil.GoldenFileNamed(t, filename)
	var response QueryResponse
	err := json.Unmarshal(content, &response)
	require.NoError(t, err)

	var results []*Result
	for _, item := range response.Items {
		results = append(results, NewResultFromItem(item))
	}
	return results
}
