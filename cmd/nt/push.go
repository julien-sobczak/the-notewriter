package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	pushCmd.Flags().BoolVarP(&force, "f", "", false, "force push")
	pushCmd.Flags().BoolVarP(&interactive, "i", "", false, "ask before pushing")
	rootCmd.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push to remote",
	Long:  `Push to remote new objects.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()

		remoteName := args[0]
		if core.CurrentDB().Remote(remoteName) == nil {
			fmt.Printf("There is no remote %q currently configured.\n", remoteName)
			fmt.Println("Please specify one in .nt/config")
			os.Exit(1)
		}

		err := core.CurrentRepository().Push(remoteName, interactive, force)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
