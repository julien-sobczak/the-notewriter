package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/julien-sobczak/the-notewriter/internal/core"
)

var verboseInfo bool
var verboseDebug bool
var verboseTrace bool

var parallel int

var rootCmd = &cobra.Command{
	Use:   "nt",
	Short: "The NoteWriter is a file-based note management tool",
	Long:  `A Powerful and Flexible Note Management Tool using only Markdown files.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Missing command...
			return
		}
		if args[0] != "init" {
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
		core.CurrentCollection().Close()
	}()
}

func main() {
	Execute()
}
