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
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		remoteName := args[0]
		if core.CurrentDB().Remote(remoteName) == nil {
			fmt.Printf("There is no remote %q currently configured.\n", remoteName)
			fmt.Println("Please specify one in .nt/config")
			os.Exit(1)
		}
		err := core.CurrentRepository().Pull(remoteName, interactive, force)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
