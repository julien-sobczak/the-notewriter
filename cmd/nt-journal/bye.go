package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const TitleShutdownRoutine = "Shutdown Routine"

func init() {
	rootCmd.AddCommand(byeCmd)
}

// Run locally:
//
//	$ go run cmd/nt-journal/*.go bye
var byeCmd = &cobra.Command{
	Use:     "bye",
	Aliases: []string{"bye-bye"},
	Short:   "Start the shutdown routine",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			fmt.Println("No argument expected")
			os.Exit(1)
		}

		CheckConfig()

		// Step 1: Create the today journal file is not present
		today := time.Now()
		entryPath, err := CreateJournalEntryIfMissing(today)
		if err != nil {
			log.Fatal(err)
		}
		if err := GenerateTodaySymlink(entryPath); err != nil {
			log.Fatal(err)
		}

		// Avoid generating the routine twice
		present, err := ContainsMarkdownSection(entryPath, TitleShutdownRoutine)
		if err != nil {
			log.Fatal(err)
		}
		if present {
			fmt.Println("Routine already generated today. Skipping.")
			os.Exit(1)
		}

		// Step 2: Generate the shutdown template
		routineContent := GenerateShutdownRoutine()

		// Step 3: Append to the journal
		if err := AppendToJournal(TitleShutdownRoutine, routineContent); err != nil { // TODO use config to customize the title instead
			log.Fatal(err)
		}

		fmt.Printf("✨ Routine generated. Browse %s\n", entryPath) // TODO use bubbletea to ask to open the file instead and run command `open`

		if AskToOpenInEditor() {
			if err := OpenInEditor(entryPath); err != nil {
				log.Fatal(err)
			}
		}
	},
}

func GenerateShutdownRoutine() string {
	routineContent := `
### ❓ How was my day? Why?

___

### 📋 3+1 tasks to complete tomorrow:

* [ ] ___ (work)
* [ ] ___
* [ ] ___
* [ ] ___
	`

	return strings.TrimSpace(routineContent)
}
