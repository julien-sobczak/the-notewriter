package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(gcCmd)
}

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Garbage collect",
	Long:  `Garbage collect unreferenced objects/blobs locally.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		if core.CurrentDB().Origin() == nil {
			fmt.Println("There is no remote currently configured.")
			fmt.Println("Please specify one in .nt/config")
			os.Exit(1)
		}
		err := core.CurrentDB().GC()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
