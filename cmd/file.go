package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(fileCmd)
	fileCmd.AddCommand(getFileCmd)
}

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Manage files",
	Long:  `General subcommands to manage files.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: must specify an action like get")
	},
}

var getFileCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a file",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Println("Too many arguments. You can only have one which must be a wikilink")
			os.Exit(1)
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
