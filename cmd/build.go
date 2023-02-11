package cmd

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/spf13/cobra"
)

var outputDirectory string

func init() {
	buildCmd.Flags().StringVarP(&outputDirectory, "output", "o", "./build", "directory containing the generated resources")
	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build database",
	Long:  `Parse files and refresh the database.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		_, err := core.CurrentCollection().Build(outputDirectory)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}
