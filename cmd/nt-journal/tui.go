package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
)

/*
 * The command nt-journal is a TUI.
 * All BubbleTea-related code is present in this file to make easy to refactor or switch to another library someday.
 * This version uses charmbracelet/huh for TUI forms and selection.
 */

func ChooseEmotion(emotions []*Emotion) *Emotion {
	options := make([]huh.Option[string], len(emotions))
	for i, emotion := range emotions {
		options[i] = huh.NewOption(emotion.Emoji+" "+emotion.Title, emotion.Key)
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("How do you feel?").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, emotion := range emotions {
		if emotion.Key == selected {
			return emotion
		}
	}
	return nil
}

func AskToOpenInEditor() bool {
	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Open in the editor?").
				Options(
					huh.NewOption("Yes", "yes"),
					huh.NewOption("No", "no"),
				).
				Value(&choice),
		),
	)
	if err := form.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return choice == "yes"
}
