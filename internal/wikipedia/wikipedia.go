package wikipedia

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/julien-sobczak/the-notetaker/internal/core"
)

// Wikipedia provides reference management using Wikipedia API.
type Wikipedia struct {
}

func NewReferenceManager() *Wikipedia {
	return &Wikipedia{}
}

// Module query structure

type QueryResponse struct {
	Query Query `json:"query"`
}
type Query struct {
	Search []Result `json:"search"`
}
type Result struct {
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
	Text   interface{} `json:"text"`
}

func (p Parse) RawText() string {
	return p.Text.(map[string]interface{})["*"].(string)
}

func (w *Wikipedia) Search(query string) (core.Reference, error) {

	// Search for Wikipedia pages
	response := search(query)

	// Ask user to choose a page to continue
	pageID := askPageID(response)

	// Retrieve Wikipedia page content
	pageContent := get(pageID)

	// Load the HTML document
	infobox := parseWikitext(pageContent)

	// Ask user to choose the attributes to keep
	var options []string
	for _, attribute := range infobox.Attributes {
		options = append(options, attribute.Key+": "+truncateText(fmt.Sprintf("%v", attribute.Value), 30)) // TODO format value for terminal
	}
	answers := []string{}
	prompt := &survey.MultiSelect{
		Message: "Which attributes?",
		Options: options,
	}
	survey.AskOne(prompt, &answers)

	return nil, nil
}

func search(query string) QueryResponse {
	requestURL := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&utf8=&format=json", url.QueryEscape(query))
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
		fmt.Printf("Error unmarshalling JSON response: %s\n", err)
		os.Exit(1)
	}
	return response
}

func get(pageID int) string {
	requestURL := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=parse&contentmodel=text&pageid=%d&prop=wikitext&format=json", pageID)
	resp, err := http.Get(requestURL)
	if err != nil {
		fmt.Printf("Error making HTTP request: %s\n", err)
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Wrong status code for HTTP request: %v\n", resp.StatusCode)
		os.Exit(1)
	}

	var response ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		fmt.Printf("Error unmarshalling JSON response: %s\n", err)
		os.Exit(1)
	}
	return response.Parse.RawText()
}

func askPageID(response QueryResponse) int {
	pageIDs := make(map[string]int)
	var options []string
	for _, result := range response.Query.Search {
		options = append(options, result.Title)
		pageIDs[result.Title] = result.PageID
	}

	var answer string
	prompt := &survey.Select{
		Message: "Which page?",
		Options: options,
	}
	survey.AskOne(prompt, &answer, survey.WithValidator(survey.Required))
	return pageIDs[answer]
}

/* Helpers */

func truncateText(s string, max int) string {
	if max > len(s) {
		return s
	}
	return s[:max]
}
