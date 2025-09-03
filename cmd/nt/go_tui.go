package main

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/julien-sobczak/the-notewriter/internal/core"
)

// promptForPlaceholders handles the interactive input for all placeholders using huh
func promptForPlaceholders(gotoLink *core.Goto) (map[string]string, error) {
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
