package anki

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
)

// Importer handles the conversion of Anki collections to NoteWriter format
type Importer struct {
	Collection       *Collection
	TagMappings      map[string]string
	MediaDir         string
	IgnoreScheduling bool
	Staged           bool

	// Track which media files have been copied where
	copiedMedia map[string][]string
}

// Import processes the Anki collection and writes flashcards to markdown files
func (imp *Importer) Import() error {
	// Step 1: Convert notes and cards to markdown files
	imp.copiedMedia = make(map[string][]string)

	// Group cards by note
	cardsByNote := make(map[int64][]*Card)
	for _, card := range imp.Collection.Cards {
		cardsByNote[card.NID] = append(cardsByNote[card.NID], card)
	}

	// Group reviews by card
	reviewsByCard := make(map[int64][]*Review)
	for _, review := range imp.Collection.Reviews {
		reviewsByCard[review.CID] = append(reviewsByCard[review.CID], review)
	}

	// Track output files and notes/cards written to each
	outputNotes := make(map[string][]*Note)
	outputCards := make(map[string][]*Card)

	// Process each note
	for _, note := range imp.Collection.Notes {
		cards := cardsByNote[note.ID]
		if len(cards) == 0 {
			continue
		}

		// Determine output file based on tags
		outputFile, err := imp.determineOutputFile(note)
		if err != nil {
			return err
		}

		// Track notes/cards for this output file
		outputNotes[outputFile] = append(outputNotes[outputFile], note)
		outputCards[outputFile] = append(outputCards[outputFile], cards...)

		// Convert note and cards to markdown
		markdown, mediaFiles, err := imp.convertNote(note, cards)
		if err != nil {
			return fmt.Errorf("failed to convert note %d: %w", note.ID, err)
		}

		// Copy media files
		if err := imp.copyMediaFilesForMarkdownFile(outputFile, mediaFiles); err != nil {
			return fmt.Errorf("⚠️ Warning: failed to copy media files: %w", err)
		}

		// Append to output file
		if err := appendToFile(outputFile, markdown); err != nil {
			return fmt.Errorf("failed to write to %s: %w", outputFile, err)
		}

		fmt.Printf("✏️ Appended %d card(s) from note %d to %s\n", len(cards), note.ID, outputFile)
	}

	// Warn about duplicate media files
	for mediaFile, locations := range imp.copiedMedia {
		if len(locations) > 1 {
			fmt.Printf("⚠️ Warning: media file '%s' was copied to multiple locations: %s\n", mediaFile, strings.Join(locations, ", "))
		}
	}

	// Step 2: For each output file, create a packfile with all reviews for its notes/cards
	if imp.IgnoreScheduling {
		return nil
	}
	for _, cards := range outputCards {
		// Build all reviews for these cards
		var operations []*core.Operation

		for _, card := range cards {
			cardReviews := reviewsByCard[card.ID]
			for _, ankiReview := range cardReviews {
				// Build the review
				review := &core.FlashcardReview{
					Feedback:  core.FeedbackGood, // Could be mapped from ankiReview.Ease if needed
					Duration:  time.Duration(ankiReview.Time) * time.Millisecond,
					DueAt:     getDueAtForCard(card),
					Algorithm: "anki-sm-2",
					Settings: map[string]any{
						"ease":    ankiReview.Ease,
						"ivl":     ankiReview.Ivl,
						"lastIvl": ankiReview.LastIvl,
						"factor":  ankiReview.Factor,
						"time":    ankiReview.Time,
						"type":    ankiReview.Type,
					},
				}

				// Build the operation
				flashcardOID := oid.NewFromString("anki-" + fmt.Sprint(card.ID))
				op := core.NewOperationReviewFlashcard(flashcardOID, *review)
				op.Timestamp = time.UnixMilli(ankiReview.ID)
				operations = append(operations, op)
			}
		}

		// Create packfile
		packFile, err := core.NewPackFileFromOperations(operations)
		if err != nil {
			return fmt.Errorf("failed to create packfile: %w", err)
		}

		// If staged, add to index
		if imp.Staged {
			index := core.CurrentIndex()
			if err := index.Add(packFile); err != nil { // TODO needed? Probably for pack file to be pushed to remote
				return fmt.Errorf("failed to add packfile to index: %w", err)
			}
			if err := index.Save(); err != nil {
				return fmt.Errorf("failed to save index: %w", err)
			}
		}
	}

	return nil
}

// Helper to get DueAt from anki.Card
func getDueAtForCard(card *Card) time.Time {
	// If Due is a timestamp (e.g. > 10^9), treat as unix seconds
	if card.Due > 1000000000 {
		return time.Unix(int64(card.Due), 0)
	}
	// Otherwise treat as days from last card modification time
	return time.Unix(card.Mod, 0).Add(time.Duration(card.Due) * 24 * time.Hour)
}

// determineOutputFile determines which file to write the flashcard to based on tags
func (imp *Importer) determineOutputFile(note *Note) (string, error) {
	tags := strings.FieldsSeq(note.Tags)

	// Check each tag against mappings
	for tag := range tags {
		if filepath, ok := imp.TagMappings[tag]; ok {
			return filepath, nil
		}
	}

	// Use default mapping if exists
	if filepath, ok := imp.TagMappings["default"]; ok {
		return filepath, nil
	}

	// Return error if no mapping found
	return "", fmt.Errorf("no mapping found for note %d with tags: %s (and no default mapping configured)", note.ID, note.Tags)
}

// convertNote converts an Anki note and its cards to markdown format
func (imp *Importer) convertNote(note *Note, cards []*Card) (string, []string, error) {
	model, ok := imp.Collection.FindModel(note.MID)
	if !ok {
		return "", nil, fmt.Errorf("model %d not found", note.MID)
	}

	var sb strings.Builder
	var mediaFiles []string

	// If note has multiple cards, create a parent heading
	if len(cards) > 1 {
		sb.WriteString(fmt.Sprintf("\n## Untitled `@slug: anki-%d`\n", note.ID))

		for _, card := range cards {
			cardMarkdown, media, err := imp.convertCard(model, note, card, 3)
			if err != nil {
				return "", nil, err
			}
			sb.WriteString(cardMarkdown)
			mediaFiles = append(mediaFiles, media...)
		}
	} else {
		// Single card
		card := cards[0]
		cardMarkdown, media, err := imp.convertCard(model, note, card, 2)
		if err != nil {
			return "", nil, err
		}
		sb.WriteString(cardMarkdown)
		mediaFiles = media
	}

	return sb.String(), mediaFiles, nil
}

// convertCard converts a single card to markdown
func (imp *Importer) convertCard(model *Model, note *Note, card *Card, headingLevel int) (string, []string, error) {
	front, back, media, err := imp.evaluateTemplate(model, note, card)
	if err != nil {
		return "", nil, err
	}

	var sb strings.Builder
	headingPrefix := strings.Repeat("#", headingLevel)
	sb.WriteString(fmt.Sprintf("\n%s Flashcard: Untitled `@cid: %d` `@slug: anki-%d`", headingPrefix, card.ID, card.ID))

	// Add tags if present
	if len(note.Tags) > 0 {
		tags := strings.Fields(note.Tags)
		if len(tags) > 0 {
			sb.WriteString(" `#")
			sb.WriteString(strings.Join(tags, "` `#"))
			sb.WriteString("`")
		}
	}

	sb.WriteString("\n\n")
	sb.WriteString(front)
	sb.WriteString("\n")
	if back != "" {
		// Cloze deletions don't have a back
		sb.WriteString("\n---\n\n")
		sb.WriteString(back)
		sb.WriteString("\n")
	}

	return sb.String(), media, nil
}

func (imp *Importer) evaluateClozeTemplate(model *Model, note *Note, card *Card) (front, back string, mediaFiles []string, err error) {
	// Cloze templates are handled differently
	// Anki template uses a special directive when rendering the field "{(cloze:Text}}"

	// TheNoteWriter uses a slightly different syntax for cloze deletions
	// Convert {{c1::text}} to [{c1::text}]
	textField := note.Fields[0]
	textField = regexp.MustCompile(`\{\{c(.*?)\}\}`).ReplaceAllString(textField, `[{c$1}]`)

	// Evaluate the template by considering only the question as the back will be generated automatically by TheNoteWriter
	template := model.Templates[card.Ord]
	front = imp.evaluateTemplateString(template.Qfmt, map[string]string{
		"cloze:Text": textField,
	})

	front, frontMedia := imp.htmlToMarkdown(front)

	return front, "", frontMedia, nil
}

// evaluateTemplate evaluates the Anki template for a card
func (imp *Importer) evaluateTemplate(model *Model, note *Note, card *Card) (front, back string, mediaFiles []string, err error) {
	if card.Ord >= len(model.Templates) {
		// Guard against out-of-bounds
		return "", "", nil, fmt.Errorf("template %d not found in model", card.Ord)
	}

	if model.Name == "Cloze" {
		// Handle cloze separately
		return imp.evaluateClozeTemplate(model, note, card)
	}

	template := model.Templates[card.Ord]

	// Build field map
	fieldMap := make(map[string]string)
	for i, field := range model.Fields {
		if i < len(note.Fields) {
			fieldMap[field.Name] = note.Fields[i]
		}
	}

	// Evaluate front (question)
	front = imp.evaluateTemplateString(template.Qfmt, fieldMap)
	// Evaluate back (answer)
	// The answer template can reference {{FrontSide}}
	fieldMap["FrontSide"] = front
	back = imp.evaluateTemplateString(template.Afmt, fieldMap)

	// Anki typically prepends FrontSide to the back and use "<hr id=answer>" as separator
	// Trim the front side if present and any spaces around
	if idx := strings.Index(back, "<hr id=answer>"); idx != -1 {
		back = back[idx+len("<hr id=answer>"):]
	}

	front = strings.TrimSpace(front)
	back = strings.TrimSpace(back)

	// Process HTML to markdown and collect media files
	front, frontMedia := imp.htmlToMarkdown(front)
	back, backMedia := imp.htmlToMarkdown(back)

	mediaFiles = append(frontMedia, backMedia...)

	return front, back, mediaFiles, nil
}

// evaluateTemplateString replaces {{field}} placeholders with values
func (imp *Importer) evaluateTemplateString(template string, fields map[string]string) string {
	result := template

	// Replace {{field}} with field values
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		fieldName := strings.TrimSpace(match[2 : len(match)-2])
		if value, ok := fields[fieldName]; ok {
			return value
		}
		return match
	})

	return result
}

// htmlToMarkdown converts basic HTML to markdown and processes media references
func (imp *Importer) htmlToMarkdown(html string) (string, []string) {
	text := html
	var mediaFiles []string

	// Convert <b> and <strong> to **
	text = regexp.MustCompile(`<b>(.*?)</b>`).ReplaceAllString(text, `**$1**`)
	text = regexp.MustCompile(`<strong>(.*?)</strong>`).ReplaceAllString(text, `**$1**`)

	// Convert <i> and <em> to _
	text = regexp.MustCompile(`<i>(.*?)</i>`).ReplaceAllString(text, `_$1_`)
	text = regexp.MustCompile(`<em>(.*?)</em>`).ReplaceAllString(text, `_$1_`)

	// Process media references (e.g., <img src="filename.jpg">)
	text, mediaFiles = imp.processMediaReferences(text)

	// Note: Other HTML tags are left intact as specified

	return text, mediaFiles
}

// processMediaReferences converts Anki media references to markdown
func (imp *Importer) processMediaReferences(text string) (string, []string) {
	var mediaFiles []string

	// Match <img src="filename">
	imgRe := regexp.MustCompile(`<img[^>]*src=["']([^"']+)["'][^>]*>`)
	text = imgRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := imgRe.FindStringSubmatch(match)
		if len(matches) > 1 {
			filename := matches[1]
			mediaFiles = append(mediaFiles, filename)
			// Convert to markdown image (keep it simple as <img>)
			return fmt.Sprintf("![%s](%s)", filename, getMediaPath(imp.MediaDir, filename))
		}
		return match
	})

	// Match [sound:filename]
	soundRe := regexp.MustCompile(`\[sound:([^\]]+)\]`)
	text = soundRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := soundRe.FindStringSubmatch(match)
		if len(matches) > 1 {
			filename := matches[1]
			mediaFiles = append(mediaFiles, filename)
			// Convert to markdown link
			return fmt.Sprintf("![%s](%s)", filename, getMediaPath(imp.MediaDir, filename))
		}
		return match
	})

	return text, mediaFiles
}

// getMediaPath returns the path to use for a media file in markdown
func getMediaPath(mediaDir string, filename string) string {
	if mediaDir != "" {
		return fmt.Sprintf("./%s/%s", mediaDir, filename)
	}
	return fmt.Sprintf("./%s", filename)
}

// appendToFile appends markdown content to a file, creating it if necessary
func appendToFile(filepath string, content string) error {
	// Check if file exists
	exists := true
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		exists = false
	}

	// Create directory if needed
	dir := filepath
	if idx := strings.LastIndex(filepath, "/"); idx != -1 {
		dir = filepath[:idx]
	} else {
		dir = ""
	}

	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// If file is new, add a heading based on filename
	if !exists {
		heading := filenameToHeading(filepath)
		if _, err := fmt.Fprintf(f, "# %s\n", heading); err != nil {
			return err
		}
	}

	if _, err := f.WriteString(content); err != nil {
		return err
	}

	return nil
}

// filenameToHeading converts a filepath to a heading
func filenameToHeading(filePath string) string {
	// Get just the filename without extension
	base := filepath.Base(filePath)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	// Convert to title case
	words := strings.Split(base, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

// copyMediaFiles copies media files referenced in the note to the appropriate location
func (imp *Importer) copyMediaFilesForMarkdownFile(mdFile string, mediaFiles []string) error {
	// Copy each media file
	for _, filename := range mediaFiles {
		// Find the media ID for this filename
		var mediaID string
		for id, name := range imp.Collection.Media {
			if name == filename {
				mediaID = id
				break
			}
		}

		if mediaID == "" {
			fmt.Printf("⚠️  Warning: media file '%s' not found in collection\n", filename)
			continue
		}

		srcPath := filepath.Join(imp.Collection.TempDir, mediaID)
		destPath := filepath.Join(imp.DetermineMediaDirForMarkdownFile(mdFile), filename)

		// Check if source file exists
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			fmt.Printf("⚠️  Warning: media source file '%s' not found\n", srcPath)
			continue
		}

		// Track copied media
		imp.copiedMedia[filename] = append(imp.copiedMedia[filename], destPath)

		// Copy file if it doesn't already exist (using pkg/filesystem)
		if err := filesystem.CopyFileIfNotExists(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy media file %s: %w", filename, err)
		}
	}

	return nil
}

func (imp *Importer) DetermineMediaDirForMarkdownFile(mdFile string) string {
	outputDir := filepath.Dir(mdFile)
	if imp.MediaDir != "" {
		// Append options media dir if present
		outputDir = filepath.Join(outputDir, imp.MediaDir)
	}
	return outputDir
}

// ImportOption configures the Importer.
type ImportOption func(*Importer)

// WithTagMappings sets the tag mappings.
func WithTagMappings(mappings map[string]string) ImportOption {
	return func(imp *Importer) {
		imp.TagMappings = mappings
	}
}

// WithMediaDir sets the media directory.
func WithMediaDir(mediaDir string) ImportOption {
	return func(imp *Importer) {
		imp.MediaDir = mediaDir
	}
}

// WithIgnoreScheduling sets the ignore scheduling flag.
func WithIgnoreScheduling(ignore bool) ImportOption {
	return func(imp *Importer) {
		imp.IgnoreScheduling = ignore
	}
}

// Add WithStaged option
func WithStaged(staged bool) ImportOption {
	return func(imp *Importer) {
		imp.Staged = staged
	}
}

// Import allows importing using functional options, hiding Importer type.
func (c *Collection) Import(opts ...ImportOption) error {
	imp := &Importer{
		Collection: c,
	}
	for _, opt := range opts {
		opt(imp)
	}
	return imp.Import()
}
