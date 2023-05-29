package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

var commitMessage string

func init() {
	commitCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "commit message")
	rootCmd.AddCommand(commitCmd)
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit",
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		err := core.CurrentDB().Commit(commitMessage)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
