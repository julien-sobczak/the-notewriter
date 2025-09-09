package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/julien-sobczak/the-notewriter/internal/core"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test-list-items <path-to-markdown-file>")
		os.Exit(1)
	}

	filePath := os.Args[1]
	absPath, _ := filepath.Abs(filePath)

	// Parse the file
	parsedFile, err := core.ParseFileFromDisk(absPath)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Find notes that have list items
	for _, note := range parsedFile.Notes {
		if len(note.Items.Children) > 0 {
			fmt.Printf("Note: %s\n", note.Title.String())
			fmt.Printf("List Items JSON:\n")
			
			jsonBytes, _ := json.MarshalIndent(note.Items, "", "  ")
			fmt.Println(string(jsonBytes))
			fmt.Println()
		}
	}
}