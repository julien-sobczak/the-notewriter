package main

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/julien-sobczak/the-notewriter/internal/core"
)

/*
 * Place code for TUI interactions using charmbracelet/huh here.
 * This makes it easier to refactor or switch to another library someday.
 */

// PromptForPlaceholders handles the interactive input for all placeholders using huh
func PromptForPlaceholders(gotoLink *core.Goto) (map[string]string, error) {
	values := make(map[string]string)
	placeholders := gotoLink.Placeholders()
	fields := make([]huh.Field, 0, len(placeholders))

	for _, placeholder := range placeholders {
		currentURL := gotoLink.Expand(values)
		label := fmt.Sprintf("Value for %s (%s)", placeholder.Name, currentURL.Short().ToANSI())

		switch {
		case placeholder.Ellipsis:
			// Autocomplete input
			fields = append(fields, huh.NewInput().
				Title(label).
				Suggestions(placeholder.Options).
				Key(placeholder.Name),
			)
		case len(placeholder.Options) > 0:
			// Select input
			fields = append(fields, huh.NewSelect[string]().
				Title(label).
				Options(huh.NewOptions(placeholder.Options...)...).
				Key(placeholder.Name),
			)
		default:
			// Text input
			fields = append(fields, huh.NewInput().
				Title(label).
				Key(placeholder.Name),
			)
		}
	}

	form := huh.NewForm(huh.NewGroup(fields...))
	if err := form.Run(); err != nil {
		return nil, err
	}

	for _, placeholder := range placeholders {
		val := form.Get(placeholder.Name)
		if str, ok := val.(string); ok {
			values[placeholder.Name] = str
		}
	}

	return values, nil
}

// promptForHooks prompts the user to select which hooks to install
func promptForHooks(remotes []core.ConfigRemote) ([]string, string, error) {
	var selectedHooks []string
	var installPrePush bool
	var selectedRemote string

	// First, ask which hooks to install
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select which Git hooks to install:").
				Options(
					huh.NewOption("pre-commit (runs 'nt lint; nt add .' before commit)", "pre-commit"),
					huh.NewOption("post-commit (runs 'nt commit' after commit)", "post-commit"),
					huh.NewOption("pre-push (runs 'nt push <remote>' before push)", "pre-push"),
				).
				Value(&selectedHooks),
		),
	)

	if err := form.Run(); err != nil {
		return nil, "", err
	}

	// Check if pre-push was selected
	for _, hook := range selectedHooks {
		if hook == "pre-push" {
			installPrePush = true
			break
		}
	}

	// If pre-push was selected, ask for remote
	if installPrePush {
		if len(remotes) == 0 {
			return nil, "", fmt.Errorf("no remotes configured in .nt/config.jsonnet")
		}

		// Build remote options
		remoteOptions := make([]huh.Option[string], 0, len(remotes))
		for _, remote := range remotes {
			label := fmt.Sprintf("%s (%s)", remote.Name, remote.Type)
			remoteOptions = append(remoteOptions, huh.NewOption(label, remote.Name))
		}

		remoteForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select remote for pre-push hook:").
					Options(remoteOptions...).
					Value(&selectedRemote),
			),
		)

		if err := remoteForm.Run(); err != nil {
			return nil, "", err
		}
	}

	return selectedHooks, selectedRemote, nil
}
