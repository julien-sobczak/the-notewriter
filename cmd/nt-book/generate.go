package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/julien-sobczak/the-notewriter/internal/core"
)

//go:embed book-style.css
var defaultCSS string

var dryRun bool

var generateCmd = &cobra.Command{
	Use:   "generate [book-title]",
	Short: "Generate book from notes using pandoc",
	Long: `Generate ePub and PDF books from notes based on configuration.
If no book title is specified, all books in the configuration will be generated.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config := core.CurrentConfig()
		
		// Check if pandoc is available
		if err := checkPandocAvailable(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		
		var booksToGenerate []*core.ConfigBook
		if len(args) == 0 {
			// Generate all books
			booksToGenerate = config.ConfigFile.Books
		} else {
			// Find the specific book
			bookTitle := args[0]
			for _, book := range config.ConfigFile.Books {
				if book.Title == bookTitle {
					booksToGenerate = append(booksToGenerate, book)
					break
				}
			}
			if len(booksToGenerate) == 0 {
				fmt.Fprintf(os.Stderr, "Error: Book '%s' not found in configuration\n", bookTitle)
				os.Exit(1)
			}
		}
		
		if len(booksToGenerate) == 0 {
			fmt.Println("No books configured. Add books to your .nt/config.jsonnet file.")
			return
		}
		
		for _, book := range booksToGenerate {
			if err := generateBook(config, book); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating book '%s': %v\n", book.Title, err)
			}
		}
	},
}

func init() {
	generateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Generate and display Markdown content without creating book files")
	rootCmd.AddCommand(generateCmd)
}

func checkPandocAvailable() error {
	_, err := exec.LookPath("pandoc")
	if err != nil {
		return fmt.Errorf("pandoc is not installed or not in PATH. Please install pandoc to generate books")
	}
	return nil
}

func generateBook(config *core.Config, book *core.ConfigBook) error {
	fmt.Printf("📖 Generating book: %s\n", book.Title)
	
	// Generate markdown content
	markdown, err := generateMarkdownContent(config, book)
	if err != nil {
		return fmt.Errorf("failed to generate markdown content: %v", err)
	}
	
	if dryRun {
		fmt.Printf("\n--- Markdown content for '%s' ---\n", book.Title)
		fmt.Print(markdown)
		fmt.Printf("\n--- End of '%s' ---\n\n", book.Title)
		return nil
	}
	
	// Create temporary markdown file
	tempDir := config.TempDir()
	tempMarkdownFile := filepath.Join(tempDir, "book_"+slugify(book.Title)+".md")
	if err := os.WriteFile(tempMarkdownFile, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("failed to write temporary markdown file: %v", err)
	}
	defer os.Remove(tempMarkdownFile)
	
	// Create temporary CSS file
	tempCSSFile := filepath.Join(tempDir, "book_"+slugify(book.Title)+".css")
	if err := os.WriteFile(tempCSSFile, []byte(defaultCSS), 0644); err != nil {
		return fmt.Errorf("failed to write temporary CSS file: %v", err)
	}
	defer os.Remove(tempCSSFile)
	
	// Generate book files for each format
	for _, format := range book.Format {
		outputPath := getOutputPath(config, book, format)
		if err := generateBookFile(tempMarkdownFile, tempCSSFile, outputPath, format, book); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating %s format: %v\n", format, err)
		} else {
			fmt.Printf("✅ Generated %s: %s\n", format, outputPath)
		}
	}
	
	return nil
}

func generateMarkdownContent(config *core.Config, book *core.ConfigBook) (string, error) {
	var content bytes.Buffer
	
	// Add book title
	content.WriteString(fmt.Sprintf("# %s\n\n", book.Title))
	
	// Add author information
	if len(book.Author) > 0 {
		content.WriteString(fmt.Sprintf("*By %s*\n\n", strings.Join(book.Author, ", ")))
	}
	
	// Add page break after title page
	content.WriteString("\\newpage\n\n")
	
	// Process chapters
	for _, chapter := range book.Chapters {
		chapterContent, err := generateSectionContent(config, chapter, 1)
		if err != nil {
			return "", fmt.Errorf("failed to generate chapter '%s': %v", chapter.Title, err)
		}
		content.WriteString(chapterContent)
	}
	
	return content.String(), nil
}

func generateSectionContent(config *core.Config, section *core.ConfigBookSection, level int) (string, error) {
	var content bytes.Buffer
	
	// Add section heading
	headingPrefix := strings.Repeat("#", level)
	content.WriteString(fmt.Sprintf("%s %s\n\n", headingPrefix, section.Title))
	
	// Add subtitle if present
	if section.Subtitle != "" {
		content.WriteString(fmt.Sprintf("*%s*\n\n", section.Subtitle))
	}
	
	// Handle different content types
	if section.Text != "" {
		// Direct text content
		content.WriteString(section.Text)
		content.WriteString("\n\n")
	} else if section.Query != "" {
		// Query-based content
		queryContent, err := generateQueryContent(config, section)
		if err != nil {
			return "", fmt.Errorf("failed to execute query '%s': %v", section.Query, err)
		}
		content.WriteString(queryContent)
	} else if len(section.Notes) > 0 {
		// Specific notes
		notesContent, err := generateNotesContent(config, section)
		if err != nil {
			return "", fmt.Errorf("failed to generate notes content: %v", err)
		}
		content.WriteString(notesContent)
	}
	
	// Handle nested sections
	if len(section.Sections) > 0 {
		for _, subsection := range section.Sections {
			subsectionContent, err := generateSectionContent(config, subsection, level+1)
			if err != nil {
				return "", fmt.Errorf("failed to generate subsection '%s': %v", subsection.Title, err)
			}
			content.WriteString(subsectionContent)
		}
	}
	
	return content.String(), nil
}

func generateQueryContent(config *core.Config, section *core.ConfigBookSection) (string, error) {
	// Parse the query
	query, err := core.ParseQuery(section.Query)
	if err != nil {
		return "", fmt.Errorf("invalid query: %v", err)
	}
	
	// For now, let's load all notes and filter them manually
	// TODO: Implement proper query execution in the future
	db := core.CurrentDB()
	notes, err := core.QueryNotes(db.Client(), "")
	if err != nil {
		return "", fmt.Errorf("query execution failed: %v", err)
	}
	
	// Filter notes based on query (simple implementation)
	var filteredNotes []*core.Note
	for _, note := range notes {
		if matchesQuery(note, query) {
			filteredNotes = append(filteredNotes, note)
		}
	}
	
	var content bytes.Buffer
	
	// Add each note's content
	for i, note := range filteredNotes {
		if section.PageBreaks && i > 0 {
			content.WriteString("\\newpage\n\n")
		}
		
		// Add note title as subheading
		content.WriteString(fmt.Sprintf("## %s\n\n", string(note.Title)))
		
		// Add note content (excluding the title part)
		noteContent := string(note.Body) // Use Body instead of Content to avoid title duplication
		content.WriteString(noteContent)
		content.WriteString("\n\n")
		
		// Add comments if requested
		if section.IncludeComments {
			if comment := string(note.Comment); comment != "" {
				content.WriteString(fmt.Sprintf("*Comment: %s*\n\n", comment))
			}
		}
	}
	
	return content.String(), nil
}

func generateNotesContent(config *core.Config, section *core.ConfigBookSection) (string, error) {
	var content bytes.Buffer
	
	for i, noteSpec := range section.Notes {
		if section.PageBreaks && i > 0 {
			content.WriteString("\\newpage\n\n")
		}
		
		var note *core.Note
		var err error
		
		if noteSpec.Wikilink != "" {
			// Find note by wikilink
			note, err = findNoteByWikilink(noteSpec.Wikilink)
		} else if noteSpec.Slug != "" {
			// Find note by slug
			note, err = findNoteBySlug(noteSpec.Slug)
		} else {
			return "", fmt.Errorf("note specification must have either wikilink or slug")
		}
		
		if err != nil {
			return "", fmt.Errorf("failed to find note: %v", err)
		}
		
		// Add note title as subheading
		content.WriteString(fmt.Sprintf("## %s\n\n", string(note.Title)))
		
		// Add note content (excluding the title part)
		noteContent := string(note.Body) // Use Body instead of Content to avoid title duplication
		content.WriteString(noteContent)
		content.WriteString("\n\n")
		
		// Add comments if requested
		if section.IncludeComments {
			if comment := string(note.Comment); comment != "" {
				content.WriteString(fmt.Sprintf("*Comment: %s*\n\n", comment))
			}
		}
	}
	
	return content.String(), nil
}

func findNoteByWikilink(wikilink string) (*core.Note, error) {
	// Parse wikilink format: "path/to/file#Note: Title" or just "path/to/file"
	parts := strings.Split(wikilink, "#")
	
	var noteTitle string
	if len(parts) > 1 {
		noteTitle = parts[1]
	}
	
	// Use the repository method to find notes by wikilink pattern
	db := core.CurrentDB()
	notes, err := core.QueryNotes(db.Client(), "WHERE wikilink LIKE ?", "%"+wikilink+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to query notes: %v", err)
	}
	
	if len(notes) == 0 {
		return nil, fmt.Errorf("no notes found for wikilink '%s'", wikilink)
	}
	
	// If no specific title, return first note
	if noteTitle == "" {
		return notes[0], nil
	}
	
	// Find note with matching title
	for _, note := range notes {
		if string(note.Title) == noteTitle {
			return note, nil
		}
	}
	
	return nil, fmt.Errorf("note with title '%s' not found in wikilink %s", noteTitle, wikilink)
}

func findNoteBySlug(slug string) (*core.Note, error) {
	// Query note by slug
	db := core.CurrentDB()
	note, err := core.QueryNote(db.Client(), "WHERE slug = ?", "", slug)
	if err != nil {
		return nil, fmt.Errorf("failed to query note by slug: %v", err)
	}
	if note == nil {
		return nil, fmt.Errorf("no note found with slug '%s'", slug)
	}
	
	return note, nil
}

func generateBookFile(markdownFile, cssFile, outputPath, format string, book *core.ConfigBook) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}
	
	// Build pandoc command
	args := []string{}
	
	// Input file
	args = append(args, markdownFile)
	
	// Output file
	args = append(args, "-o", outputPath)
	
	// Add CSS styling
	args = append(args, "--css", cssFile)
	
	// Format-specific options
	switch strings.ToLower(format) {
	case "epub":
		// Add metadata for EPUB
		args = append(args, "--metadata", fmt.Sprintf("title=%s", book.Title))
		if len(book.Author) > 0 {
			args = append(args, "--metadata", fmt.Sprintf("author=%s", strings.Join(book.Author, ", ")))
		}
		args = append(args, "--metadata", fmt.Sprintf("lang=%s", book.Language))
		
		// Add cover if specified
		if book.Cover != "" {
			coverPath, err := resolveCoverPath(book.Cover)
			if err != nil {
				fmt.Printf("Warning: Failed to resolve cover image: %v\n", err)
			} else {
				args = append(args, "--epub-cover-image", coverPath)
			}
		}
		
	case "pdf":
		// PDF-specific options
		args = append(args, "--pdf-engine=weasyprint")
		
		// Add metadata for PDF
		args = append(args, "--metadata", fmt.Sprintf("title=%s", book.Title))
		if len(book.Author) > 0 {
			args = append(args, "--metadata", fmt.Sprintf("author=%s", strings.Join(book.Author, ", ")))
		}
		
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	// Add table of contents if requested
	if book.TOC {
		args = append(args, "--toc")
	}
	
	// Execute pandoc
	cmd := exec.Command("pandoc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pandoc command failed: %v\nOutput: %s", err, string(output))
	}
	
	return nil
}

func getOutputPath(config *core.Config, book *core.ConfigBook, format string) string {
	if book.Build != "" {
		// Use configured build path with extension substitution
		path := strings.ReplaceAll(book.Build, "${extension}", format)
		if !filepath.IsAbs(path) {
			path = filepath.Join(config.RootDirectory, path)
		}
		return path
	}
	
	// Default to slug of title in root directory
	filename := fmt.Sprintf("%s.%s", slugify(book.Title), format)
	return filepath.Join(config.RootDirectory, filename)
}

func resolveCoverPath(coverPath string) (string, error) {
	// If it's an HTTP URL, return as-is (pandoc can handle URLs)
	if strings.HasPrefix(coverPath, "http://") || strings.HasPrefix(coverPath, "https://") {
		return coverPath, nil
	}
	
	// If it's a relative path, make it absolute
	if !filepath.IsAbs(coverPath) {
		config := core.CurrentConfig()
		coverPath = filepath.Join(config.RootDirectory, coverPath)
	}
	
	// Check if file exists
	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		return "", fmt.Errorf("cover image file does not exist: %s", coverPath)
	}
	
	return coverPath, nil
}

func slugify(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)
	
	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")
	
	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")
	
	return slug
}

// matchesQuery is a simple implementation to check if a note matches a query
// This is a basic implementation - in a full implementation, this would be more comprehensive
func matchesQuery(note *core.Note, query *core.Query) bool {
	// Check slug match
	if query.Slug != "" && note.Slug != query.Slug {
		return false
	}
	
	// Check type match
	if len(query.Types) > 0 {
		found := false
		for _, queryType := range query.Types {
			if note.Type == queryType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check tags match
	if len(query.Tags) > 0 {
		for _, queryTag := range query.Tags {
			if !note.Tags.Includes(queryTag) {
				return false
			}
		}
	}
	
	// Check path match
	if query.Path != "" {
		// Support both exact matches and pattern matches
		if strings.Contains(note.RelativePath, query.Path) || 
		   regexp.MustCompile(query.Path).MatchString(note.RelativePath) {
			// Match found
		} else {
			return false
		}
	}
	
	// Check search terms (simple contains check in content and title)
	if len(query.Terms) > 0 {
		contentText := strings.ToLower(string(note.Content))
		titleText := strings.ToLower(string(note.Title))
		
		for _, term := range query.Terms {
			termLower := strings.ToLower(term)
			if !strings.Contains(contentText, termLower) && !strings.Contains(titleText, termLower) {
				return false
			}
		}
	}
	
	return true
}