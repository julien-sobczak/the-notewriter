package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
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

		var pathSpecs []core.PathSpec
		for _, arg := range args {
			pathSpecs = append(pathSpecs, core.PathSpec(arg))
		}

		result, err := core.CurrentRepository().Lint(rules, pathSpecs)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(result)
	},
}
