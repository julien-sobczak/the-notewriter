package zotero

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "embed"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/julien-sobczak/the-notetaker/internal/core"
	sopts "go.nhat.io/surveyexpect/options"
)

// TODO document this file

type ReferenceType string

const (
	// See https://github.com/zotero/zotero-schema/blob/master/schema.json for inspiration
	TypeArtwork          ReferenceType = "artwork"
	TypeAudioRecording   ReferenceType = "audioRecording"
	TypeBlogPost         ReferenceType = "blogPost"
	TypeBook             ReferenceType = "book"
	TypeBookSection      ReferenceType = "bookSection"
	TypeConferencePaper  ReferenceType = "conferencePaper"
	TypeDocument         ReferenceType = "document"
	TypeFilm             ReferenceType = "film"
	TypeJournalArticle   ReferenceType = "journalArticle"
	TypeMagazineArticle  ReferenceType = "magazineArticle"
	TypeLetter           ReferenceType = "letter"
	TypeNewspaperArticle ReferenceType = "newspaperArticle"
	TypePodcast          ReferenceType = "podcast"
	TypeThesis           ReferenceType = "thesis"
	TypeWebpage          ReferenceType = "webpage"
)

// List of supported types for references
var ReferenceTypes = []ReferenceType{
	TypeArtwork,
	TypeAudioRecording,
	TypeBlogPost,
	TypeBook,
	TypeBookSection,
	TypeConferencePaper,
	TypeDocument,
	TypeFilm,
	TypeJournalArticle,
	TypeMagazineArticle,
	TypeLetter,
	TypeNewspaperArticle,
	TypePodcast,
	TypeThesis,
	TypeWebpage,
}

func isSupportedType(itemType ReferenceType) bool {
	for _, supportedItemType := range ReferenceTypes {
		if supportedItemType == itemType {
			return true
		}
	}
	return false
}

//
// Schema management
//

//go:embed schema.json
var schemaFile []byte

// Retrieved from https://raw.githubusercontent.com/zotero/zotero-schema/master/schema.json
// Check fhttps://www.zotero.org/support/kb/item_types_and_fields for descriptions
var schemas map[string]ItemSchema

type Schema struct {
	Version   int          `json:"version"`
	ItemTypes []ItemSchema `json:"itemTypes"`
}
type ItemSchema struct {
	Type         string              `json:"itemType"`
	Fields       []FieldSchema       `json:"fields"`
	CreatorTypes []CreatorTypeSchema `json:"creatorTypes"`
}
type FieldSchema struct {
	Field     string `json:"field"`
	BaseField string `json:"baseField"`
}

type CreatorTypeSchema struct {
	Type    string `json:"creatorType"`
	Primary bool   `json:"primary"`
}

var ignoredFields = map[string]interface{}{
	"abstractNote":   nil,
	"callNumber":     nil,
	"libraryCatalog": nil,
}

func init() {
	// Parse the schema file
	var schema Schema
	err := json.Unmarshal(schemaFile, &schema)
	if err != nil {
		fmt.Printf("Unable to read Zotera schema file: %v", err)
		os.Exit(1)
	}

	schemas = make(map[string]ItemSchema)
	for _, typeSchema := range schema.ItemTypes {
		schemas[typeSchema.Type] = typeSchema
	}
}

//
// Reference Note
//

type ZoteroReference struct {
	// The Zotero schema is defined in the following repository:
	// https://github.com/zotero/zotero-schema/blob/master/schema.json (11,000 lines).
	fields map[string]interface{}
}

func (z ZoteroReference) String() string {
	return fmt.Sprintf("%s, by %s", z.ShortTitle(), strings.Join(z.Authors(), ", "))
}

func (z *ZoteroReference) Attributes() []core.Attribute {
	var attributes []core.Attribute

	attributes = append(attributes, core.Attribute{
		Key:   "creators",
		Value: z.fields["creators"],
	})

	itemType, _ := z.fields["itemType"].(string)
	for _, field := range schemas[itemType].Fields {
		fmt.Println(field) // FIXME remove
		// ignore some fields
		if _, ok := ignoredFields[field.Field]; ok {
			continue
		}
		if value, ok := z.fields[field.Field]; ok {
			// Ignore null fields
			if value != nil {
				attributes = append(attributes, core.Attribute{
					Key:   field.Field,
					Value: value,
				})
			}
		}
	}

	return attributes
}

func (z *ZoteroReference) GetAttributeValue(key string) interface{} {
	for _, attribute := range z.Attributes() {
		if attribute.Key == key {
			return attribute.Value
		}
	}
	return nil
}

func (r *ZoteroReference) ShortTitle() string {
	if val, ok := r.fields["shortTitle"]; ok {
		return val.(string)
	}
	return r.Title()
}

func (r *ZoteroReference) Title() string {
	return r.fields["title"].(string)
}

func (r *ZoteroReference) schema() ItemSchema {
	itemType, _ := r.fields["itemType"].(string)
	return schemas[itemType]
}

func (r *ZoteroReference) primaryCreatorType() string {
	schema := r.schema()
	for _, creatorType := range schema.CreatorTypes {
		if creatorType.Primary {
			return creatorType.Type
		}
	}
	return ""
}

func (r *ZoteroReference) Authors() []string {
	var result []string
	for _, creatorRaw := range r.fields["creators"].([]interface{}) {
		creator := creatorRaw.(map[string]interface{})
		if creator["creatorType"] == r.primaryCreatorType() {
			result = append(result, fmt.Sprintf("%s %s", creator["firstName"].(string), creator["lastName"].(string)))
		}
	}
	return result
}

func (r *ZoteroReference) PublicationYear() string {
	return r.fields["date"].(string)
}

func (r *ZoteroReference) Bibliography() string {
	// TODO implement
	return r.Title() + " by " + strings.Join(r.Authors(), " ")
}

//
// Reference Manager
//

// Zotero provides reference management using Zotero Translation Server.
// See https://github.com/zotero/translation-server
type Zotero struct {
	// Override in tests to use a mock server
	BaseURL string
	// Override in tests to capture in/out
	Stdio *terminal.Stdio
}

func NewReferenceManager() *Zotero {
	return &Zotero{
		// Zotero Translation Server uses the port 1969 by default
		BaseURL: "http://localhost:1969",
	}
}

func (z *Zotero) Search(query string) (core.Reference, error) {
	if !IsCmdDefined("docker") {
		return nil, fmt.Errorf("'docker' command is required to execute Zotero Translation Server")
	}

	// Ensure Zotero Translation Server is running
	alreadyStarted := false
	var pid int

	if z.IsPortOpen() {
		fmt.Println("Zotero Translation Server is already started")
		alreadyStarted = true
	} else {
		// Start Zotero
		z.startServerLocally()
	}

	// Query Zotero
	// Ex: curl -XPOST http://localhost:1969/search -H 'Content-Type: text/plain' -d '0525538836' | jq .
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/search", z.BaseURL), strings.NewReader(query))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP response code: %d", res.StatusCode)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}

	var filteredResults []*ZoteroReference
	for _, result := range results {
		itemTypeRaw := result["itemType"].(string)
		if !isSupportedType(ReferenceType(itemTypeRaw)) {
			// Ignore esoteric sources
			continue
		}
		zoteroReference := &ZoteroReference{
			fields: result,
		}
		filteredResults = append(filteredResults, zoteroReference)
	}

	var options []string
	referencesByOption := make(map[string]core.Reference)
	for _, reference := range filteredResults {
		options = append(options, reference.String())
		referencesByOption[reference.String()] = reference
	}

	var answer string
	prompt := &survey.Select{
		Message: "Which reference?",
		Options: options,
	}
	var surveyOpts []survey.AskOpt
	surveyOpts = append(surveyOpts, survey.WithValidator(survey.Required))
	if z.Stdio != nil {
		surveyOpts = append(surveyOpts, sopts.WithStdio(*z.Stdio))
	}
	survey.AskOne(prompt, &answer, surveyOpts...)
	answerReference := referencesByOption[answer]

	// Stop Zotero only if started by the CLI 0787960756
	if !alreadyStarted {
		tryKillProcess(pid)
	}

	return answerReference, nil
}

func (z *Zotero) startServerLocally() error {
	fmt.Println("Starting Zotero Translation Server...")
	// Ex: "docker run -d -p 1969:1969 --rm zotero/translation-server"
	cmd := exec.Command("docker", "run", "-d", "-p", "1969:1969", "--rm", "zotero/translation-server")
	err := cmd.Start()
	if err != nil {
		return err
	}
	_ = cmd.Process.Pid

	// Wait for port to be open
	// TODO refactor to avoid passive wait
	startedAt := time.Now()
	for {
		if z.IsPortOpen() {
			// Just to be sure the service is ready
			time.Sleep(100 * time.Millisecond)
			break
		}
		if time.Since(startedAt).Seconds() > 5 {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

func IsCmdDefined(cmdName string) bool {
	_, err := exec.LookPath(cmdName)
	return err == nil
}

func (z *Zotero) IsPortOpen() bool {
	u, err := url.Parse(z.BaseURL)
	if err != nil {
		return false
	}

	target := fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	conn, err := net.DialTimeout("tcp", target, 1*time.Second)
	if err != nil {
		return false
	}

	if conn != nil {
		conn.Close()
		return true
	}

	return false
}

func tryKillProcess(pid int) {
	p := os.Process{Pid: pid}
	err := p.Kill()
	if err != nil {
		fmt.Printf("Unable to kill background process: %v", err.Error())
	}
	// Ignore if the command cannot be stopped
}
