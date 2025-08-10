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

		// TODO prompt for replacements if templatized Go link

		err = browser.OpenURL(link.URL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to browse to %s: %v", link.URL, err)
			os.Exit(1)
		}
	},
}
