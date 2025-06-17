package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	gcCmd.Flags().BoolVarP(&dryRun, "n", "", false, "no side-effects")
	rootCmd.AddCommand(gcCmd)
}

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Garbage collect",
	Long:  `Garbage collect unreferenced objects/blobs locally.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		err := core.CurrentDB().GC(dryRun)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
