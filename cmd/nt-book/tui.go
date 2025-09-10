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
				Options(
					huh.NewOption("Markdown", "markdown"),
					huh.NewOption("ePub", "epub"),
					huh.NewOption("PDF", "pdf"),
				).
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

	// Generate markdown content once
	markdownContent, err := generateMarkdownContent(config, book)
	if err != nil {
		fmt.Printf("Failed to generate markdown content: %v\n", err)
		os.Exit(1)
	}

	// Generate each selected format
	for _, format := range selectedFormats {
		var errGen error
		// Use spinner as generation might take time
		err := spinner.New().
			Title(fmt.Sprintf("Generating %s...", format)).
			Action(func() {
				errGen = generateBookFormat(config, book, markdownContent, format)
			}).
			Run()
		if err != nil {
			fmt.Printf("Failed to generate %s: %v\n", format, err)
			os.Exit(1)
		}
		if errGen != nil {
			fmt.Printf("Failed to generate %s: %v\n", format, errGen)
			os.Exit(1)
		}

		// Print the output path for the generated file
		outputPath := book.OutputPath(core.CurrentConfig(), format)
		fmt.Printf("✓ Generated %s successfully: %s\n", format, outputPath)
	}
}
