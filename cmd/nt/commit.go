package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)


func init() {
	rootCmd.AddCommand(commitCmd)
}

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit",
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		err := core.CurrentRepository().Commit()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
