package anki

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
)

const defaultEaseFactor = 2.5                   // Default ease factor for cards without the field "factor" defined
var defaultSteps = []string{"1m", "10m", "24h"} // Default steps for learning queue

// Importer handles the conversion of Anki collections to NoteWriter format
type Importer struct {
	Collection       *Collection
	OutputFile       string
	MediaDir         string
	IgnoreScheduling bool
	Staged           bool

	// Track which media files have been copied where
	copiedMedia map[string][]string
}

// Import processes the Anki collection and writes flashcards to markdown files
func (i *Importer) Import() (string, error) {
	// Step 1: Convert notes and cards to markdown files
	i.copiedMedia = make(map[string][]string)

	// Group cards by note
	cardsByNote := make(map[int64][]*Card)
	for _, card := range i.Collection.Cards {
		cardsByNote[card.NID] = append(cardsByNote[card.NID], card)
	}

	// Group reviews by card
	reviewsByCard := make(map[int64][]*Review)
	for _, review := range i.Collection.Reviews {
		reviewsByCard[review.CID] = append(reviewsByCard[review.CID], review)
	}

	// Process each note
	for _, note := range i.Collection.Notes {
		cards := cardsByNote[note.ID]
		if len(cards) == 0 {
			continue
		}

		// Convert note and cards to markdown
		markdown, mediaFiles, err := i.convertNote(note, cards)
		if err != nil {
			return "", fmt.Errorf("failed to convert note %d: %w", note.ID, err)
		}

		// Copy media files
		if err := i.copyMediaFiles(mediaFiles); err != nil {
			return "", fmt.Errorf("⚠️ Warning: failed to copy media files: %w", err)
		}

		// Append to output file
		if err := i.appendToFile(markdown); err != nil {
			return "", fmt.Errorf("failed to write to %s: %w", i.OutputFile, err)
		}

		fmt.Printf("✏️ Appended %d card(s) from note %d to %s\n", len(cards), note.ID, i.OutputFile)
	}

	// Warn about duplicate media files
	for mediaFile, locations := range i.copiedMedia {
		if len(locations) > 1 {
			fmt.Printf("⚠️ Warning: media file '%s' was copied to multiple locations: %s\n", mediaFile, strings.Join(locations, ", "))
		}
	}

	// Step 2: For each output file, create a packfile with all reviews for its notes/cards
	if i.IgnoreScheduling {
		return "", nil
	}
	var operations []*core.Operation
	for _, card := range i.Collection.Cards {
		// Build all reviews for these cards
		cardReviews := reviewsByCard[card.ID]
		cardOperations := AnkiReviewsToOperations(i.Collection.CreatedAt, card, cardReviews)
		operations = append(operations, cardOperations...)
	}

	// Create packfile
	packFile, err := core.NewPackFileFromOperations(operations)
	if err != nil {
		return "", fmt.Errorf("failed to create packfile: %w", err)
	}

	return packFile.ObjectPath(), nil
}

// AnkiReviewsToOperations converts Anki reviews for a card to TheNoteWriter operations,
// including an additional operation to continue the SRS history with the new algorithm.
func AnkiReviewsToOperations(collectionCreatedAt time.Time, card *Card, ankiReviews []*Review) []*core.Operation {
	// The list of all reviews for this card)
	var operations []*core.Operation

	var lastReview *core.FlashcardReview
	var lastReviewTimestamp time.Time
	for _, ankiReview := range ankiReviews {
		// Build the review
		review := &core.FlashcardReview{
			Algorithm:  "anki-sm-2",
			Confidence: AnkiEaseToConfidence(ankiReview.Ease), // Good = moderate-high confidence (could be mapped from ankiReview.Ease if needed)
			Duration:   time.Duration(ankiReview.Time) * time.Millisecond,
			DueAt:      AnkiDueToDueAt(card.Due, collectionCreatedAt),
			Settings: map[string]any{
				"cid":     card.ID,
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
		reviewedAt := AnkiIDToTimestamp(ankiReview.ID).UTC()
		op := core.NewOperationReviewFlashcardAt(flashcardOID, *review, reviewedAt)
		operations = append(operations, op)

		// Memorize the last review to convert to TheNoteWriter's SRS algorithm
		if lastReviewTimestamp.Before(op.Timestamp) {
			lastReviewTimestamp = op.Timestamp
			lastReview = review
		}
	}

	// Append an additional operation compatible with TheNoteWriter SRS algorithm
	if lastReview != nil {

		repetitions := len(ankiReviews) // Total number of reviews for this card

		// Map Anki's SM-2 review data to TheNoteWriter's SRS settings to continue the study history using the new algorithm
		newSettings := make(map[string]any)
		newSettings["repetitions"] = repetitions

		if lastReview.Settings["type"] == 0 { // Learning queue
			// If the last review is in the learning queue, we can't reliably convert new algorithm
			newSettings["queue"] = "learning"
			if repetitions < len(defaultSteps) {
				newSettings["step"] = repetitions - 1 // Map directly to the step based on the number of reviews, as it's likely the user is progressing through the steps
				newSettings["interval"] = defaultSteps[repetitions-1]
			} else {
				// Default to the last step as we can't reliably determine the current step
				newSettings["step"] = len(defaultSteps) - 1
				newSettings["interval"] = defaultSteps[len(defaultSteps)-1]
			}
			newSettings["easeFactor"] = defaultEaseFactor
		} else { // Reviewing Queue
			newSettings["queue"] = "reviewing"
			newSettings["interval"] = fmt.Sprintf("%dd", lastReview.Settings["ivl"].(int))
			factor, err := strconv.ParseFloat(fmt.Sprintf("%d", lastReview.Settings["factor"]), 64)
			if err != nil {
				factor = defaultEaseFactor
			}
			newSettings["easeFactor"] = factor / 1000.0
		}
		review := &core.FlashcardReview{
			Algorithm:  core.DefaultSRSAlgorithm, // Force new algorithm
			Confidence: lastReview.Confidence,
			Duration:   lastReview.Duration,
			DueAt:      lastReview.DueAt,
			// Map Anki SM-2 settings to TheNoteWriter SRS settings to continue the study history using the new algorithm
			Settings: newSettings,
		}

		// Build the operation
		flashcardOID := oid.NewFromString("anki-" + fmt.Sprint(card.ID))
		reviewedAt := lastReviewTimestamp.Add(1 * time.Second) // Add a small delay to ensure it's after the last review (Last write wins),
		op := core.NewOperationReviewFlashcardAt(flashcardOID, *review, reviewedAt)
		operations = append(operations, op)
	}

	return operations
}

// AnkiIDToTimestamp converts an Anki review ID (which is a timestamp in milliseconds) to a time.Time
func AnkiIDToTimestamp(ankiID int64) time.Time {
	if ankiID > 1000000000000 { // milliseconds?
		return time.UnixMilli(ankiID)
	}
	return time.Unix(ankiID, 0) // seconds
}

// AnkiEaseToConfidence maps Anki's ease values to a confidence score for TheNoteWriter
func AnkiEaseToConfidence(ease int) int {
	switch ease {
	case 1: // Again
		return 0
	case 2: // Hard
		return 30
	case 3: // Good
		return 60
	case 4: // Easy
		return 90
	default:
		return 0
	}
}

// AnkiDueToDueAt returns the due date for a card, using collectionTimestamp as the reference for days-based due values.
func AnkiDueToDueAt(ankiDue int, collectionTimestamp time.Time) time.Time {
	// If Due is a timestamp (e.g. > 10^9), treat as unix seconds
	if ankiDue > 1000000000 {
		return time.Unix(int64(ankiDue), 0)
	}
	// Otherwise treat as days from collection creation (crt)
	return collectionTimestamp.Add(time.Duration(ankiDue) * 24 * time.Hour)
}

// convertNote converts an Anki note and its cards to markdown format
func (i *Importer) convertNote(note *Note, cards []*Card) (string, []string, error) {
	model, ok := i.Collection.FindModel(note.MID)
	if !ok {
		return "", nil, fmt.Errorf("model %d not found", note.MID)
	}

	var sb strings.Builder
	var mediaFiles []string

	// If note has multiple cards, create a parent heading
	if len(cards) > 1 {
		sb.WriteString(fmt.Sprintf("\n## Untitled `@slug: anki-%d`\n", note.ID))

		for _, card := range cards {
			cardMarkdown, media, err := i.convertCard(model, note, card, 3)
			if err != nil {
				return "", nil, err
			}
			sb.WriteString(cardMarkdown)
			mediaFiles = append(mediaFiles, media...)
		}
	} else {
		// Single card
		card := cards[0]
		cardMarkdown, media, err := i.convertCard(model, note, card, 2)
		if err != nil {
			return "", nil, err
		}
		sb.WriteString(cardMarkdown)
		mediaFiles = media
	}

	return sb.String(), mediaFiles, nil
}

// convertCard converts a single card to markdown
func (i *Importer) convertCard(model *Model, note *Note, card *Card, headingLevel int) (string, []string, error) {
	front, back, media, err := i.evaluateTemplate(model, note, card)
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

func (i *Importer) evaluateClozeTemplate(model *Model, note *Note, card *Card) (front, back string, mediaFiles []string, err error) {
	// Cloze templates are handled differently
	// Anki template uses a special directive when rendering the field "{(cloze:Text}}"

	// TheNoteWriter uses a slightly different syntax for cloze deletions
	// Convert {{c1::text}} to [{c1::text}]
	textField := note.Fields[0]
	textField = regexp.MustCompile(`\{\{c(.*?)\}\}`).ReplaceAllString(textField, `[{c$1}]`)

	// Evaluate the template by considering only the question as the back will be generated automatically by TheNoteWriter
	template := model.Templates[card.Ord]
	front = i.evaluateTemplateString(template.Qfmt, map[string]string{
		"cloze:Text": textField,
	})

	front, frontMedia := i.htmlToMarkdown(front)

	return front, "", frontMedia, nil
}

// evaluateTemplate evaluates the Anki template for a card
func (i *Importer) evaluateTemplate(model *Model, note *Note, card *Card) (front, back string, mediaFiles []string, err error) {
	if card.Ord >= len(model.Templates) {
		// Guard against out-of-bounds
		return "", "", nil, fmt.Errorf("template %d not found in model", card.Ord)
	}

	if model.Name == "Cloze" {
		// Handle cloze separately
		return i.evaluateClozeTemplate(model, note, card)
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
	front = i.evaluateTemplateString(template.Qfmt, fieldMap)
	// Evaluate back (answer)
	// The answer template can reference {{FrontSide}}
	fieldMap["FrontSide"] = front
	back = i.evaluateTemplateString(template.Afmt, fieldMap)

	// Anki typically prepends FrontSide to the back and use "<hr id=answer>" as separator
	// Trim the front side if present and any spaces around
	if idx := strings.Index(back, "<hr id=answer>"); idx != -1 {
		back = back[idx+len("<hr id=answer>"):]
	}

	front = strings.TrimSpace(front)
	back = strings.TrimSpace(back)

	// Process HTML to markdown and collect media files
	front, frontMedia := i.htmlToMarkdown(front)
	back, backMedia := i.htmlToMarkdown(back)

	mediaFiles = append(frontMedia, backMedia...)

	return front, back, mediaFiles, nil
}

// evaluateTemplateString replaces {{field}} placeholders with values
func (i *Importer) evaluateTemplateString(template string, fields map[string]string) string {
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
func (i *Importer) htmlToMarkdown(html string) (string, []string) {
	text := html
	var mediaFiles []string

	// Convert &nbsp; to space
	text = strings.ReplaceAll(text, "&nbsp;", " ")

	// Replace code blocks
	text = regexp.MustCompile(`(?s)<pre><code>(.*?)</code></pre>`).ReplaceAllString(text, "\n```\n$1\n```\n")

	// Convert arcane newlines to real newlines
	text = regexp.MustCompile(`<div><br></div>`).ReplaceAllString(text, "\n\n")
	text = regexp.MustCompile(`<br\s*/?>`).ReplaceAllString(text, "\n\n")

	// Convert simple newlines to real newlines (Anki sometimes uses <div> for newlines without <br>)
	text = regexp.MustCompile(`<div>(.*?)</div>`).ReplaceAllString(text, "${1}\n\n")

	// Convert <b> and <strong> to **
	text = regexp.MustCompile(`<b>(.*?)</b>`).ReplaceAllString(text, `**${1}**`)
	text = regexp.MustCompile(`<strong>(.*?)</strong>`).ReplaceAllString(text, `**${1}**`)

	// Convert <i> and <em> to _
	text = regexp.MustCompile(`<i>(.*?)</i>`).ReplaceAllString(text, `_${1}_`)
	text = regexp.MustCompile(`<em>(.*?)</em>`).ReplaceAllString(text, `_${1}_`)

	// Process media references (e.g., <img src="filename.jpg">)
	text, mediaFiles = i.processMediaReferences(text)

	// Note: Other HTML tags are left intact as specified

	return strings.TrimSpace(text), mediaFiles
}

// processMediaReferences converts Anki media references to markdown
func (i *Importer) processMediaReferences(text string) (string, []string) {
	mediaFiles := []string{}

	// Match <img src="filename">
	imgRe := regexp.MustCompile(`<img[^>]*src=["']([^"']+)["'][^>]*>`)
	text = imgRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := imgRe.FindStringSubmatch(match)
		if len(matches) > 1 {
			filename := matches[1]
			mediaFiles = append(mediaFiles, filename)
			// Convert to markdown image (keep it simple as <img>)
			return fmt.Sprintf("![%s](%s)", filename, i.getMediaRelativePath(filename))
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
			return fmt.Sprintf("![%s](%s)", filename, i.getMediaRelativePath(filename))
		}
		return match
	})

	return text, mediaFiles
}

// getMediaRelativePath returns the relative path to use for a media file in markdown
func (i *Importer) getMediaRelativePath(mediaFilename string) string {
	if i.MediaDir != "" {
		return fmt.Sprintf("./%s/%s", i.MediaDir, mediaFilename)
	}
	return fmt.Sprintf("./%s", mediaFilename)
}

// appendToFile appends markdown content to output file, creating it if necessary
func (i *Importer) appendToFile(content string) error {
	// Check if file exists
	exists := true
	if _, err := os.Stat(i.OutputFile); os.IsNotExist(err) {
		exists = false

		// Create missing parent directories
		if err := os.MkdirAll(filepath.Dir(i.OutputFile), 0755); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(i.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// If file is new, add a dummy heading
	if !exists {
		if _, err := f.WriteString("# Untitled\n"); err != nil {
			return err
		}
	}

	if _, err := f.WriteString(content); err != nil {
		return err
	}

	return nil
}

// copyMediaFiles copies media files referenced in the note to the appropriate location
func (i *Importer) copyMediaFiles(mediaFiles []string) error {
	// Copy each media file
	for _, filename := range mediaFiles {
		// Find the media ID for this filename
		var mediaID string
		for id, name := range i.Collection.Media {
			if name == filename {
				mediaID = id
				break
			}
		}

		if mediaID == "" {
			fmt.Printf("⚠️  Warning: media file '%s' not found in collection\n", filename)
			continue
		}

		srcPath := filepath.Join(i.Collection.TempDir, mediaID)
		destPath := filepath.Join(filepath.Dir(i.OutputFile), i.getMediaRelativePath(filename))

		// Check if source file exists
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			fmt.Printf("⚠️  Warning: media source file '%s' not found\n", srcPath)
			continue
		}

		// Track copied media
		i.copiedMedia[filename] = append(i.copiedMedia[filename], destPath)

		// Copy file if it doesn't already exist (using pkg/filesystem)
		if err := filesystem.CopyFileIfNotExists(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy media file %s: %w", filename, err)
		}
	}

	return nil
}

// ImportOption configures the Importer.
type ImportOption func(*Importer)

// WithMediaDir sets the media directory.
func WithMediaDir(mediaDir string) ImportOption {
	return func(i *Importer) {
		i.MediaDir = mediaDir
	}
}

// WithIgnoreScheduling sets the ignore scheduling flag.
func WithIgnoreScheduling(ignore bool) ImportOption {
	return func(i *Importer) {
		i.IgnoreScheduling = ignore
	}
}

// Add WithStaged option
func WithStaged(staged bool) ImportOption {
	return func(i *Importer) {
		i.Staged = staged
	}
}

// Import allows importing using functional options, hiding Importer type.
func (c *Collection) Import(outputFile string, opts ...ImportOption) (string, error) {
	importer := &Importer{
		Collection: c,
		OutputFile: outputFile,
	}
	for _, opt := range opts {
		opt(importer)
	}
	return importer.Import()
}
