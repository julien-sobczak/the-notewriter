package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var referenceKind string

func init() {
	newReferenceCmd.Flags().StringVarP(&referenceKind, "kind", "k", "book", "kind of reference")
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
		} else {
			err := Col.AddNewReferenceNote(args[0], referenceKind)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	},
}
