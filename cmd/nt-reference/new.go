package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/reference"
	"github.com/julien-sobczak/the-notewriter/internal/reference/googlebooks"
	"github.com/julien-sobczak/the-notewriter/internal/reference/wikipedia"
	"github.com/julien-sobczak/the-notewriter/internal/reference/zotero"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newCmd)
}

// Run locally:
//
//	$ go run cmd/nt-reference/*.go new
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Prompt for a new reference",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			fmt.Println("No argument expected")
			os.Exit(1)
		}

		CheckConfig() // Useful to find reference categories
		configReference := core.CurrentConfig().ConfigFile.Reference

		// Step 1: Choose a category (Book, Person, etc.)
		_, selectedConfigReference := ChooseCategory(configReference)

		// Instantiate the manager
		var manager = createManager(selectedConfigReference)

		if ready, _ := manager.Ready(); !ready {
			WaitManagerIsReady(manager)
		}

		// Step 2: Search
		input := AskSearchQuery()
		if input == "" {
			os.Exit(0)
		}
		results, err := manager.Search(input)
		if err != nil {
			log.Fatal(err)
		}
		if len(results) == 0 {
			log.Fatalf("No results found for query %q", input)
		}
		var result reference.Result
		if len(results) == 1 {
			result = results[0]
		} else {
			// Ask which result to use
			result = SelectSearchResult(results)
			if result == nil {
				os.Exit(0)
			}
		}

		// Step 3: Review the generate reference text
		resultText, err := reference.EvaluateTemplate(selectedConfigReference.Template, result)
		if err != nil {
			log.Fatal(err)
		}
		ReviewResult(resultText)
		resultPath, err := reference.EvaluateTemplate(selectedConfigReference.Path, result)
		if err != nil {
			log.Fatal(err)
		}

		// Step 4: Write to save?
		filename := AskFilename(resultPath)
		if filename == "" {
			os.Exit(0)
		}
		saveTo(filename, resultText)
	},
}

func createManager(category core.ConfigReference) reference.Manager {
	switch category.Manager {
	case "zotero":
		var err error
		manager, err := zotero.NewManager()
		if err != nil {
			log.Fatal(err)
		}
		return manager
	case "wikipedia":
		return wikipedia.NewManager()
	case "google-books":
		return googlebooks.NewManager()
	}
	log.Fatalf("Unknown reference manager %q", category.Manager)
	return nil
}

func saveTo(path string, text string) {
	// Write by appending if the file already exists
	absoluteFilepath := filepath.Join(core.CurrentConfig().RootDirectory, path)
	f, err := os.OpenFile(absoluteFilepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Unable to open file %q: %v", absoluteFilepath, err)
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf("\n%s\n", text)); err != nil {
		log.Fatalf("Unable to write to file %q: %v", absoluteFilepath, err)
	}
}
