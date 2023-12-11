package googlebooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/julien-sobczak/the-notewriter/internal/reference"
)

const (
	// How many results to return in maximum
	maxResults = 5
)

// Module query structure

type QueryResponse struct {
	Kind       string  `json:"kind"`
	TotalItems int     `json:"totalItems"`
	Items      []*Item `json:"items"`
}
type Item struct {
	Kind       string         `json:"kind"`
	ID         string         `json:"id"`
	SelfLink   string         `json:"selfLink"`
	VolumeInfo map[string]any `json:"volumeInfo"`
}

// Manager provides reference management using Google Books API.
type Manager struct {
	// Overriden in tests to use a mock server
	BaseURL string
}

type Result struct {
	Kind       string
	ID         string
	SelfLink   string
	volumeInfo map[string]any
}

func (r *Result) Description() string {
	text := ""
	if title, ok := r.volumeInfo["title"]; ok {
		text += title.(string)
	}
	if subtitle, ok := r.volumeInfo["subtitle"]; ok {
		text += ": " + subtitle.(string)
	}
	if len(text) > 0 {
		return text
	}
	return fmt.Sprintf("%s (%s)", r.ID, r.Kind)
}

func (r *Result) Attributes() map[string]any {
	results := map[string]any{
		"id":   r.ID,
		"kind": r.Kind,
	}
	for k, v := range r.volumeInfo {
		results[k] = v
	}
	return results
}

func NewManager() *Manager {
	return &Manager{
		BaseURL: "https://www.googleapis.com/books/v1",
	}
}

/* Reference interface */

func (m *Manager) Ready() (bool, error) {
	// Nothing to start locally
	return true, nil
}

func (m *Manager) Search(query string) ([]reference.Result, error) {
	// Not complete, but match most occurrences
	regexISBN10 := regexp.MustCompile(`^([0-9]{9}X|[0-9]{10}|\d-?\d{6}-?\d{2}-?\d)$`) // Ex: 0-123456-47-9, 0123456479
	regexISBN13 := regexp.MustCompile(`^\d{3}-?\d-?\d{6}-?\d{2}-?\d$`)                // Ex: 978-0-123456-47-2, 978-0123456472, 9780123456472

	q := query // Ex: https://www.googleapis.com/books/v1/volumes?q=good+inside

	// Optimization: Use typed search for ISBNs to speed up queries
	// and avoid unnecessary processing on Google API side
	// Ex: https://www.googleapis.com/books/v1/volumes?q=isbn:9780008505547
	if regexISBN10.MatchString(q) || regexISBN13.MatchString(q) {
		q = fmt.Sprintf("ibsn:%s", ExtractDigits(q))
	}

	requestURL := fmt.Sprintf("%s/volumes?q=%s", m.BaseURL, url.QueryEscape(q))
	res, err := http.Get(requestURL)
	if err != nil {
		fmt.Printf("Error making HTTP request: %s\n", err)
		os.Exit(1)
	}
	if res.StatusCode != http.StatusOK {
		fmt.Printf("Wrong status code for HTTP request: %v\n", res.StatusCode)
		os.Exit(1)
	}

	var response QueryResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		fmt.Printf("Error unmarshalling query JSON response: %v\n", err)
		os.Exit(1)
	}

	var results []reference.Result
	for i, item := range response.Items {
		if i == maxResults {
			// Limit results to avoid long lists
			break
		}
		results = append(results, &Result{
			Kind:       item.Kind,
			ID:         item.ID,
			SelfLink:   item.SelfLink,
			volumeInfo: item.VolumeInfo,
		})
	}

	return results, nil
}

/* Helpers */

// ExtractDigits returns the digits by triming any other characters.
func ExtractDigits(txt string) string {
	re := regexp.MustCompile(`\d*`)

	var result bytes.Buffer

	submatchall := re.FindAllString(txt, -1)
	for _, element := range submatchall {
		result.WriteString(element)
	}

	return result.String()
}
