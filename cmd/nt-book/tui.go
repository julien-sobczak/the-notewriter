package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"

	"github.com/julien-sobczak/the-notewriter/internal/core"
)

func runInteractiveMode() {
	config := core.CurrentConfig()
	
	if len(config.ConfigFile.Books) == 0 {
		fmt.Println("No books configured in .nt/config.jsonnet")
		os.Exit(1)
	}
	
	// Prepare book options for selection
	var bookOptions []huh.Option[string]
	for _, book := range config.ConfigFile.Books {
		bookOptions = append(bookOptions, huh.NewOption(book.Title, book.Title))
	}
	
	var selectedBook string
	var selectedFormats []string
	
	// Collect all unique formats from all books, always include markdown
	formatMap := make(map[string]bool)
	formatMap["markdown"] = true // Always include markdown
	
	for _, book := range config.ConfigFile.Books {
		for _, format := range book.Format {
			formatMap[format] = true
		}
	}
	
	var formatOptions []huh.Option[string]
	for format := range formatMap {
		switch format {
		case "epub":
			formatOptions = append(formatOptions, huh.NewOption("ePub", "epub"))
		case "pdf":
			formatOptions = append(formatOptions, huh.NewOption("PDF", "pdf"))
		case "markdown":
			formatOptions = append(formatOptions, huh.NewOption("Markdown", "markdown"))
		default:
			formatOptions = append(formatOptions, huh.NewOption(format, format))
		}
	}
	
	// Create the form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a book to generate:").
				Options(bookOptions...).
				Value(&selectedBook),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select output formats:").
				Options(formatOptions...).
				Value(&selectedFormats).
				Filterable(false),
		),
	)
	
	// Run the form
	err := form.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	
	// Find the selected book
	var book *core.ConfigBook
	for _, b := range config.ConfigFile.Books {
		if b.Title == selectedBook {
			book = b
			break
		}
	}
	
	if book == nil {
		fmt.Printf("Book '%s' not found\n", selectedBook)
		os.Exit(1)
	}
	
	// Generate each selected format with spinner
	for _, format := range selectedFormats {
		err := spinner.New().
			Title(fmt.Sprintf("Generating %s...", format)).
			Action(func() {
				generateBookFormat(config, book, format)
			}).
			Run()
		
		if err != nil {
			fmt.Printf("Failed to generate %s: %v\n", format, err)
			os.Exit(1)
		}
		
		fmt.Printf("✓ Generated %s successfully\n", format)
	}
}