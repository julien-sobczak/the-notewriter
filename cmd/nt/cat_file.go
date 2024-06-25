package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/spf13/cobra"
)

var outputFormat string

func init() {
	catFileCmd.Flags().StringVarP(&outputFormat, "format", "o", "yaml", "format of output. Allowed: json, md, yaml")
	rootCmd.AddCommand(catFileCmd)
}

var catFileCmd = &cobra.Command{
	Use:   "cat-file",
	Short: "Display a repository file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Println("Too many arguments. You can only have one which must be a wikilink or an OID")
			os.Exit(1)
		}

		arg := args[0]

		if isOID(arg) {
			oid := arg

			// OIDs can represent a pack file, an object inside a pack file, or a blob.

			blob, err := core.CurrentRepository().FindBlobFromOID(oid)
			if err == nil && blob != nil {
				dumpObject(blob)
				return
			}

			commit, ok := core.CurrentDB().ReadCommit(oid)
			if ok {
				dumpObject(commit)
				return
			}

			object, err := core.CurrentDB().ReadCommittedObject(oid)
			if err == nil && object != nil {
				dumpObject(object)
				return
			}

			fmt.Fprintf(os.Stderr, "No object found with OID %s", oid)
			os.Exit(1)
		}

		// Try wikilinks now
		wikilinkText := args[0]
		wikilink, err := markdown.NewWikilink("[[" + wikilinkText + "]]")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Argument %q doesn't match an OID and isn't a valid wikilink", wikilink)
			os.Exit(1)
		}

		if wikilink.Section() != "" {
			// Search a note
			notes, err := core.CurrentRepository().FindNotesByWikilink(wikilink.Link)
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
				file, err := core.CurrentRepository().FindFileByWikilink(wikilink.Link)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				if file == nil {
					fmt.Fprintf(os.Stderr, "No file or note matching wikilink %q", wikilink)
					os.Exit(1)
				}
				notes, err = core.CurrentRepository().FindNotesByFileOID(file.OID)
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
			dumpObject(note)

		} else {
			// Search for a single matching file
			files, err := core.CurrentRepository().FindFilesByWikilink(wikilink.Link)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if len(files) == 0 {
				fmt.Fprintf(os.Stderr, "No file matching wikilink %q", wikilink)
				os.Exit(1)
			}
			if len(files) > 1 {
				fmt.Fprintf(os.Stderr, "Multiple files matching wikilink %q", wikilink)
				os.Exit(1)
			}
			file := files[0]
			dumpObject(file)
		}

	},
}

func dumpObject(object core.Dumpable) {
	switch outputFormat {
	case "yaml":
		fmt.Println(object.ToYAML())
	case "json":
		fmt.Println(object.ToJSON())
	case "md":
		fallthrough
	case "markdown":
		fmt.Println(object.ToMarkdown())
	default:
		fmt.Fprintf(os.Stderr, "Unsupported output format %q", outputFormat)
		os.Exit(1)
	}
}

// isOID checks if the value looks like an OID.
func isOID(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
