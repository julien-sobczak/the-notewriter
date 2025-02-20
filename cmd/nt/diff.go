package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

var cached bool
var staged bool

func init() {
	diffCmd.Flags().BoolVarP(&cached, "cached", "", false, "Show staged changes")
	diffCmd.Flags().BoolVarP(&staged, "staged", "", false, "Show staged changes")
	rootCmd.AddCommand(diffCmd)
}

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show changes",
	Long:  `Show changes between commit and working tree.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			fmt.Println("Too many arguments. No argument is supported.")
			os.Exit(1)
		}

		stagedOrCached := staged || cached
		diff, err := core.CurrentRepository().Diff(argsToPathSpecs(args), stagedOrCached)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Output diff using $PAGER like git
		pagerCmd := exec.Command(os.Getenv("PAGER"))
		// Feed it with the string you want to display.
		pagerCmd.Stdin = strings.NewReader(formatDiff(diff, true))
		// This is crucial - otherwise it will write to a null device.
		pagerCmd.Stdout = os.Stdout
		err = pagerCmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	},
}

func formatDiff(result core.ObjectDiffs, colored bool) string {
	var sb strings.Builder
	for _, diff := range result {
		patch := diff.Patch()
		if colored {
			patch = coloredDiff(patch)
		}

		sb.WriteString(patch)
		sb.WriteString("\n")
	}
	return sb.String()
}

func coloredDiff(diff string) string {
	var sb strings.Builder
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			sb.WriteString(color.RedString(line))
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			sb.WriteString(color.GreenString(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
