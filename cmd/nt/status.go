package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Status",
	Long:  `Show the staging area.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		output, err := core.CurrentRepository().Status(argsToPathSpecs(args))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Print(output)
	},
}
