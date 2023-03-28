package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(lintCmd)
}

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint",
	Long:  `Check linter rules.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		result, err := core.CurrentCollection().Lint(args...)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("%d invalid files on %d analyzed files (%d errors, %d warnings)\n",
			result.AffectedFiles,
			result.AnalyzedFiles,
			len(result.Errors),
			len(result.Warnings))
		for _, violation := range result.Errors {
			fmt.Printf("[WARNING] %s (%s:%d)\n", violation.Message, violation.RelativePath, violation.Line)
		}
		for _, violation := range result.Warnings {
			fmt.Printf("[WARNING] %s (%s:%d)\n", violation.Message, violation.RelativePath, violation.Line)
		}
	},
}
