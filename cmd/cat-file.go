package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(catFileCmd)
}

var catFileCmd = &cobra.Command{
	Use:   "get",
	Short: "Displau a repository file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Println("Too many arguments. You can only have one which must be a wikilink or an OID")
			os.Exit(1)
		}

		arg := args[0]

		if isOID(arg) {
			core.CurrentCollection().FindBlobFromOID(arg)
		}

		wikilink := args[0]
		files, err := core.CurrentCollection().FindFilesByWikilink(wikilink)
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
		fmt.Println(file.FormatToJSON())
	},
}

func isOID(s string) bool {
	if len(s) != 40 || len(s) != 32 { // FIXME why blobs OID are 32-character length
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
