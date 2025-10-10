package main

import (
	"os"
	"os/exec"
	"strings"
)

// GetEditor returns the editor to use (from $EDITOR or default to vi)
func GetEditor() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	return editor
}

// GetPager returns the pager to use (from $PAGER or default to less)
func GetPager() string {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}
	return pager
}

// GetEditorCmd returns an exec.Cmd for the editor with the given arguments
func GetEditorCmd(args ...string) *exec.Cmd {
	editor := GetEditor()
	// Support arguments in $EDITOR as VS Code requires to use 'code --wait' to wait for the tab to be closed
	editorArgs := strings.Split(editor, " ")
	editorBinary := editorArgs[0]
	if len(editorArgs) > 1 {
		editorArgs = editorArgs[1:]
	} else {
		editorArgs = []string{}
	}
	editorArgs = append(editorArgs, args...)

	return exec.Command(editorBinary, editorArgs...)
}
