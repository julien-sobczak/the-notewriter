package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(restoreCmd)
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore local database",
	Long:  `Restore local database by clearing the staging area.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		err := core.CurrentDB().Restore()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
