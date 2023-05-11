package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

var outputFormat string

func init() {
	getNoteCmd.Flags().StringVarP(&outputFormat, "format", "o", "json", "format of output. Allowed: json, md, html, or text")
	rootCmd.AddCommand(noteCmd)
	noteCmd.AddCommand(getNoteCmd)
}

var noteCmd = &cobra.Command{
	Use:   "note",
	Short: "Manage notes",
	Long:  `General subcommands to manage notes.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: must specify an action like get")
	},
}

var getNoteCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a note",
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
		if len(notes) > 1 {
			fmt.Fprintf(os.Stderr, "Multiple notes found with same wikilink %q", wikilink)
			os.Exit(1)
		}
		// Try to find a file containing a single note and matching the wikilink
		if len(notes) == 0 {
			file, err := core.CurrentCollection().FindFileByWikilink(wikilink)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if file == nil {
				fmt.Fprintf(os.Stderr, "No file or note matching wikilink %q", wikilink)
				os.Exit(1)
			}
			notes, err = core.CurrentCollection().FindNotesByFileOID(file.OID)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if len(notes) == 0 {
				fmt.Fprintf(os.Stderr, "No note found in file %s matching wikilink %q", file.RelativePath, wikilink)
				os.Exit(1)
			}
			if len(notes) > 1 {
				fmt.Fprintf(os.Stderr, "Multiple notes found in file %s matching wikilink %q", file.RelativePath, wikilink)
				os.Exit(1)
			}
		}
		note := notes[0]
		switch outputFormat {
		case "json":
			fmt.Println(note.FormatToJSON())
		case "md":
			fallthrough
		case "markdown":
			fmt.Println(note.FormatToMarkdown())
		case "html":
			fmt.Println(note.FormatToHTML())
		case "text":
			fmt.Println(note.FormatToText())
		default:
			fmt.Fprintf(os.Stderr, "Unsupported output format %q", outputFormat)
			os.Exit(1)
		}
	},
}
