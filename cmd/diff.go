package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/spf13/cobra"
)

var cached bool
var staged bool

func init() {
	diffCmd.Flags().BoolVarP(&cached, "cached", "", false, "Show staged changes")
	diffCmd.Flags().BoolVarP(&staged, "staged", "", false, "Show staged changes")
	rootCmd.AddCommand(diffCmd)
}

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show changes",
	Long:  `Show changes between commit and working tree.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			fmt.Println("Too many arguments. No argument is supported.")
			os.Exit(1)
		}

		stagedOrCached := staged || cached
		diff, err := core.CurrentCollection().Diff(stagedOrCached)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(diff)
	},
}
