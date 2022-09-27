package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"

	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/julien-sobczak/the-notetaker/internal/wikipedia"
	"github.com/julien-sobczak/the-notetaker/internal/zotero"
)

var rootCmd = &cobra.Command{
	Use:   "the-notetaker",
	Short: "The NoteTaker is a file-based note management tool",
	Long:  `A Powerful and Flexible Note Management Tool using only Markdown files.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

var CollectionDir string
var Col *core.Collection

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringVarP(&CollectionDir, "collection", "c", "", "Collection directory (default is $HOME/notes)")
	// TODO add support for an environment variable to override this flag
}

func initConfig() {
	if CollectionDir == "" {
		// Search in home directory
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		CollectionDir = path.Join(home, "notes")
		if _, err := os.Stat(CollectionDir); os.IsNotExist(err) {
			fmt.Printf("Default collection %q doesn't exists. Use flag '--collection' to override the default location.", CollectionDir)
			os.Exit(1)
		}
	}

	var err error
	zoteroManager := zotero.NewReferenceManager()
	wikipediaManager := wikipedia.NewReferenceManager()
	Col, err = core.NewCollection(CollectionDir, zoteroManager, wikipediaManager)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
