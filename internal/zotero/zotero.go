package zotero

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "embed"

	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/manifoldco/promptui"
)

var primaryCreatorsByType map[string]string

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
	// See https://github.com/zotero/zotero-schema/blob/master/schema.json for reference
	primaryCreatorsByType = map[string]string{
		"artwork":          "artist",
		"audioRecording":   "performer",
		"blogPost":         "author",
		"book":             "author",
		"bookSection":      "author",
		"conferencePaper":  "author",
		"document":         "author",
		"film":             "director",
		"journalArticle":   "author",
		"magazineArticle":  "author",
		"letter":           "author",
		"newspaperArticle": "author",
		"podcast":          "podcaster",
		"thesis":           "author",
		"webpage":          "author",
	}

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

// Zotero provides reference management using Zotero Translation Server.
// See https://github.com/zotero/translation-server
type Zotero struct {
}

type ZoteroReference struct {
	// The Zotero schema is defined in the following repository:
	// https://github.com/zotero/zotero-schema/blob/master/schema.json (11,000 lines).
	fields map[string]interface{}
}

func (z ZoteroReference) String() string {
	return fmt.Sprintf("%s, by %s", z.ShortTitle(), strings.Join(z.Authors(), ", "))
}

func (r *ZoteroReference) Type() core.ReferenceType {
	return core.ReferenceType(r.fields["itemType"].(string))
}

func (r *ZoteroReference) Title() string {
	return r.fields["title"].(string)
}

func (r *ZoteroReference) ShortTitle() string {
	if val, ok := r.fields["shortTitle"]; ok {
		return val.(string)
	}
	return r.fields["title"].(string)
}

func (r *ZoteroReference) Authors() []string {
	var result []string
	for _, creatorRaw := range r.fields["creators"].([]interface{}) {
		creator := creatorRaw.(map[string]interface{})
		if creator["creatorType"] == primaryCreatorsByType[string(r.Type())] {
			result = append(result, fmt.Sprintf("%s %s", creator["firstName"].(string), creator["lastName"].(string)))
		}
	}
	return result
}

func (r *ZoteroReference) PublicationYear() string {
	return r.fields["date"].(string)
}

func (r *ZoteroReference) Attributes() map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range r.fields {
		result[key] = value
	}
	return result
}

func (r *ZoteroReference) AttributesOrder() []string {
	var result []string
	result = append(result, "creators")
	itemType, _ := r.fields["itemType"].(string)
	for _, field := range schemas[itemType].Fields {
		if _, ok := ignoredFields[field.Field]; ok {
			continue
		}
		result = append(result, field.Field)
	}
	return result
}

func (r *ZoteroReference) Bibliography() string {
	// TODO implement
	return r.Title() + " by " + strings.Join(r.Authors(), " ")
}

func NewReferenceManager() *Zotero {
	return &Zotero{}
}

func (z *Zotero) Search(query string) (core.Reference, error) {
	if !IsCmdDefined("docker") {
		return nil, fmt.Errorf("'docker' command is required to execute Zotero Translation Server")
	}

	// Ensure Zotero Translation Server is running
	alreadyStarted := false
	var pid int
	// Zotero Translation Server uses the port 1969 by default
	if IsPortOpen("localhost", 1969) {
		fmt.Println("Zotero Translation Server is already started")
		alreadyStarted = true
	} else {
		// Start Zotero
		fmt.Println("Starting Zotero Translation Server...")
		// Ex: "docker run -d -p 1969:1969 --rm zotero/translation-server"
		cmd := exec.Command("docker", "run", "-d", "-p", "1969:1969", "--rm", "zotero/translation-server")
		err := cmd.Start()
		if err != nil {
			return nil, err
		}
		pid = cmd.Process.Pid

		// Wait for port to be open
		// TODO refactor to avoid passive wait
		startedAt := time.Now()
		for {
			if IsPortOpen("localhost", 1969) {
				// Just to be sure the service is ready
				time.Sleep(100 * time.Millisecond)
				break
			}
			if time.Since(startedAt).Seconds() > 5 {
				break
			}
			time.Sleep(1 * time.Second)
		}
	}

	// Query Zotero
	// Ex: curl -XPOST http://localhost:1969/search -H 'Content-Type: text/plain' -d '0525538836' | jq .
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:1969/search", strings.NewReader(query))
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

	var filteredResults []core.Reference
	for _, result := range results {
		typeRaw := result["itemType"].(string)
		_, ok := primaryCreatorsByType[typeRaw]
		if !ok {
			// Ignore esoteric sources
			continue
		}
		if ok {
			zoteroReference := &ZoteroReference{
				fields: result,
			}
			filteredResults = append(filteredResults, zoteroReference)
		}
	}

	prompt := promptui.Select{
		Label: "Select",
		Items: filteredResults,
	}

	indexResult, _, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed: %v\n", err)
		os.Exit(1)
	}

	// Stop Zotero only if started by the CLI 0787960756
	if !alreadyStarted {
		tryKillProcess(pid)
	}

	return filteredResults[indexResult], nil
}

func IsCmdDefined(cmdName string) bool {
	_, err := exec.LookPath(cmdName)
	return err == nil
}

func IsPortOpen(host string, port int) bool {
	target := fmt.Sprintf("%s:%d", host, port)
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
