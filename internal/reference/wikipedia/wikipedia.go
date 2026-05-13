package wikipedia

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/reference"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

const (
	// How many Wikipedia pages to traverse
	maxResults = 3

	// Default timeout for HTTP requests
	defaultTimeout = 10 * time.Second
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
	Title  string `json:"title"`
	PageID int    `json:"pageid"`
	Text   any    `json:"wikitext"`
}

func (p Parse) RawText() string {
	return p.Text.(map[string]any)["*"].(string)
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
	queryResponse, err := m.search(query)
	if err != nil {
		return nil, err
	}

	for i, queryResult := range queryResponse.Query.Results {
		if i > maxResults {
			// Limit the number of results to limit HTTP queries
			break
		}

		// Retrieve Wikipedia page content
		pageResponse, err := m.get(queryResult.PageID)
		if err != nil {
			return nil, err
		}

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

func (m *Manager) search(query string) (QueryResponse, error) {
	requestURL := fmt.Sprintf("%s/w/api.php?action=query&list=search&srsearch=%s&utf8=&format=json", m.BaseURL, url.QueryEscape(query))
	client := &http.Client{Timeout: defaultTimeout}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return QueryResponse{}, fmt.Errorf("error creating HTTP request for %s: %w", requestURL, err)
	}
	res, err := client.Do(req)
	if err != nil {
		curlCmd, _ := text.RequestToCurl(req)
		return QueryResponse{}, &reference.FetchError{Err: fmt.Errorf("error making HTTP request: %w", err), Cmd: curlCmd}
	}
	if res.StatusCode != http.StatusOK {
		curlCmd, _ := text.RequestToCurl(req)
		return QueryResponse{}, &reference.FetchError{Err: fmt.Errorf("wrong status code for HTTP request: %d", res.StatusCode), Cmd: curlCmd}
	}
	var response QueryResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return QueryResponse{}, fmt.Errorf("error unmarshalling query JSON response: %w", err)
	}
	return response, nil
}

func (m *Manager) get(pageID int) (*ParseResponse, error) {
	requestURL := fmt.Sprintf("%s/w/api.php?action=parse&contentmodel=text&pageid=%d&prop=wikitext&format=json", m.BaseURL, pageID)
	client := &http.Client{Timeout: defaultTimeout}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request for %s: %w", requestURL, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		curlCmd, _ := text.RequestToCurl(req)
		return nil, &reference.FetchError{Err: fmt.Errorf("error making HTTP request: %w", err), Cmd: curlCmd}
	}
	if resp.StatusCode != http.StatusOK {
		curlCmd, _ := text.RequestToCurl(req)
		return nil, &reference.FetchError{Err: fmt.Errorf("wrong status code for HTTP request: %d", resp.StatusCode), Cmd: curlCmd}
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	// Debug:
	// fmt.Println(string(b))

	var response ParseResponse
	err = json.NewDecoder(strings.NewReader(string(b))).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling parse JSON response: %w", err)
	}
	return &response, nil
}

/* Helpers */

// Wikipedia generates the Wikipedia URL from the page ID and title.
func WikipediaURL(pageId int, pageTitle string) string {
	return fmt.Sprintf("https://en.wikipedia.org/wiki/%s", strings.ReplaceAll(pageTitle, " ", "_"))
}
