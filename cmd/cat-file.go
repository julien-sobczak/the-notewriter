package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var outputFormat string

func init() {
	catFileCmd.Flags().StringVarP(&outputFormat, "format", "o", "yaml", "format of output. Allowed: json, md, html, or text")
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

			// OIDs can represent a commit, an object inside a commit, or a blob.

			blob, err := core.CurrentCollection().FindBlobFromOID(oid)
			if err == nil && blob != nil {
				dumpObject(blob)
				return
			}

			commit, err := core.CurrentDB().ReadCommit(oid)
			if err == nil && commit != nil {
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
		wikilink, err := core.NewWikilink("[[" + wikilinkText + "]]")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Argument %q doesn't match an OID and isn't a valid wikilink", wikilink)
			os.Exit(1)
		}

		if wikilink.Anchored() {
			// Search a note
			notes, err := core.CurrentCollection().FindNotesByWikilink(wikilink.Link)
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
				file, err := core.CurrentCollection().FindFileByWikilink(wikilink.Link)
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
			dumpNote(note)

		} else {
			// Search for a single matching file
			files, err := core.CurrentCollection().FindFilesByWikilink(wikilink.Link)
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
			dumpFile(file)
		}

	},
}

func dumpYAML(v interface{}) {
	data, err := yaml.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to write YAML: %v", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", data)
}

func dumpJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to write JSON: %v", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", data)
}

func dumpObject(v interface{}) {
	switch outputFormat {
	case "yaml":
		dumpYAML(v)
	case "json":
		dumpJSON(v)
	case "md":
		fallthrough
	case "markdown":
		fallthrough
	case "html":
		fallthrough
	case "text":
		fallthrough
	default:
		fmt.Fprintf(os.Stderr, "Unsupported output format %q", outputFormat)
		os.Exit(1)
	}
}

func dumpFile(file *core.File) {
	switch outputFormat {
	case "yaml":
		fmt.Println(file.FormatToYAML())
	case "json":
		fmt.Println(file.FormatToJSON())
	case "md":
		fallthrough
	case "markdown":
		fmt.Println(file.FormatToMarkdown())
	case "html":
		fmt.Println(file.FormatToHTML())
	case "text":
		fmt.Println(file.FormatToText())
	default:
		fmt.Fprintf(os.Stderr, "Unsupported output format %q", outputFormat)
		os.Exit(1)
	}
}

func dumpNote(note *core.Note) {
	switch outputFormat {
	case "yaml":
		fmt.Println(note.FormatToYAML())
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
