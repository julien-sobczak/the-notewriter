package wikipedia

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/julien-sobczak/the-notewriter/internal/reference"
)

const (
	// How many Wikipedia pages to traverse
	maxResults = 3
)

// Module query structure

type QueryResponse struct {
	Query Query `json:"query"`
}
type Query struct {
	Results []QueryResult `json:"search"`
}
type QueryResult struct {
	Title  string `json:"title"`
	PageID int    `json:"pageid"`
}

// Module parse structure

type ParseResponse struct {
	Parse Parse `json:"parse"`
}
type Parse struct {
	Title  string      `json:"title"`
	PageID int         `json:"pageid"`
	Text   interface{} `json:"wikitext"`
}

func (p Parse) RawText() string {
	return p.Text.(map[string]interface{})["*"].(string)
}

// Manager provides reference management using Wikipedia API.
type Manager struct {
	// Override in tests to use a mock server
	BaseURL string
}

type Result struct {
	PageID     int
	PageTitle  string
	URL        string
	attributes map[string]any
}

func (r *Result) Description() string {
	return r.PageTitle
}

func (r *Result) Attributes() map[string]any {
	results := map[string]any{
		"name":   r.PageTitle,
		"pageId": r.PageID,
		"url":    r.URL,
	}
	for k, v := range r.attributes {
		results[k] = v
	}
	return results
}

func NewManager() *Manager {
	return &Manager{
		BaseURL: "https://en.wikipedia.org",
	}
}

/* Reference interface */

func (m *Manager) Ready() (bool, error) {
	// Nothing to start locally
	return true, nil
}

func (m *Manager) Search(query string) ([]reference.Result, error) {
	var results []reference.Result
	// Search for Wikipedia pages
	queryResponse := m.search(query)

	for i, queryResult := range queryResponse.Query.Results {
		if i > maxResults {
			// Limit the number of results to limit HTTP queries
			break
		}

		// Retrieve Wikipedia page content
		pageResponse := m.get(queryResult.PageID)

		// Load the HTML document
		infobox := parseWikitext(pageResponse.Parse.RawText())

		result := &Result{
			PageID:     queryResult.PageID,
			PageTitle:  pageResponse.Parse.Title,
			URL:        WikipediaURL(queryResult.PageID, pageResponse.Parse.Title),
			attributes: infobox.Attributes,
		}
		results = append(results, result)
	}

	return results, nil
}

func (m *Manager) search(query string) QueryResponse {
	requestURL := fmt.Sprintf("%s/w/api.php?action=query&list=search&srsearch=%s&utf8=&format=json", m.BaseURL, url.QueryEscape(query))
	res, err := http.Get(requestURL)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err)
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
	return response
}

func (m *Manager) get(pageID int) *ParseResponse {
	requestURL := fmt.Sprintf("%s/w/api.php?action=parse&contentmodel=text&pageid=%d&prop=wikitext&format=json", m.BaseURL, pageID)
	resp, err := http.Get(requestURL)
	if err != nil {
		fmt.Printf("Error making HTTP request: %v\n", err)
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Wrong status code for HTTP request: %v\n", resp.StatusCode)
		os.Exit(1)
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		os.Exit(1)
	}
	// Debug:
	// fmt.Println(string(b))

	var response ParseResponse
	err = json.NewDecoder(strings.NewReader(string(b))).Decode(&response)
	if err != nil {
		fmt.Printf("Error unmarshalling parse JSON response: %v\n", err)
		os.Exit(1)
	}
	return &response
}

/* Helpers */

// Wikipedia generates the Wikipedia URL from the page ID and title.
func WikipediaURL(pageId int, pageTitle string) string {
	return fmt.Sprintf("https://en.wikipedia.org/wiki/%s", strings.ReplaceAll(pageTitle, " ", "_"))
}
