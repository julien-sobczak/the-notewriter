package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"

	"github.com/julien-sobczak/the-notewriter/internal/core"
)

// runInteractiveMode runs the interactive journal routine selection and appends it to the journal.
func runInteractiveMode() {
	config := core.CurrentConfig()

	journals := config.ConfigFile.Journals
	if len(journals) == 0 {
		fmt.Println("No journals configured in .nt/config.jsonnet")
		os.Exit(1)
	}

	// Step 1: Select journal (skip if only one)
	var journal *core.ConfigJournal
	if len(journals) == 1 {
		journal = journals[0]
	} else {
		journal = selectJournal(journals)
		if journal == nil {
			os.Exit(0)
		}
	}

	// Step 2: Select routine (always prompt)
	routine := selectRoutine(journal)
	if routine == nil {
		os.Exit(0)
	}

	// Step 3: Generate the template content
	generatedContent, err := core.GenerateRoutineContent(routine)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating routine content: %v\n", err)
		os.Exit(1)
	}

	// Step 4: Open editor for the user to fill in placeholders
	tmpFile, err := os.CreateTemp("", "nt-journal-*.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temporary file: %v\n", err)
		os.Exit(1)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(generatedContent); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to temporary file: %v\n", err)
		os.Exit(1)
	}
	tmpFile.Close()

	if err := openEditor(tmpPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening editor: %v\n", err)
		os.Exit(1)
	}

	// Step 5: Read the edited content
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading edited content: %v\n", err)
		os.Exit(1)
	}
	editedContent := string(data)

	// Step 6: Append to journal file
	absPath, err := core.AppendRoutineToJournal(journal, routine, editedContent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error appending routine to journal: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Routine %q appended to %s\n", routine.Name, absPath)
}

// selectJournal prompts the user to select a journal from the list.
func selectJournal(journals []*core.ConfigJournal) *core.ConfigJournal {
	options := make([]huh.Option[string], len(journals))
	for i, j := range journals {
		options[i] = huh.NewOption(j.Name, j.Name)
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a journal:").
				Options(options...).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return nil
	}

	for _, j := range journals {
		if j.Name == selected {
			return j
		}
	}
	return nil
}

// selectRoutine prompts the user to select a routine from the journal.
func selectRoutine(journal *core.ConfigJournal) *core.ConfigRoutine {
	if len(journal.Routines) == 0 {
		fmt.Printf("No routines configured for journal %q\n", journal.Name)
		return nil
	}

	options := make([]huh.Option[string], len(journal.Routines))
	for i, r := range journal.Routines {
		options[i] = huh.NewOption(r.Name, r.Name)
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a routine:").
				Options(options...).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return nil
	}

	for i := range journal.Routines {
		if journal.Routines[i].Name == selected {
			return &journal.Routines[i]
		}
	}
	return nil
}
