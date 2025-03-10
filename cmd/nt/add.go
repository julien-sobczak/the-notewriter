package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add objects to staging area",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Missing required argument")
			os.Exit(1)
		}

		CheckConfig()

		err := core.CurrentRepository().Add(argsToPathSpecs(args))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
