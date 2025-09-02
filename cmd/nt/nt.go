package main

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/julien-sobczak/the-notewriter/internal/core"
)

var verboseInfo bool
var verboseDebug bool
var verboseTrace bool

var parallel int

// Common flags used by multiple commands
var force bool
var interactive bool
var dryRun bool

var rootCmd = &cobra.Command{
	Use:   "nt",
	Short: "The NoteWriter is a file-based note management tool",
	Long:  `A Powerful and Flexible Note Management Tool using only Markdown files.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior: show a random quote
		if err := showRandomQuote(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() != "init" {
			// Ignore when configuration doesn't still exist
			CheckConfig()
		}

		// Enable verbose output. The most verbose level wins when multiple flags are passsed.
		if verboseInfo {
			core.CurrentLogger().SetVerboseLevel(core.VerboseInfo)
		}
		if verboseDebug {
			core.CurrentLogger().SetVerboseLevel(core.VerboseDebug)
		}
		if verboseTrace {
			core.CurrentLogger().SetVerboseLevel(core.VerboseTrace)
		}

		if parallel > 0 {
			core.CurrentConfig().SetParallel(parallel)
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if cmd.Name() != "init" {
			// Ignore when configuration doesn't still exist
			SaveConfig()
		}
	},
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

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		core.CurrentRepository().Close()
	}()
}

/* Quote Display Functions */

// showRandomQuote displays a random quote from the database
func showRandomQuote() error {
	quote, err := getRandomQuote()
	if err != nil {
		return err
	}
	
	if quote == nil {
		fmt.Println("No quotes found in the repository.")
		return nil
	}

	// Display LongTitle and Body with blank line between them
	fmt.Println(formatMarkdownText(string(quote.LongTitle)))
	fmt.Println()
	fmt.Println(formatMarkdownText(string(quote.Body)))
	
	return nil
}

// getRandomQuote retrieves a random quote note from the database
// IMPROVEMENT: Make this query configurable in a future version
func getRandomQuote() (*core.Note, error) {
	// Query for all Quote type notes and select randomly
	db := core.CurrentDB().Client()
	
	// Use ORDER BY RANDOM() LIMIT 1 to get a random quote efficiently
	rows, err := db.Query(`
		SELECT 
			oid,
			packfile_oid,
			file_oid,
			slug,
			note_type,
			relative_path,
			wikilink,
			title,
			long_title,
			short_title,
			attributes,
			tags,
			"line",
			content,
			hashsum,
			body,
			comment,
			created_at,
			updated_at,
			indexed_at,
			marked,
			marked_at,
			annotations
		FROM note 
		WHERE note_type = "Quote" 
		ORDER BY RANDOM() 
		LIMIT 1`)
	
	if err != nil {
		return nil, fmt.Errorf("failed to query for quotes: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil // No quotes found
	}

	// Parse the result using the same pattern as QueryNote
	var n core.Note
	var createdAt string
	var updatedAt string
	var lastIndexedAt string
	var markedAt interface{}
	var tagsRaw string
	var attributesRaw string
	var annotationsRaw string

	err = rows.Scan(
		&n.OID,
		&n.PackFileOID,
		&n.FileOID,
		&n.Slug,
		&n.Type,
		&n.RelativePath,
		&n.Wikilink,
		&n.Title,
		&n.LongTitle,
		&n.ShortTitle,
		&attributesRaw,
		&tagsRaw,
		&n.Line,
		&n.Content,
		&n.Hash,
		&n.Body,
		&n.Comment,
		&createdAt,
		&updatedAt,
		&lastIndexedAt,
		&n.Marked,
		&markedAt,
		&annotationsRaw,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan quote row: %w", err)
	}

	return &n, nil
}

// formatMarkdownText processes markdown text to remove emphasis and apply color formatting
func formatMarkdownText(text string) string {
	// First handle bold emphasis (**text** and __text__) and apply bold+cyan color
	boldRegex := regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)
	text = boldRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the content between ** or __
		content := strings.Trim(match, "*_")
		return color.New(color.FgCyan, color.Bold).Sprint(content)
	})
	
	// Then handle italic emphasis (*text* and _text_) and apply yellow color
	italicRegex := regexp.MustCompile(`\*([^*]+)\*|_([^_]+)_`)
	text = italicRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the content between * or _
		content := strings.Trim(match, "*_")
		return color.YellowString(content)
	})
	
	return text
}

func main() {
	Execute()
}

/* Utilities */

// argsToPathSpecs converts a list of arguments to a list of PathSpecs.
func argsToPathSpecs(args []string) core.PathSpecs {
	var pathSpecs []core.PathSpec
	for _, arg := range args {
		pathSpecs = append(pathSpecs, core.PathSpec(arg))
	}
	if len(pathSpecs) == 0 {
		pathSpecs = core.AnyPath
	}
	return pathSpecs
}
