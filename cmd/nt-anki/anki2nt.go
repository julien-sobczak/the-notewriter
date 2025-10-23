package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AnkiToNoteWriter handles the conversion of Anki collections to NoteWriter format
type AnkiToNoteWriter struct {
	Collection       *AnkiCollection
	TagMappings      map[string]string
	MediaDir         string
	IgnoreScheduling bool
	
	// Track which media files have been copied where
	copiedMedia map[string][]string
}

// Convert processes the Anki collection and writes flashcards to markdown files
func (c *AnkiToNoteWriter) Convert() error {
	c.copiedMedia = make(map[string][]string)
	
	// Group cards by note
	cardsByNote := make(map[int64][]*AnkiCard)
	for _, card := range c.Collection.Cards {
		cardsByNote[card.NID] = append(cardsByNote[card.NID], card)
	}
	
	// Process each note
	for _, note := range c.Collection.Notes {
		cards := cardsByNote[note.ID]
		if len(cards) == 0 {
			continue
		}
		
		// Determine output file based on tags
		outputFile := c.determineOutputFile(note)
		
		// Convert note and cards to markdown
		markdown, mediaFiles, err := c.convertNote(note, cards)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to convert note %d: %v\n", note.ID, err)
			continue
		}
		
		// Copy media files
		if len(mediaFiles) > 0 {
			if err := c.copyMediaFiles(mediaFiles, outputFile); err != nil {
				fmt.Printf("⚠️  Warning: failed to copy media files: %v\n", err)
			}
		}
		
		// Append to output file
		if err := c.appendToFile(outputFile, markdown); err != nil {
			return fmt.Errorf("failed to write to %s: %w", outputFile, err)
		}
		
		fmt.Printf("✏️  Appended %d card(s) from note %d to %s\n", len(cards), note.ID, outputFile)
	}
	
	// Warn about duplicate media files
	for mediaFile, locations := range c.copiedMedia {
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
func (c *AnkiToNoteWriter) determineOutputFile(note *AnkiNote) string {
	tags := strings.Fields(note.Tags)
	
	// Check each tag against mappings
	for _, tag := range tags {
		if filepath, ok := c.TagMappings[tag]; ok {
			return filepath
		}
	}
	
	// Use default mapping if exists
	if filepath, ok := c.TagMappings["default"]; ok {
		return filepath
	}
	
	// Fallback
	return "unclassified.md"
}

// convertNote converts an Anki note and its cards to markdown format
func (c *AnkiToNoteWriter) convertNote(note *AnkiNote, cards []*AnkiCard) (string, []string, error) {
	model, ok := c.Collection.Models[note.Mid]
	if !ok {
		return "", nil, fmt.Errorf("model %d not found", note.Mid)
	}
	
	var sb strings.Builder
	var mediaFiles []string
	
	// If note has multiple cards, create a parent heading
	if len(cards) > 1 {
		sb.WriteString(fmt.Sprintf("\n## Untitled `@slug: anki-%d`\n\n", note.ID))
		
		for _, card := range cards {
			front, back, media, err := c.evaluateTemplate(model, note, card)
			if err != nil {
				return "", nil, err
			}
			mediaFiles = append(mediaFiles, media...)
			
			sb.WriteString(fmt.Sprintf("### Flashcard: Untitled `@cid: %d` `@slug: anki-%d`\n\n", card.ID, card.ID))
			sb.WriteString(fmt.Sprintf("**Front:**\n\n%s\n\n", front))
			sb.WriteString(fmt.Sprintf("**Back:**\n\n%s\n\n", back))
		}
	} else {
		// Single card
		card := cards[0]
		front, back, media, err := c.evaluateTemplate(model, note, card)
		if err != nil {
			return "", nil, err
		}
		mediaFiles = media
		
		sb.WriteString(fmt.Sprintf("\n## Flashcard: Untitled `@cid: %d` `@slug: anki-%d`\n\n", card.ID, card.ID))
		sb.WriteString(fmt.Sprintf("**Front:**\n\n%s\n\n", front))
		sb.WriteString(fmt.Sprintf("**Back:**\n\n%s\n\n", back))
	}
	
	return sb.String(), mediaFiles, nil
}

// evaluateTemplate evaluates the Anki template for a card
func (c *AnkiToNoteWriter) evaluateTemplate(model *AnkiModel, note *AnkiNote, card *AnkiCard) (front, back string, mediaFiles []string, err error) {
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
	front = c.evaluateTemplateString(template.Qfmt, fieldMap)
	
	// Evaluate back (answer)
	// The answer template can reference {{FrontSide}}
	fieldMap["FrontSide"] = front
	back = c.evaluateTemplateString(template.Afmt, fieldMap)
	
	// Process HTML to markdown and collect media files
	front, frontMedia := c.htmlToMarkdown(front)
	back, backMedia := c.htmlToMarkdown(back)
	
	mediaFiles = append(frontMedia, backMedia...)
	
	return front, back, mediaFiles, nil
}

// evaluateTemplateString replaces {{field}} placeholders with values
func (c *AnkiToNoteWriter) evaluateTemplateString(template string, fields map[string]string) string {
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
func (c *AnkiToNoteWriter) htmlToMarkdown(html string) (string, []string) {
	text := html
	var mediaFiles []string
	
	// Convert <b> and <strong> to **
	text = regexp.MustCompile(`<b>(.*?)</b>`).ReplaceAllString(text, `**$1**`)
	text = regexp.MustCompile(`<strong>(.*?)</strong>`).ReplaceAllString(text, `**$1**`)
	
	// Convert <i> and <em> to _
	text = regexp.MustCompile(`<i>(.*?)</i>`).ReplaceAllString(text, `_$1_`)
	text = regexp.MustCompile(`<em>(.*?)</em>`).ReplaceAllString(text, `_$1_`)
	
	// Process media references (e.g., <img src="filename.jpg">)
	text, mediaFiles = c.processMediaReferences(text)
	
	// Note: Other HTML tags are left intact as specified
	
	return text, mediaFiles
}

// processMediaReferences converts Anki media references to markdown
func (c *AnkiToNoteWriter) processMediaReferences(text string) (string, []string) {
	var mediaFiles []string
	
	// Match <img src="filename">
	imgRe := regexp.MustCompile(`<img[^>]*src=["']([^"']+)["'][^>]*>`)
	text = imgRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := imgRe.FindStringSubmatch(match)
		if len(matches) > 1 {
			filename := matches[1]
			mediaFiles = append(mediaFiles, filename)
			// Convert to markdown image
			return fmt.Sprintf("![%s](%s)", filename, c.getMediaPath(filename))
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
			return fmt.Sprintf("[🔊 %s](%s)", filename, c.getMediaPath(filename))
		}
		return match
	})
	
	return text, mediaFiles
}

// getMediaPath returns the path to use for a media file in markdown
func (c *AnkiToNoteWriter) getMediaPath(filename string) string {
	if c.MediaDir != "" {
		return fmt.Sprintf("./%s/%s", c.MediaDir, filename)
	}
	return fmt.Sprintf("./%s", filename)
}

// appendToFile appends markdown content to a file, creating it if necessary
func (c *AnkiToNoteWriter) appendToFile(filepath string, content string) error {
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
		heading := c.filenameToHeading(filepath)
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
func (c *AnkiToNoteWriter) filenameToHeading(filepath string) string {
	// Get just the filename without extension
	base := filepath
	if idx := strings.LastIndex(filepath, "/"); idx != -1 {
		base = filepath[idx+1:]
	}
	
	if strings.HasSuffix(base, ".md") {
		base = base[:len(base)-3]
	}
	
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
func (c *AnkiToNoteWriter) copyMediaFiles(mediaFiles []string, outputFile string) error {
	// Determine output directory for media
	outputDir := filepath.Dir(outputFile)
	if c.MediaDir != "" {
		outputDir = filepath.Join(outputDir, c.MediaDir)
	}
	
	// Create media directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	
	// Copy each media file
	for _, filename := range mediaFiles {
		// Find the media ID for this filename
		var mediaID string
		for id, name := range c.Collection.Media {
			if name == filename {
				mediaID = id
				break
			}
		}
		
		if mediaID == "" {
			fmt.Printf("⚠️  Warning: media file '%s' not found in collection\n", filename)
			continue
		}
		
		srcPath := filepath.Join(c.Collection.TempDir, mediaID)
		destPath := filepath.Join(outputDir, filename)
		
		// Check if source file exists
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			fmt.Printf("⚠️  Warning: media source file '%s' not found\n", srcPath)
			continue
		}
		
		// Track copied media
		c.copiedMedia[filename] = append(c.copiedMedia[filename], destPath)
		
		// Copy file if it doesn't already exist
		if _, err := os.Stat(destPath); err == nil {
			fmt.Printf("   Media file '%s' already exists at %s\n", filename, destPath)
			continue
		}
		
		if err := copyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy media file %s: %w", filename, err)
		}
		
		fmt.Printf("   📎 Copied media file '%s' to %s\n", filename, destPath)
	}
	
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	return err
}
