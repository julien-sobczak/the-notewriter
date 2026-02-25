package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)


var skipHooks bool

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().BoolVar(&skipHooks, "skip-hooks", false, "Skip running hooks")
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit",
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		err := core.CurrentRepository().Commit(skipHooks)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
