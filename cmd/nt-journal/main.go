package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/julien-sobczak/the-notewriter/internal/core"
)

var verboseInfo bool
var verboseDebug bool
var verboseTrace bool

var rootCmd = &cobra.Command{
	Use:   "nt-journal",
	Short: "nt-journal manages daily journal routines",
	Run: func(cmd *cobra.Command, args []string) {
		CheckConfig()
		runInteractiveMode()
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		CheckConfig()

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

// getEditor returns the editor to use from $EDITOR or defaults to vi.
func getEditor() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	return editor
}

// openEditor opens the given file in the user's editor and waits for it to close.
func openEditor(path string) error {
	editor := getEditor()
	// Support arguments in $EDITOR (e.g. "code --wait")
	editorArgs := strings.Split(editor, " ")
	editorBinary := editorArgs[0]
	if len(editorArgs) > 1 {
		editorArgs = append(editorArgs[1:], path)
	} else {
		editorArgs = []string{path}
	}

	cmd := exec.Command(editorBinary, editorArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	Execute()
}
