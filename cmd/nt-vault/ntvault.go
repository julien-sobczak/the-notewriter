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

var rootCmd = &cobra.Command{
	Use:   "nt-vault",
	Short: "nt-vault is a tool to encrypt/decrypt markdown files",
	Long:  `A tool to encrypt and decrypt markdown files using AES encryption, inspired by ansible-vault.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Enable verbose output. The most verbose level wins when multiple flags are passed.
		if verboseInfo {
			core.CurrentLogger().SetVerboseLevel(core.VerboseInfo)
		}
		if verboseDebug {
			core.CurrentLogger().SetVerboseLevel(core.VerboseDebug)
		}
		if verboseTrace {
			core.CurrentLogger().SetVerboseLevel(core.VerboseTrace)
		}
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
		if core.CurrentRepository() != nil {
			core.CurrentRepository().Close()
		}
	}()
}

func main() {
	Execute()
}
