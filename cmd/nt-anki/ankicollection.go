package main

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// AnkiCollection represents the extracted Anki collection data
type AnkiCollection struct {
	TempDir string
	DB      *sql.DB
	Notes   []*AnkiNote
	Cards   []*AnkiCard
	Reviews []*AnkiReview
	Models  map[int64]*AnkiModel
	Media   map[string]string // media ID -> filename
}

// AnkiNote represents a note from the Anki database
type AnkiNote struct {
	ID     int64
	GUID   string
	Mid    int64  // Model ID
	Tags   string // Space-separated tags
	Fields []string
}

// AnkiCard represents a card from the Anki database
type AnkiCard struct {
	ID   int64
	NID  int64 // Note ID
	Ord  int   // Template ordinal
	Type int
	Due  int
	Ivl  int
	Reps int
}

// AnkiModel represents a note type (model) from the Anki collection
type AnkiModel struct {
	ID        int64
	Name      string
	Fields    []AnkiField
	Templates []AnkiTemplate
}

type AnkiField struct {
	Name string `json:"name"`
	Ord  int    `json:"ord"`
}

type AnkiTemplate struct {
	Name string `json:"name"`
	Ord  int    `json:"ord"`
	Qfmt string `json:"qfmt"` // Question format
	Afmt string `json:"afmt"` // Answer format
}

// AnkiReview represents a review from the revlog table
type AnkiReview struct {
	ID      int64
	CID     int64 // Card ID
	Ease    int
	Ivl     int
	LastIvl int
	Factor  int
	Time    int // Time spent in milliseconds
	Type    int // 0=learn, 1=review, 2=relearn
}

// ExtractAnkiCollection extracts and parses an Anki .apkg file
func ExtractAnkiCollection(apkgPath string) (*AnkiCollection, error) {
	// Create a temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "anki-import-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Extract the ZIP archive
	if err := extractZip(apkgPath, tempDir); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to extract archive: %w", err)
	}

	// Open the SQLite database
	dbPath := filepath.Join(tempDir, "collection.anki2")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	collection := &AnkiCollection{
		TempDir: tempDir,
		DB:      db,
	}

	// Load media mapping
	if err := collection.loadMedia(); err != nil {
		collection.Close()
		return nil, fmt.Errorf("failed to load media: %w", err)
	}

	// Load models from col table
	if err := collection.loadModels(); err != nil {
		collection.Close()
		return nil, fmt.Errorf("failed to load models: %w", err)
	}

	// Load notes
	if err := collection.loadNotes(); err != nil {
		collection.Close()
		return nil, fmt.Errorf("failed to load notes: %w", err)
	}

	// Load cards
	if err := collection.loadCards(); err != nil {
		collection.Close()
		return nil, fmt.Errorf("failed to load cards: %w", err)
	}

	// Load reviews (revlog)
	if err := collection.loadReviews(); err != nil {
		collection.Close()
		return nil, fmt.Errorf("failed to load reviews: %w", err)
	}

	return collection, nil
}

// Close cleans up resources
func (c *AnkiCollection) Close() error {
	if c.DB != nil {
		c.DB.Close()
	}
	if c.TempDir != "" {
		os.RemoveAll(c.TempDir)
	}
	return nil
}

// loadMedia loads the media mapping from the media file
func (c *AnkiCollection) loadMedia() error {
	mediaPath := filepath.Join(c.TempDir, "media")
	data, err := os.ReadFile(mediaPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.Media = make(map[string]string)
			return nil // No media file is OK
		}
		return err
	}

	c.Media = make(map[string]string)
	if err := json.Unmarshal(data, &c.Media); err != nil {
		return fmt.Errorf("failed to parse media file: %w", err)
	}

	return nil
}

// loadModels loads note types (models) from the col table
func (c *AnkiCollection) loadModels() error {
	var modelsJSON string
	err := c.DB.QueryRow("SELECT models FROM col").Scan(&modelsJSON)
	if err != nil {
		return err
	}

	var modelsMap map[string]interface{}
	if err := json.Unmarshal([]byte(modelsJSON), &modelsMap); err != nil {
		return fmt.Errorf("failed to parse models JSON: %w", err)
	}

	c.Models = make(map[int64]*AnkiModel)
	for idStr, modelData := range modelsMap {
		var id int64
		fmt.Sscanf(idStr, "%d", &id)

		modelMap := modelData.(map[string]interface{})
		model := &AnkiModel{
			ID:   id,
			Name: modelMap["name"].(string),
		}

		// Parse fields
		if flds, ok := modelMap["flds"].([]interface{}); ok {
			for _, f := range flds {
				fMap := f.(map[string]interface{})
				model.Fields = append(model.Fields, AnkiField{
					Name: fMap["name"].(string),
					Ord:  int(fMap["ord"].(float64)),
				})
			}
		}

		// Parse templates
		if tmpls, ok := modelMap["tmpls"].([]interface{}); ok {
			for _, t := range tmpls {
				tMap := t.(map[string]interface{})
				model.Templates = append(model.Templates, AnkiTemplate{
					Name: tMap["name"].(string),
					Ord:  int(tMap["ord"].(float64)),
					Qfmt: tMap["qfmt"].(string),
					Afmt: tMap["afmt"].(string),
				})
			}
		}

		c.Models[id] = model
	}

	return nil
}

// loadNotes loads all notes from the database
func (c *AnkiCollection) loadNotes() error {
	rows, err := c.DB.Query("SELECT id, guid, mid, tags, flds FROM notes")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		note := &AnkiNote{}
		var fldsStr string
		if err := rows.Scan(&note.ID, &note.GUID, &note.Mid, &note.Tags, &fldsStr); err != nil {
			return err
		}

		// Split fields by ASCII unit separator (0x1f)
		note.Fields = strings.Split(fldsStr, "\x1f")

		c.Notes = append(c.Notes, note)
	}

	return rows.Err()
}

// loadCards loads all cards from the database
func (c *AnkiCollection) loadCards() error {
	rows, err := c.DB.Query("SELECT id, nid, ord, type, due, ivl, reps FROM cards")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		card := &AnkiCard{}
		if err := rows.Scan(&card.ID, &card.NID, &card.Ord, &card.Type, &card.Due, &card.Ivl, &card.Reps); err != nil {
			return err
		}
		c.Cards = append(c.Cards, card)
	}

	return rows.Err()
}

// loadReviews loads all reviews from the revlog table
func (c *AnkiCollection) loadReviews() error {
	rows, err := c.DB.Query("SELECT id, cid, ease, ivl, lastIvl, factor, time, type FROM revlog")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		review := &AnkiReview{}
		if err := rows.Scan(&review.ID, &review.CID, &review.Ease, &review.Ivl, &review.LastIvl, &review.Factor, &review.Time, &review.Type); err != nil {
			return err
		}
		c.Reviews = append(c.Reviews, review)
	}

	return rows.Err()
}

// extractZip extracts a ZIP file to the specified directory
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Prevent directory traversal
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
