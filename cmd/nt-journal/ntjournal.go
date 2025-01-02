package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/julien-sobczak/the-notewriter/internal/core"

	_ "embed"
)

//go:embed affirmations.md
var AffirmationsRaw string

//go:embed prompts.md
var PromptsRaw string

var verboseInfo bool
var verboseDebug bool
var verboseTrace bool

var rootCmd = &cobra.Command{
	Short: "nt-journal is an extra tool to generate journal prompts",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Missing command...
			return
		}
		CheckConfig()

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

		rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	},
}

func init() {
	// Use PersistentFlags to make flags accessible to sub-commands
	rootCmd.PersistentFlags().BoolVarP(&verboseInfo, "v", "", false, "enable verbose info output")
	rootCmd.PersistentFlags().BoolVarP(&verboseDebug, "vv", "", false, "enable verbose debug output")
	rootCmd.PersistentFlags().BoolVarP(&verboseTrace, "vvv", "", false, "enable verbose trace output")
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

func CheckConfig() {
	err := core.CurrentConfig().Check()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
