package zotero

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/internal/core"
)

var primaryCreatorsByType map[string]string

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
		alreadyStarted = true
	} else {
		// Start Zotero
		// Ex: "docker run -d -p 1969:1969 --rm zotero/translation-server"
		cmd := exec.Command("docker", "run", "-d", "-p", "1969:1969", "--rm", "zotero/translation-server")
		err := cmd.Start()
		if err != nil {
			return nil, err
		}
		pid = cmd.Process.Pid

		// Wait for port to be open
		// TODO refactor
		startedAt := time.Now()
		for {
			if IsPortOpen("localhost", 1969) {
				break
			}
			if time.Since(startedAt).Seconds() > 5 {
				break
			}
			time.Sleep(1 * time.Second)
		}
	}

	// Query Zotero
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

	// Search for first matching result (ignore some types of reference)
	var uniqueResult core.Reference
	for _, result := range results {
		typeRaw := result["itemType"].(string)
		_, ok := primaryCreatorsByType[typeRaw]
		if ok {
			uniqueResult = &ZoteroReference{
				fields: result,
			}
			break
		}
	}

	// Stop Zotero only if started by the CLI 0787960756
	if !alreadyStarted {
		tryKillProcess(pid)
	}

	return uniqueResult, nil
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
		log.Printf("Unable to kill background process: %v", err.Error())
	}
	// Ignore if the command cannot be stopped
}
