package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const TitleStartupRoutine = "Morning Routine"

func init() {
	rootCmd.AddCommand(hiCmd)
}

// Run locally:
//
//	$ go run cmd/nt-journal/*.go hi
var hiCmd = &cobra.Command{
	Use:     "hi",
	Aliases: []string{"hello", "bonjour"},
	Short:   "Start the startup routine",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			fmt.Println("No argument expected")
			os.Exit(1)
		}

		CheckConfig()

		// Step 1: Determine the current mood
		emotion := ChooseEmotion(Emotions)

		// Step 2: Create the today journal file is not present
		today := time.Now()
		entryPath, err := CreateJournalEntryIfMissing(today)
		if err != nil {
			log.Fatal(err)
		}
		if err := GenerateTodaySymlink(entryPath); err != nil {
			log.Fatal(err)
		}

		// Avoid generating the routine twice
		present, err := ContainsMarkdownSection(entryPath, TitleStartupRoutine)
		if err != nil {
			log.Fatal(err)
		}
		if present {
			fmt.Println("Routine already generated today. Skipping.")
			os.Exit(1)
		}

		// Step 3: Generate the startup template
		routineContent := GenerateStartupRoutine(emotion)

		// Step 4: Append to the journal
		if err := AppendToJournal(TitleStartupRoutine, routineContent); err != nil { // TODO use config to customize the title instead
			log.Fatal(err)
		}

		fmt.Printf("✨ Routine generated at %s\n", entryPath)

		if AskToOpenInEditor() {
			if err := OpenInEditor(entryPath); err != nil {
				log.Fatal(err)
			}
		}
	},
}

func GenerateStartupRoutine(emotion *Emotion) string {
	affirmations := MustParseAffirmations(AffirmationsRaw)
	prompts := MustParsePrompts(PromptsRaw)

	var randomAffirmation *Affirmation
	var randomPrompt *Prompt

	// Select 1 affirmation + 1 prompt
	randomAffirmation = affirmations[rand.IntN(len(affirmations))]
	var promptsMatchingEmotion []*Prompt
	for _, prompt := range prompts {
		if !haveCommonElements(prompt.Tags, emotion.Tags) {
			promptsMatchingEmotion = append(promptsMatchingEmotion, prompt)
		}
	}
	if len(promptsMatchingEmotion) > 0 {
		randomPrompt = promptsMatchingEmotion[rand.IntN(len(promptsMatchingEmotion))]
	} else {
		randomPrompt = prompts[rand.IntN(len(prompts))]
	}

	if randomAffirmation == nil {
		log.Fatal("No affirmation found 😱")
	}
	if randomPrompt == nil {
		log.Fatal("No prompt found 😱")
	}

	routineContent := fmt.Sprintf(`
### 💪 Affirmation

**%s**

### 😘 Gratitude Journal

3 things I appreciate:

* ___
* ___
* ___

### 🤔 Prompt

_%s_
___


### 🎯 My BIG thing for today

___


### 📋 3+1 tasks

* [ ] ___ (work)
* [ ] ___
* [ ] ___
* [ ] ___

	`, randomAffirmation.Description, randomPrompt.Description)

	return strings.TrimSpace(routineContent)
}
