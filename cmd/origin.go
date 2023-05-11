package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(originCmd)
	originCmd.AddCommand(gcOriginCmd)
}

var originCmd = &cobra.Command{
	Use:   "origin",
	Short: "Origin",
	Long:  `Manage origin.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: must specify an action like gc")
	},
}

var gcOriginCmd = &cobra.Command{
	Use:   "gc",
	Short: "Garbage Collect",
	Long:  `Garbage collect unreferenced objects/blobs remotely.`,
	Run: func(cmd *cobra.Command, args []string) {

		CheckConfig()
		if core.CurrentDB().Origin() == nil {
			fmt.Println("There is no remote currently configured.")
			fmt.Println("Please specify one in .nt/config")
			os.Exit(1)
		}
		err := core.CurrentDB().OriginGC()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
