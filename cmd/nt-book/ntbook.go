package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/spf13/cobra"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

var verboseInfo bool
var verboseDebug bool
var verboseTrace bool

var rootCmd = &cobra.Command{
	Use:   "nt-book",
	Short: "nt-book generates ePub and PDF books from notes using pandoc",
	Run: func(cmd *cobra.Command, args []string) {
		// Interactive mode when run without subcommands
		runInteractiveMode()
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		CheckConfig()

		// Enable verbose output. The most verbose level wins when multiple flags are passed.
		if verboseInfo {
			core.CurrentLogger().SetVerboseLevel(core.VerboseInfo)
		}
		if verboseDebug {
			core.CurrentLogger().SetVerboseLevel(core.VerboseDebug)
		}
		if verboseTrace {
			core.CurrentLogger().SetVerboseLevel(core.VerboseTrace)
		}
	},
}

func init() {
	// Use PersistentFlags to make flags accessible to sub-commands
	rootCmd.PersistentFlags().BoolVarP(&verboseInfo, "v", "", false, "enable verbose info output")
	rootCmd.PersistentFlags().BoolVarP(&verboseDebug, "vv", "", false, "enable verbose debug output")
	rootCmd.PersistentFlags().BoolVarP(&verboseTrace, "vvv", "", false, "enable verbose trace output")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		core.CurrentRepository().Close()
	}()
}

func CheckConfig() {
	err := core.CurrentConfig().Check()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

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
	
	// Get the first book to know available formats
	availableFormats := config.ConfigFile.Books[0].Format
	var formatOptions []huh.Option[string]
	for _, format := range availableFormats {
		switch format {
		case "epub":
			formatOptions = append(formatOptions, huh.NewOption("ePub", "epub"))
		case "pdf":
			formatOptions = append(formatOptions, huh.NewOption("PDF", "pdf"))
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
				generateBookFormat(config, book, format, false) // false = not dry run
			}).
			Run()
		
		if err != nil {
			fmt.Printf("Failed to generate %s: %v\n", format, err)
			os.Exit(1)
		}
		
		fmt.Printf("✓ Generated %s successfully\n", format)
	}
}

// generateBookFormat generates a book in a specific format
func generateBookFormat(config *core.Config, book *core.ConfigBook, format string, dryRun bool) error {
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
	tempMarkdownFile := filepath.Join(tempDir, "book_"+text.Slugify(book.Title)+".md")
	if err := os.WriteFile(tempMarkdownFile, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("failed to write temporary markdown file: %v", err)
	}
	defer os.Remove(tempMarkdownFile)
	
	// Create temporary CSS file
	tempCSSFile := filepath.Join(tempDir, "book_"+text.Slugify(book.Title)+".css")
	if err := os.WriteFile(tempCSSFile, []byte(defaultCSS), 0644); err != nil {
		return fmt.Errorf("failed to write temporary CSS file: %v", err)
	}
	defer os.Remove(tempCSSFile)
	
	// Generate the book file for the specific format
	outputPath := getOutputPath(config, book, format)
	if err := generateBookFile(tempMarkdownFile, tempCSSFile, outputPath, format, book); err != nil {
		return fmt.Errorf("error generating %s format: %v", format, err)
	}
	
	return nil
}

func main() {
	Execute()
}