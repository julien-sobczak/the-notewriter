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
		markdown, err := c.convertNote(note, cards)
		if err != nil {
			fmt.Printf("⚠️  Warning: failed to convert note %d: %v\n", note.ID, err)
			continue
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
func (c *AnkiToNoteWriter) convertNote(note *AnkiNote, cards []*AnkiCard) (string, error) {
	model, ok := c.Collection.Models[note.Mid]
	if !ok {
		return "", fmt.Errorf("model %d not found", note.Mid)
	}
	
	var sb strings.Builder
	
	// If note has multiple cards, create a parent heading
	if len(cards) > 1 {
		sb.WriteString(fmt.Sprintf("\n## Untitled `@slug: anki-%d`\n\n", note.ID))
		
		for _, card := range cards {
			front, back, err := c.evaluateTemplate(model, note, card)
			if err != nil {
				return "", err
			}
			
			sb.WriteString(fmt.Sprintf("### Flashcard: Untitled `@cid: %d` `@slug: anki-%d`\n\n", card.ID, card.ID))
			sb.WriteString(fmt.Sprintf("**Front:**\n\n%s\n\n", front))
			sb.WriteString(fmt.Sprintf("**Back:**\n\n%s\n\n", back))
		}
	} else {
		// Single card
		card := cards[0]
		front, back, err := c.evaluateTemplate(model, note, card)
		if err != nil {
			return "", err
		}
		
		sb.WriteString(fmt.Sprintf("\n## Flashcard: Untitled `@cid: %d` `@slug: anki-%d`\n\n", card.ID, card.ID))
		sb.WriteString(fmt.Sprintf("**Front:**\n\n%s\n\n", front))
		sb.WriteString(fmt.Sprintf("**Back:**\n\n%s\n\n", back))
	}
	
	return sb.String(), nil
}

// evaluateTemplate evaluates the Anki template for a card
func (c *AnkiToNoteWriter) evaluateTemplate(model *AnkiModel, note *AnkiNote, card *AnkiCard) (front, back string, err error) {
	if card.Ord >= len(model.Templates) {
		return "", "", fmt.Errorf("template %d not found in model", card.Ord)
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
	
	// Process HTML to markdown
	front = c.htmlToMarkdown(front)
	back = c.htmlToMarkdown(back)
	
	return front, back, nil
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
func (c *AnkiToNoteWriter) htmlToMarkdown(html string) string {
	text := html
	
	// Convert <b> and <strong> to **
	text = regexp.MustCompile(`<b>(.*?)</b>`).ReplaceAllString(text, `**$1**`)
	text = regexp.MustCompile(`<strong>(.*?)</strong>`).ReplaceAllString(text, `**$1**`)
	
	// Convert <i> and <em> to _
	text = regexp.MustCompile(`<i>(.*?)</i>`).ReplaceAllString(text, `_$1_`)
	text = regexp.MustCompile(`<em>(.*?)</em>`).ReplaceAllString(text, `_$1_`)
	
	// Process media references (e.g., <img src="filename.jpg">)
	text = c.processMediaReferences(text)
	
	// Note: Other HTML tags are left intact as specified
	
	return text
}

// processMediaReferences converts Anki media references to markdown
func (c *AnkiToNoteWriter) processMediaReferences(text string) string {
	// Match <img src="filename">
	imgRe := regexp.MustCompile(`<img[^>]*src=["']([^"']+)["'][^>]*>`)
	text = imgRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := imgRe.FindStringSubmatch(match)
		if len(matches) > 1 {
			filename := matches[1]
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
			// Convert to markdown link
			return fmt.Sprintf("[🔊 %s](%s)", filename, c.getMediaPath(filename))
		}
		return match
	})
	
	return text
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

// copyMediaFile copies a media file from the Anki collection to the output directory
func (c *AnkiToNoteWriter) copyMediaFile(mediaID, outputDir string) error {
	filename, ok := c.Collection.Media[mediaID]
	if !ok {
		return fmt.Errorf("media file %s not found in collection", mediaID)
	}
	
	srcPath := filepath.Join(c.Collection.TempDir, mediaID)
	destPath := filepath.Join(outputDir, filename)
	
	// Track copied media
	c.copiedMedia[filename] = append(c.copiedMedia[filename], destPath)
	
	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	
	// Copy file
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	
	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	
	_, err = io.Copy(dst, src)
	return err
}
