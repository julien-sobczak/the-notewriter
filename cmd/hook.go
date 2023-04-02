package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runHookCmd)
}

var runHookCmd = &cobra.Command{
	Use:   "run-hook",
	Short: "Run hooks",
	Long:  `Run all hooks on a single note.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Println("Too many arguments. You can only have one which must be a wikilink")
			os.Exit(1)
		}

		wikilink := args[0]
		notes, err := core.CurrentCollection().FindNotesByWikilink(wikilink)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if len(notes) == 0 {
			fmt.Fprintf(os.Stderr, "No note found with wikilink %q", wikilink)
			os.Exit(1)
		}
		if len(notes) > 1 {
			fmt.Fprintf(os.Stderr, "Multiple notes found with same wikilink %q", wikilink)
			os.Exit(1)
		}
		note := notes[0]
		err = note.RunHooks()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while executing hook(s): %v", err)
			os.Exit(1)
		}
	},
}
