package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push to remote",
	Long:  `Push to remote new objects.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		if core.CurrentDB().Origin() == nil {
			fmt.Println("There is no remote currently configured.")
			fmt.Println("Please specify one in .nt/config")
			os.Exit(1)
		}
		err := core.CurrentDB().Push()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
