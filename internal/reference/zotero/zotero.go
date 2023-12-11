package zotero

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "embed"

	"github.com/julien-sobczak/the-notewriter/internal/reference"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
)

const (
	// How long to wait for Zotero Translation Server to start before reporting failure
	maxStartingTime = 60 * time.Second // The container is slow but one minute must be enough

	// How many query to try before declaring the service as bugged
	maxAppempts = 3

	// ISBN for the book "The Five Dysfunctions of a Team"
	theFiveDysfunctions = "0787960756"
)

// Manager provides reference management using Zotero Translation Server.
// See https://github.com/zotero/translation-server
type Manager struct {
	// Override in tests to use a mock server
	BaseURL string

	// Zotero Translation Server is running
	ready bool
	pid   int

	// Zotero Translation Server was already running prior command execution (and must not be stopped)
	alreadyStarted bool
}

type Result struct {
	title      string
	attributes map[string]any
}

func (r *Result) Description() string {
	return r.title
}

func (r *Result) Attributes() map[string]any {
	return r.attributes
}

func NewManager() (*Manager, error) {
	// Check prerequisites
	if !IsCmdDefined("docker") {
		return nil, fmt.Errorf("'docker' command is required to execute Zotero Translation Server")
	}

	manager := &Manager{
		// Zotero Translation Server uses the port 1969 by default
		BaseURL: "http://localhost:1969",
		ready:   false,
	}
	// Start the translation server in background
	go manager.init()

	return manager, nil
}

/* Zotero Translation Server Management */

func (m *Manager) init() {
	// Ensure Zotero Translation Server is running
	if m.IsPortOpen() {
		fmt.Println("Zotero Translation Server is already started")
		m.alreadyStarted = true
	} else {
		// Start Zotero
		m.startServerLocally()
	}
	// Always ensure a basic query is working before continuing
	// to catch problems with the container as soon as possible.
	m.waitForServerReady()
	m.ready = true
}

// IsPortOpen checks if the container port is open.
func (m *Manager) IsPortOpen() bool {
	u, err := url.Parse(m.BaseURL)
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

func (m *Manager) startServerLocally() error {
	fmt.Println("Starting Zotero Translation Server...")
	// Ex: "docker run -d -p 1969:1969 --rm zotero/translation-server"
	cmd := exec.Command("docker", "run", "-d", "-p", "1969:1969", "--rm", "zotero/translation-server")
	err := cmd.Start()
	if err != nil {
		return err
	}
	m.pid = cmd.Process.Pid

	// Wait for port to be available
	startedAt := clock.Now()
	for {
		if m.IsPortOpen() {
			// Just to be sure the service is ready
			time.Sleep(100 * time.Millisecond)
			break
		}
		if time.Since(startedAt) > maxStartingTime {
			log.Fatalf("Unable to start Zotero Translation Server after %v", maxStartingTime)
		}
		time.Sleep(2 * time.Second)
	}

	return nil
}

func (m *Manager) waitForServerReady() error {
	// Try to query a famous book to be sure the server is really ready
	attempts := 0
	for {
		fmt.Println("Searching for a book...") // FIXME remove
		attempts++
		results, err := m.Search(theFiveDysfunctions)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if len(results) > 0 {
			return nil
		}
		if attempts >= maxAppempts {
			log.Fatal("Zotero Translation Server seems not operational")
		}
		time.Sleep(5 * time.Second)
	}
	// The function exits with the above fatal log in case of errors
}

/* Reference interface */

func (m *Manager) Ready() (bool, error) {
	return m.ready, nil
}

func (m *Manager) Search(query string) ([]reference.Result, error) {
	// Check if the query is valid
	if query == "" {
		return nil, errors.New("search is empty")
	}

	// Ex: curl -XPOST http://localhost:1969/search -H 'Content-Type: text/plain' -d '0525538836' | jq .
	client := &http.Client{}
	fmt.Printf("POST %s/search %s\n", m.BaseURL, query) // FIXME remove
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/search", m.BaseURL), strings.NewReader(query))
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
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
	var response []map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	var results []reference.Result
	for _, result := range response {
		zoteroReference := &Result{
			title:      result["title"].(string),
			attributes: result,
		}
		results = append(results, zoteroReference)
	}

	// Stop Zotero only if started by the CLI
	if !m.alreadyStarted {
		TryKillProcess(m.pid)
	}

	return results, nil
}

/* Helpers */

func IsCmdDefined(cmdName string) bool {
	_, err := exec.LookPath(cmdName)
	return err == nil
}

func TryKillProcess(pid int) {
	p := os.Process{Pid: pid}
	err := p.Kill()
	if err != nil {
		fmt.Printf("Unable to kill background process: %v", err.Error())
	}
	// Ignore if the command cannot be stopped
}
