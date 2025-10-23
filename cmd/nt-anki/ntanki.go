package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var verboseInfo bool
var verboseDebug bool
var verboseTrace bool

var parallel int

var rootCmd = &cobra.Command{
	Use:   "nt-anki",
	Short: "nt-anki is an extra tool to convert Anki flashcards to notes easily",
}

func init() {
	// Use PersistentFlags to make flags accessible to sub-commands
	rootCmd.PersistentFlags().BoolVarP(&verboseInfo, "v", "", false, "enable verbose info output")
	rootCmd.PersistentFlags().BoolVarP(&verboseDebug, "vv", "", false, "enable verbose debug output")
	rootCmd.PersistentFlags().BoolVarP(&verboseTrace, "vvv", "", false, "enable verbose trace output")
	rootCmd.PersistentFlags().IntVarP(&parallel, "parallel", "t", 0, "Number of workers to use when generating blobs")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
