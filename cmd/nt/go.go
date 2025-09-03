package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// getPlaceholderType returns the type of input needed for a placeholder
func getPlaceholderType(p core.Placeholder) string {
	if len(p.AllowedValues) > 0 && !p.Ellipsis {
		return "select"
	} else if len(p.AllowedValues) > 0 && p.Ellipsis {
		return "autocomplete"
	}
	return "input"
}

func init() {
	rootCmd.AddCommand(goCmd)
}

var goCmd = &cobra.Command{
	Use:   "go",
	Short: "Redirect to a Go link",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		goName := args[0]

		link, err := core.CurrentRepository().FindGotoByName(goName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding Go link %q: %v", goName, err)
			os.Exit(1)
		}
		if link == nil {
			fmt.Fprintf(os.Stderr, "No Go link %q found", goName)
			os.Exit(1)
		}

		finalURL := link.URL

		// Check for placeholders in the URL
		placeholders := link.Placeholders()
		if len(placeholders) > 0 {
			values, err := promptForPlaceholders(link)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting placeholder values: %v\n", err)
				os.Exit(1)
			}

			finalURL = link.Expand(values)
		}

		err = browser.OpenURL(string(finalURL))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to browse to %s: %v", finalURL, err)
			os.Exit(1)
		}
	},
}
