package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/julien-sobczak/the-notetaker/internal/core"
)

var verboseInfo bool
var verboseDebug bool
var verboseTrace bool

var CollectionDir string

var rootCmd = &cobra.Command{
	Use:   "nt",
	Short: "The NoteTaker is a file-based note management tool",
	Long:  `A Powerful and Flexible Note Management Tool using only Markdown files.`,
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()

		// Enable verbose output. The most verbose level wins when multiple flags are passsed.
		if verboseInfo {
			core.CurrentConfig().SetVerboseLevel(core.VerboseInfo)
		}
		if verboseDebug {
			core.CurrentConfig().SetVerboseLevel(core.VerboseDebug)
		}
		if verboseTrace {
			core.CurrentConfig().SetVerboseLevel(core.VerboseTrace)
		}

	},
}

func init() {
	rootCmd.Flags().BoolVarP(&verboseInfo, "verbose", "v", false, "enable verbose info output")
	rootCmd.Flags().BoolVarP(&verboseDebug, "verbose-debug", "vv", false, "enable verbose debug output")
	rootCmd.Flags().BoolVarP(&verboseTrace, "verbose-trace", "vvv", false, "enable verbose trace output")
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
		<-sigs
		core.CurrentCollection().Close()
	}()
}

func CheckConfig() {
	core.CurrentConfig()
}
