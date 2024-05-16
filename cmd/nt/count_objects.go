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
		stats, err := core.CurrentRepository().StatsInDB()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Kinds
		fmt.Println("Count per kind:")
		kinds := make(map[string]int)
		for key, value := range stats.Kinds {
			kinds[string(key)] = value
		}
		for _, kind := range keysSortedByValuesDesc(kinds) {
			fmt.Printf("- %s: %d\n", kind, kinds[kind])
		}
		fmt.Println("")

		// Tags
		countTags := keysSortedByValuesDesc(stats.Tags)
		mostPopularTags := countTags
		countTags = keysSortedByValuesAsc(stats.Tags)
		leastPopularTags := countTags
		if len(countTags) > 10 {
			mostPopularTags = mostPopularTags[0:10]
			leastPopularTags = leastPopularTags[0:10]
		}
		fmt.Println("Most popular tags")
		for _, tag := range mostPopularTags {
			fmt.Printf("- %s: %d\n", tag, stats.Tags[tag])
		}
		fmt.Println("Least popular tags")
		for _, tag := range leastPopularTags {
			fmt.Printf("- %s: %d\n", tag, stats.Tags[tag])
		}
		fmt.Println("")

		// Attributes
		countAttributes := keysSortedByValuesDesc(stats.Attributes)
		mostPopularAttributes := countAttributes
		countAttributes = keysSortedByValuesAsc(stats.Attributes)
		leastPopularAttributes := countAttributes
		if len(countAttributes) > 10 {
			mostPopularAttributes = mostPopularAttributes[0:10]
			leastPopularAttributes = leastPopularAttributes[0:10]
		}
		fmt.Println("Most popular attributes")
		for _, attribute := range mostPopularAttributes {
			fmt.Printf("- %s: %d\n", attribute, stats.Attributes[attribute])
		}
		fmt.Println("Least popular attributes")
		for _, attribute := range leastPopularAttributes {
			fmt.Printf("- %s: %d\n", attribute, stats.Attributes[attribute])
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
