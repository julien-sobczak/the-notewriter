package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
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
		result, err := core.CurrentRepository().Status(argsToPathSpecs(args))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Print(FormatStatus(result))
	},
}

// FormatStatus returns a string representation of the status result
func FormatStatus(result *core.StatusResult) string {

	// We only output results
	var sb strings.Builder

	if len(result.ChangesStaged) > 0 {
		// Show staging area content
		sb.WriteString(`Changes to be committed:` + "\n")
		sb.WriteString(`  (use "nt restore..." to unstage)` + "\n")

		for _, change := range result.ChangesStaged {
			sb.WriteString(fmt.Sprintf(`  %10s: %s (`, change.Status, change.RelativePath))
			first := true
			if change.ObjectsDeleted > 0 {
				sb.WriteString(color.RedString("-%d", change.ObjectsDeleted))
				first = false
			}
			if change.ObjectsModified > 0 {
				if !first {
					sb.WriteString("/")
				}
				sb.WriteString(color.BlueString("%d", change.ObjectsModified))
				first = false
			}
			if change.ObjectsAdded > 0 {
				if !first {
					sb.WriteString("/")
				}
				sb.WriteString(color.GreenString("+%d", change.ObjectsAdded))
				first = false
			}
			sb.WriteString(")\n")
		}
	}

	if len(result.ChangesNotStaged) > 0 {
		sb.WriteString("\n")
		sb.WriteString(`Changes not staged for commit:` + "\n")
		sb.WriteString(`  (use "nt add <file>..." to update what will be committed)` + "\n")
		for _, change := range result.ChangesNotStaged {
			sb.WriteString(fmt.Sprintf(`  %10s: %s`+"\n", change.Status, change.RelativePath))
			// Object counts are not available for unstaged files
		}
	}

	return sb.String()
}
