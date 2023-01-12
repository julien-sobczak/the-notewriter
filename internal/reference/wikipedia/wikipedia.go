package wikipedia

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/julien-sobczak/the-notetaker/internal/reference"
	sopts "go.nhat.io/surveyexpect/options"
)

// WikipediaReference satisfies the Reference interface
// and retrieves metadata using the Wikipedia API.
type WikipediaReference struct {
	PageID     int
	PageTitle  string
	URL        string
	attributes []reference.Attribute
}

func (r *WikipediaReference) Attributes() []reference.Attribute {
	var results []reference.Attribute
	results = append(results, reference.Attribute{
		Key:   "name",
		Value: r.PageTitle,
	})
	results = append(results, reference.Attribute{
		Key:   "pageId",
		Value: r.PageID,
	})
	results = append(results, reference.Attribute{
		Key:   "url",
		Value: r.URL,
	})
	results = append(results, r.attributes...)
	return results
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
	Text   interface{} `json:"wikitext"`
}

func (p Parse) RawText() string {
	return p.Text.(map[string]interface{})["*"].(string)
}

// Wikipedia provides reference management using Wikipedia API.
type Wikipedia struct {
	// Override in tests to use a mock server
	BaseURL string
	// Override in tests to capture in/out
	Stdio *terminal.Stdio
}

func NewReferenceManager() *Wikipedia {
	return &Wikipedia{
		BaseURL: "https://en.wikipedia.org",
	}
}

func (w *Wikipedia) Search(query string) (reference.Reference, error) {

	// Search for Wikipedia pages
	queryResponse := w.search(query)

	// Ask user to choose a page to continue
	pageID := w.askPageID(queryResponse)

	// Retrieve Wikipedia page content
	pageResponse := w.get(pageID)

	// Load the HTML document
	infobox := parseWikitext(pageResponse.Parse.RawText())

	// Ask user to choose the most relevant attributes
	attributes := w.askAttributes(infobox)

	result := &WikipediaReference{
		PageID:     pageID,
		PageTitle:  pageResponse.Parse.Title,
		URL:        WikipediaURL(pageID, pageResponse.Parse.Title),
		attributes: attributes,
	}
	return result, nil
}

func (w *Wikipedia) search(query string) QueryResponse {
	requestURL := fmt.Sprintf("%s/w/api.php?action=query&list=search&srsearch=%s&utf8=&format=json", w.BaseURL, url.QueryEscape(query))
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
		fmt.Printf("Error unmarshalling query JSON response: %s\n", err)
		os.Exit(1)
	}
	return response
}

func (w *Wikipedia) get(pageID int) *ParseResponse {
	requestURL := fmt.Sprintf("%s/w/api.php?action=parse&contentmodel=text&pageid=%d&prop=wikitext&format=json", w.BaseURL, pageID)
	resp, err := http.Get(requestURL)
	if err != nil {
		fmt.Printf("Error making HTTP request: %s\n", err)
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
		fmt.Printf("Error unmarshalling parse JSON response: %s\n", err)
		os.Exit(1)
	}
	return &response
}

func (w *Wikipedia) askPageID(response QueryResponse) int {
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
	var surveyOpts []survey.AskOpt
	surveyOpts = append(surveyOpts, survey.WithValidator(survey.Required))
	if w.Stdio != nil {
		surveyOpts = append(surveyOpts, sopts.WithStdio(*w.Stdio))
	}
	survey.AskOne(prompt, &answer, surveyOpts...)
	return pageIDs[answer]
}

func (w *Wikipedia) askAttributes(infobox *Infobox) []reference.Attribute {
	var options []string
	answersIndices := make(map[string]int)
	for i, attribute := range infobox.Attributes {
		// TODO format value for terminal
		optionText := attribute.Key + ": " + truncateText(fmt.Sprintf("%v", attribute.Value), 30)
		options = append(options, optionText)
		answersIndices[optionText] = i
	}
	answers := []string{}
	prompt := &survey.MultiSelect{
		Message: "Which attributes?",
		Options: options,
	}
	var surveyOpts []survey.AskOpt
	if w.Stdio != nil {
		surveyOpts = append(surveyOpts, sopts.WithStdio(*w.Stdio))
	}
	survey.AskOne(prompt, &answers, surveyOpts...)

	var selection []reference.Attribute
	for _, answer := range answers {
		selection = append(selection, infobox.Attributes[answersIndices[answer]])
	}
	return selection
}

/* Helpers */

// Wikipedia generates the Wikipedia URL from the page ID and title.
func WikipediaURL(pageId int, pageTitle string) string {
	return fmt.Sprintf("https://en.wikipedia.org/wiki/%s", strings.ReplaceAll(pageTitle, " ", "_"))
}

func truncateText(s string, max int) string {
	if max > len(s) {
		return s
	}
	return s[:max]
}
