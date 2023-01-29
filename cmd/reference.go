package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/spf13/cobra"
)

var referenceKind string
var stdout bool

func init() {
	newReferenceCmd.Flags().StringVarP(&referenceKind, "kind", "k", "book", "kind of reference")
	newReferenceCmd.Flags().BoolVarP(&stdout, "stdout", "", false, "show result on stdout")
	rootCmd.AddCommand(referenceCmd)
	referenceCmd.AddCommand(newReferenceCmd)
}

var referenceCmd = &cobra.Command{
	Use:   "reference",
	Short: "Manage reference notes",
	Long:  `Specific subcommands to manage reference notes.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: must specify an action like new")
	},
}

var newReferenceCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new reference note",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Println("Too many arguments. You can only have one which is an identifier")
			os.Exit(1)
		}

		// Do not save a new file
		if stdout {
			f, err := core.CurrentCollection().CreateNewReferenceFile(args[0], referenceKind)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			frontMatter, err := f.FrontMatterString()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("---")
			fmt.Println(frontMatter)
			fmt.Println("---")
			os.Exit(0)
		}

		err := core.CurrentCollection().AddNewReferenceFile(args[0], referenceKind)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
