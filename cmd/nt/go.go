package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

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
			fmt.Fprintf(os.Stderr, "No Go link %q found", goName)
			os.Exit(1)
		}

		finalURL := link.URL

		// Check for placeholders in the URL
		placeholders := parsePlaceholders(link.URL)
		if len(placeholders) > 0 {
			fmt.Printf("URL contains placeholders: %s\n\n", link.URL)
			
			values, err := promptForPlaceholders(link.URL, placeholders)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting placeholder values: %v\n", err)
				os.Exit(1)
			}
			
			finalURL = expandURL(link.URL, values)
			fmt.Printf("\nExpanded URL: %s\n", finalURL)
		}

		fmt.Printf("Opening URL: %s\n", finalURL)
		err = browser.OpenURL(finalURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to browse to %s: %v", finalURL, err)
			os.Exit(1)
		}
	},
}
