package anki

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
)

// Importer handles the conversion of Anki collections to NoteWriter format
type Importer struct {
	Collection       *Collection
	TagMappings      map[string]string
	MediaDir         string
	IgnoreScheduling bool

	// Track which media files have been copied where
	copiedMedia map[string][]string
}

// Import processes the Anki collection and writes flashcards to markdown files
func (imp *Importer) Import() error {
	imp.copiedMedia = make(map[string][]string)

	// Group cards by note
	cardsByNote := make(map[int64][]*Card)
	for _, card := range imp.Collection.Cards {
		cardsByNote[card.NID] = append(cardsByNote[card.NID], card)
	}

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

		// Convert note and cards to markdown
		markdown, mediaFiles, err := imp.convertNote(note, cards)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to convert note %d: %v\n", note.ID, err)
			continue
		}

		// Copy media files
		if len(mediaFiles) > 0 {
			if err := imp.copyMediaFiles(mediaFiles, outputFile); err != nil {
				fmt.Printf("⚠️  Warning: failed to copy media files: %v\n", err)
			}
		}

		// Append to output file
		if err := imp.appendToFile(outputFile, markdown); err != nil {
			return fmt.Errorf("failed to write to %s: %w", outputFile, err)
		}

		fmt.Printf("✏️  Appended %d card(s) from note %d to %s\n", len(cards), note.ID, outputFile)
	}

	// Warn about duplicate media files
	for mediaFile, locations := range imp.copiedMedia {
		if len(locations) > 1 {
			fmt.Printf("⚠️  Warning: media file '%s' was copied to multiple locations:\n", mediaFile)
			for _, loc := range locations {
				fmt.Printf("   - %s\n", loc)
			}
		}
	}

	return nil
}

// determineOutputFile determines which file to write the flashcard to based on tags
func (imp *Importer) determineOutputFile(note *Note) (string, error) {
	tags := strings.Fields(note.Tags)

	// Check each tag against mappings
	for _, tag := range tags {
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
	model, ok := imp.Collection.Models[note.MID]
	if !ok {
		return "", nil, fmt.Errorf("model %d not found", note.MID)
	}

	var sb strings.Builder
	var mediaFiles []string

	// If note has multiple cards, create a parent heading
	if len(cards) > 1 {
		sb.WriteString(fmt.Sprintf("\n## Untitled `@slug: anki-%d`\n\n", note.ID))

		for _, card := range cards {
			cardMarkdown, media := imp.convertCard(model, note, card, 3)
			sb.WriteString(cardMarkdown)
			mediaFiles = append(mediaFiles, media...)
		}
	} else {
		// Single card
		card := cards[0]
		cardMarkdown, media := imp.convertCard(model, note, card, 2)
		sb.WriteString(cardMarkdown)
		mediaFiles = media
	}

	return sb.String(), mediaFiles, nil
}

// convertCard converts a single card to markdown
func (imp *Importer) convertCard(model *Model, note *Note, card *Card, headingLevel int) (string, []string) {
	front, back, media, err := imp.evaluateTemplate(model, note, card)
	if err != nil {
		// Return error as comment in markdown
		return fmt.Sprintf("\n%s Flashcard: Error - %v\n\n", strings.Repeat("#", headingLevel), err), nil
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
	sb.WriteString(fmt.Sprintf("**Front:**\n\n%s\n\n", front))
	sb.WriteString(fmt.Sprintf("**Back:**\n\n%s\n\n", back))

	return sb.String(), media
}

// evaluateTemplate evaluates the Anki template for a card
func (imp *Importer) evaluateTemplate(model *Model, note *Note, card *Card) (front, back string, mediaFiles []string, err error) {
	if card.Ord >= len(model.Templates) {
		return "", "", nil, fmt.Errorf("template %d not found in model", card.Ord)
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
	
	// Trim FrontSide from the beginning of back if present
	// Anki typically prepends FrontSide to the back
	back = strings.TrimPrefix(back, front)
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
			return fmt.Sprintf("<img src=\"%s\">", imp.getMediaPath(filename))
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
			return fmt.Sprintf("[🔊 %s](%s)", filename, imp.getMediaPath(filename))
		}
		return match
	})

	return text, mediaFiles
}

// getMediaPath returns the path to use for a media file in markdown
func (imp *Importer) getMediaPath(filename string) string {
	if imp.MediaDir != "" {
		return fmt.Sprintf("./%s/%s", imp.MediaDir, filename)
	}
	return fmt.Sprintf("./%s", filename)
}

// appendToFile appends markdown content to a file, creating it if necessary
func (imp *Importer) appendToFile(filepath string, content string) error {
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
		heading := imp.filenameToHeading(filepath)
		if _, err := f.WriteString(fmt.Sprintf("# %s\n", heading)); err != nil {
			return err
		}
	}

	if _, err := f.WriteString(content); err != nil {
		return err
	}

	return nil
}

// filenameToHeading converts a filepath to a heading
func (imp *Importer) filenameToHeading(filePath string) string {
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
func (imp *Importer) copyMediaFiles(mediaFiles []string, outputFile string) error {
	// Determine output directory for media
	outputDir := filepath.Dir(outputFile)
	if imp.MediaDir != "" {
		outputDir = filepath.Join(outputDir, imp.MediaDir)
	}

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
		destPath := filepath.Join(outputDir, filename)

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
