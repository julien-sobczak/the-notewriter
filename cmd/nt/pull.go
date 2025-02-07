package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	pullCmd.Flags().BoolVarP(&force, "f", "", false, "force push")
	pullCmd.Flags().BoolVarP(&interactive, "i", "", false, "ask before pulling")
	rootCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull remote",
	Long:  `Pull remote to retrieve new objects and update local database.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		if core.CurrentDB().Origin() == nil {
			fmt.Println("There is no remote currently configured.")
			fmt.Println("Please specify one in .nt/config")
			os.Exit(1)
		}
		err := core.CurrentRepository().Pull(interactive, force)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
