package anki

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Collection represents the extracted Anki collection data
type Collection struct {
	TempDir   string
	DB        *sql.DB
	Notes     []*Note
	Cards     []*Card
	Reviews   []*Review
	Models    []*Model
	Media     map[string]string // media ID -> filename
	CreatedAt time.Time         // Unix timestamp from col.crt
}

// Note represents a note from the Anki database
type Note struct {
	ID     int64
	GUID   string
	Mod    int64
	MID    int64  // Model ID
	Tags   string // Space-separated tags
	Fields []string
}

// Card represents a card from the Anki database
type Card struct {
	ID   int64
	NID  int64 // Note ID
	Mod  int64
	Ord  int // Template ordinal
	Type int
	Due  int
	Ivl  int
	Reps int
}

// Model represents a note type (model) from the Anki collection
type Model struct {
	ID        int64
	Name      string
	Fields    []Field
	Templates []Template
}

// Field represents a field in an Anki note type
type Field struct {
	Name string `json:"name"`
	Ord  int    `json:"ord"`
}

// Template represents a card template in an Anki note type
type Template struct {
	Name string `json:"name"`
	Ord  int    `json:"ord"`
	Qfmt string `json:"qfmt"` // Question format
	Afmt string `json:"afmt"` // Answer format
}

// Review represents a review from the revlog table
type Review struct {
	ID      int64
	CID     int64 // Card ID
	Ease    int   // Ease factor
	Ivl     int   // Interval (days) after this review
	LastIvl int   // Last interval (for learning cards)
	Factor  int   // Ease factor (percentage ×10, e.g. 2500 = 250%)
	Time    int   // Time spent answering (ms)
	Type    int   // 0=learn, 1=review, 2=relearn
}

// ExtractCollection extracts and parses an Anki .apkg file
func ExtractCollection(apkgPath string) (*Collection, error) {
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
	dbPath := filepath.Join(tempDir, "collection.anki21")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	collection := &Collection{
		TempDir: tempDir,
		DB:      db,
	}

	// Load crt (created at) from col table
	var crt int64
	err = db.QueryRow("SELECT crt FROM col").Scan(&crt)
	if err != nil {
		collection.Close()
		return nil, fmt.Errorf("failed to read collection creation time: %w", err)
	}
	collection.CreatedAt = time.Unix(crt, 0)

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
func (c *Collection) Close() error {
	if c.DB != nil {
		c.DB.Close()
	}
	if c.TempDir != "" {
		os.RemoveAll(c.TempDir)
	}
	return nil
}

// loadMedia loads the media mapping from the media file
func (c *Collection) loadMedia() error {
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
func (c *Collection) loadModels() error {
	var modelsJSON string
	err := c.DB.QueryRow("SELECT models FROM col").Scan(&modelsJSON)
	if err != nil {
		return err
	}

	var modelsMap map[string]any
	if err := json.Unmarshal([]byte(modelsJSON), &modelsMap); err != nil {
		return fmt.Errorf("failed to parse models JSON: %w", err)
	}

	for idStr, modelData := range modelsMap {
		var id int64
		fmt.Sscanf(idStr, "%d", &id)

		modelMap := modelData.(map[string]any)
		model := &Model{
			ID:   id,
			Name: modelMap["name"].(string),
		}

		// Parse fields
		if flds, ok := modelMap["flds"].([]any); ok {
			for _, f := range flds {
				fMap := f.(map[string]any)
				model.Fields = append(model.Fields, Field{
					Name: fMap["name"].(string),
					Ord:  int(fMap["ord"].(float64)),
				})
			}
		}

		// Parse templates
		if tmpls, ok := modelMap["tmpls"].([]any); ok {
			for _, t := range tmpls {
				tMap := t.(map[string]any)
				model.Templates = append(model.Templates, Template{
					Name: tMap["name"].(string),
					Ord:  int(tMap["ord"].(float64)),
					Qfmt: tMap["qfmt"].(string),
					Afmt: tMap["afmt"].(string),
				})
			}
		}

		c.Models = append(c.Models, model)
	}

	// Sort c.Models (SQLite seems to return them in random order)
	sort.Slice(c.Models, func(i, j int) bool {
		return c.Models[i].Name < c.Models[j].Name
	})

	return nil
}

// loadNotes loads all notes from the database
func (c *Collection) loadNotes() error {
	rows, err := c.DB.Query("SELECT id, guid, mod, mid, tags, flds FROM notes ORDER BY id ASC")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		note := &Note{}
		var fldsStr string
		if err := rows.Scan(&note.ID, &note.GUID, &note.Mod, &note.MID, &note.Tags, &fldsStr); err != nil {
			return err
		}

		// Split fields by ASCII unit separator (0x1f)
		note.Fields = strings.Split(fldsStr, "\x1f")

		c.Notes = append(c.Notes, note)
	}

	return rows.Err()
}

// loadCards loads all cards from the database
func (c *Collection) loadCards() error {
	rows, err := c.DB.Query("SELECT id, nid, mod, ord, type, due, ivl, reps FROM cards ORDER BY id ASC")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		card := &Card{}
		if err := rows.Scan(&card.ID, &card.NID, &card.Mod, &card.Ord, &card.Type, &card.Due, &card.Ivl, &card.Reps); err != nil {
			return err
		}
		c.Cards = append(c.Cards, card)
	}

	return rows.Err()
}

// loadReviews loads all reviews from the revlog table
func (c *Collection) loadReviews() error {
	rows, err := c.DB.Query("SELECT id, cid, ease, ivl, lastIvl, factor, time, type FROM revlog ORDER BY id ASC")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		review := &Review{}
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

// FindModel finds a model by its ID
func (c *Collection) FindModel(id int64) (*Model, bool) {
	for _, model := range c.Models {
		if model.ID == id {
			return model, true
		}
	}
	return nil, false
}
