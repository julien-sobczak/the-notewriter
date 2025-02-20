package main

import (
	"fmt"
	"os"
	"strings"

	"slices"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

var lintRules string

func init() {
	lintCmd.Flags().StringVarP(&lintRules, "rules", "r", "all", "comma-separated list of rule names used to filter")
	rootCmd.AddCommand(lintCmd)
}

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Lint",
	Long:  `Check linter rules.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		rules := strings.Split(lintRules, ",")
		if slices.Contains(rules, "all") {
			// Do not filter
			rules = []string{}
		}

		result, err := core.CurrentRepository().Lint(argsToPathSpecs(args), rules)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(result)
	},
}
