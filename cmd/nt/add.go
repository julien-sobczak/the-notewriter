package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	addCmd.Flags().BoolVar(&addForce, "force", false, "Force reparsing of Markdown files regardless of mtime")
	rootCmd.AddCommand(addCmd)
}

var addForce bool

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add objects to staging area",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Missing required argument")
			os.Exit(1)
		}

		CheckConfig()

		core.CurrentConfig().Force = addForce

		_, err := core.CurrentRepository().Add(argsToPathSpecs(args))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
