package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(countObjectsCmd)
}

var countObjectsCmd = &cobra.Command{
	Use:   "count-objects",
	Short: "Count objects",
	Long:  `Show various counter about internal database.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		counters, err := core.CurrentCollection().Counters()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Kinds
		fmt.Println("Count per kind:")
		for _, kind := range keysSortedByValuesDesc(counters.CountKind) {
			fmt.Printf("- %s: %d\n", kind, counters.CountKind[kind])
		}
		fmt.Println("")

		// Tags
		countTags := keysSortedByValuesDesc(counters.CountTags)
		mostPopularTags := countTags
		countTags = keysSortedByValuesAsc(counters.CountTags)
		leastPopularTags := countTags
		if len(countTags) > 10 {
			mostPopularTags = mostPopularTags[0:10]
			leastPopularTags = leastPopularTags[0:10]
		}
		fmt.Println("Most popular tags")
		for _, tag := range mostPopularTags {
			fmt.Printf("- %s: %d\n", tag, counters.CountTags[tag])
		}
		fmt.Println("Least popular tags")
		for _, tag := range leastPopularTags {
			fmt.Printf("- %s: %d\n", tag, counters.CountTags[tag])
		}
		fmt.Println("")

		// Attributes
		countAttributes := keysSortedByValuesDesc(counters.CountAttributes)
		mostPopularAttributes := countAttributes
		countAttributes = keysSortedByValuesAsc(counters.CountAttributes)
		leastPopularAttributes := countAttributes
		if len(countAttributes) > 10 {
			mostPopularAttributes = mostPopularAttributes[0:10]
			leastPopularAttributes = leastPopularAttributes[0:10]
		}
		fmt.Println("Most popular attributes")
		for _, attribute := range mostPopularAttributes {
			fmt.Printf("- %s: %d\n", attribute, counters.CountAttributes[attribute])
		}
		fmt.Println("Least popular attributes")
		for _, attribute := range leastPopularAttributes {
			fmt.Printf("- %s: %d\n", attribute, counters.CountAttributes[attribute])
		}
	},
}

func keysSortedByValuesAsc(data map[string]int) []string {
	return keysSortedByValues(data, true)
}

func keysSortedByValuesDesc(data map[string]int) []string {
	return keysSortedByValues(data, false)
}

// keysSortedByValues returns the keys sorted according values.
func keysSortedByValues(data map[string]int, asc bool) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if asc {
			return data[keys[i]] < data[keys[j]]
		}
		return data[keys[i]] > data[keys[j]]
	})
	return keys
}
