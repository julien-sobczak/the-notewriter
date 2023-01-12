package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/julien-sobczak/the-notetaker/internal/core"
)

var rootCmd = &cobra.Command{
	Use:   "nt",
	Short: "The NoteTaker is a file-based note management tool",
	Long:  `A Powerful and Flexible Note Management Tool using only Markdown files.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

var CollectionDir string
var Col *core.Collection

func init() {
	rootCmd.Flags().StringVarP(&CollectionDir, "collection", "c", "", "Collection directory (default is $HOME/notes)")
	// TODO add support for an environment variable to override this flag
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		_ = <-sigs
		Col.Close()
	}()
}

func CheckConfig() {
	core.CurrentConfig()
}
