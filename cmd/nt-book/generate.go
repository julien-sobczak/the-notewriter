package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/julien-sobczak/the-notewriter/internal/core"
)

//go:embed book-style.css
var defaultCSS string

var generateCmd = &cobra.Command{
	Use:   "generate [book-title]",
	Short: "Generate book from notes using pandoc",
	Long: `Generate ePub and PDF books from notes based on configuration.
If no book title is specified, all books in the configuration will be generated.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config := core.CurrentConfig()
		
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
	rootCmd.AddCommand(generateCmd)
}

func generateBook(config *core.Config, book *core.ConfigBook) error {
	fmt.Printf("📖 Generating book: %s\n", book.Title)
	
	// Generate book files for each format
	for _, format := range book.Format {
		if err := generateBookFormat(config, book, format); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating %s format: %v\n", format, err)
		} else {
			fmt.Printf("✓ Generated %s for '%s'\n", format, book.Title)
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
	
	// Add illustration if present
	if section.Illustration != "" {
		// Convert to absolute path from repository root
		illustrationPath := filepath.Join(config.RootDirectory, section.Illustration)
		if _, err := os.Stat(illustrationPath); err == nil {
			// Include image in markdown - use relative path for markdown
			content.WriteString(fmt.Sprintf("![%s](%s)\n\n", section.Title, section.Illustration))
		} else {
			// Log warning but continue
			core.CurrentLogger().Warnf("Illustration file not found: %s", illustrationPath)
		}
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
	// Use the repository's SearchNotes method with the raw query string
	notes, err := core.CurrentRepository().SearchNotes(section.Query)
	if err != nil {
		return "", fmt.Errorf("query execution failed: %v", err)
	}
	
	var content bytes.Buffer
	
	// Add each note's content
	for i, note := range notes {
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
		
		if noteSpec.Wikilink != "" {
			// Find note by wikilink
			note = MustFindNoteByWikilink(noteSpec.Wikilink)
		} else if noteSpec.Slug != "" {
			// Find note by slug
			note = MustFindNoteBySlug(noteSpec.Slug)
		} else {
			return "", fmt.Errorf("note specification must have either wikilink or slug")
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

func MustFindNoteByWikilink(wikilink string) *core.Note {
	// Use the repository method to find notes by wikilink
	note, err := core.CurrentRepository().FindNoteByWikilink(wikilink)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to find note by wikilink '%s': %v\n", wikilink, err)
		os.Exit(1)
	}
	if note == nil {
		fmt.Fprintf(os.Stderr, "Error: No note found for wikilink '%s'\n", wikilink)
		os.Exit(1)
	}
	return note
}

func MustFindNoteBySlug(slug string) *core.Note {
	// Use the repository method to find note by slug
	note, err := core.CurrentRepository().FindNoteBySlug(slug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to find note by slug '%s': %v\n", slug, err)
		os.Exit(1)
	}
	if note == nil {
		fmt.Fprintf(os.Stderr, "Error: No note found with slug '%s'\n", slug)
		os.Exit(1)
	}
	return note
}

