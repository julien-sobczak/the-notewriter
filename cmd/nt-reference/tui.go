package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/reference"
)

/*
 * The command nt-reference uses Bubble Tea under the hood to provide an interactive CLI.
 * The code is heavily based on examples. It's probably possible better code using richer models.
 * All BubbleTea-related code is present in this file to make easy to refactor or switch to another library someday.
 */

func ChooseCategory(categories []*core.ConfigReference) *core.ConfigReference {
	options := make([]huh.Option[string], len(categories))
	for i, cat := range categories {
		options[i] = huh.NewOption(cat.Title, cat.Title)
	}
	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select reference category").
				Options(options...).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return nil
	}
	for _, cat := range categories {
		if cat.Title == selected {
			return cat
		}
	}
	return nil
}

func AskSearchQuery() string {
	var query string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Search query (title, ISBN, etc.)").
				Value(&query),
		),
	)
	if err := form.Run(); err != nil {
		return ""
	}
	return query
}

func SelectSearchResult(results []reference.Result) reference.Result {
	options := make([]huh.Option[string], len(results))
	for i, result := range results {
		options[i] = huh.NewOption(result.Description(), fmt.Sprintf("%d", i))
	}
	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a result").
				Options(options...).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return nil
	}
	idx := -1
	fmt.Sscanf(selected, "%d", &idx)
	if idx >= 0 && idx < len(results) {
		return results[idx]
	}
	return nil
}

// WaitManagerIsReady waits for the manager to be ready, showing a spinner using huh.
func WaitManagerIsReady(manager reference.Manager) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := spinner.New().
		Context(ctx).
		Title("Starting reference manager...").
		Action(func() {
			for {
				if ready, _ := manager.Ready(); ready {
					return
				}
				time.Sleep(200 * time.Millisecond)
			}
		}).
		Run()
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Reference manager is ready.")
}

func ReviewResult(text string) {
	var ok string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Review the generated reference (press Enter to continue)").
				Value(&ok).
				Description(text),
		),
	)
	_ = form.Run()
}

func AskFilename(defaultPath string) string {
	var filename string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Filename to save reference").
				Value(&filename).
				Placeholder(defaultPath),
		),
	)
	if err := form.Run(); err != nil {
		return ""
	}
	if filename == "" {
		return defaultPath
	}
	return filename
}
