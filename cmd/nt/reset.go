package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset local database",
	Long:  `Reset local database by clearing the staging area.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()

		var pathSpecs []core.PathSpec
		for _, arg := range args {
			pathSpecs = append(pathSpecs, core.PathSpec(arg))
		}

		err := core.CurrentRepository().Reset(pathSpecs)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
