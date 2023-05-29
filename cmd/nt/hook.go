package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

var hookNames string

func init() {
	runHookCmd.Flags().StringVarP(&hookNames, "name", "n", "all", "comma-separated list of hook names used to execute")
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

		// Process argument(s)
		wikilink := args[0]

		// Process flag(s)
		var hooks []string
		if hookNames != "all" && hookNames != "" {
			hooks = append(hooks, strings.Split(hookNames, ",")...)
		}

		// Try to find a note matching this wikilink
		notes, err := core.CurrentCollection().FindNotesByWikilink(wikilink)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if len(notes) > 1 {
			fmt.Fprintf(os.Stderr, "Multiple notes found with same wikilink %q", wikilink)
			os.Exit(1)
		}
		if len(notes) == 1 {
			// Found the note, run the hook on it
			note := notes[0]
			err = note.RunHooks(hooks)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while executing hook(s): %v", err)
				os.Exit(1)
			}
			os.Exit(0)
		}

		// Try to find a file matching the wikilink
		file, err := core.CurrentCollection().FindFileByWikilink(wikilink)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if file == nil {
			fmt.Fprintf(os.Stderr, "No file or note matching wikilink %q", wikilink)
			os.Exit(1)
		}
		// Run the hook on all notes inside this file
		notes, err = core.CurrentCollection().FindNotesByFileOID(file.OID)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		for _, note := range notes {
			err = note.RunHooks(hooks)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while executing hook(s): %v", err)
				os.Exit(1)
			}
		}
	},
}
