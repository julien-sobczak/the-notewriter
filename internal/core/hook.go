package core

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/julien-sobczak/the-notetaker/pkg/text"
)

// RunHooks triggers all hooks on the note.
func (n *Note) RunHooks() error {
	hookValue := n.GetAttribute("hook")
	if hookValue == nil {
		// No hooks on this note
		return nil
	}
	hooks, ok := hookValue.([]interface{})
	if !ok {
		return fmt.Errorf("invalid type for hook attribute")
	}
	if len(hooks) == 0 {
		// Nothing to do
		return nil
	}

	// Start by checking all hook executable exists
	hookDir := filepath.Join(CurrentConfig().RootDirectory, ".nt", "hooks")
	hookExecutables := map[string]string{}
	for _, hookNameRaw := range hooks {
		hookName := hookNameRaw.(string)

		// Search for an executable file named `hookName(.ext)?` under `.nt/hooks`
		var matchingExecutableFiles []string
		filepath.WalkDir(hookDir, func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if text.TrimExtension(filepath.Base(path)) != hookName {
				// Ignore the extension when searching for a hook executable file
				return nil
			}

			// Ignore not executable files
			fileInfo, err := os.Lstat(path) // NB: os.Stat follows symlinks
			if err != nil {
				// Ignore the file
				return nil
			}
			if !IsExec(fileInfo.Mode()) {
				// Ignore files without any +x set (ex: rw-rw-rw-)
				return nil
			}

			matchingExecutableFiles = append(matchingExecutableFiles, path)

			return nil
		})
		if len(matchingExecutableFiles) == 0 {
			return fmt.Errorf("no executable hook file named %s found", hookName)
		}
		if len(matchingExecutableFiles) > 1 {
			return fmt.Errorf("multiple possible executable files for hook %s: %s", hookName, strings.Join(matchingExecutableFiles, ","))
		}

		// Found the match!
		hookExecutables[hookName] = matchingExecutableFiles[0]
	}

	// Trigger the hook commands
	for _, hookNameRaw := range hooks {
		hookName := hookNameRaw.(string)
		exe := hookExecutables[hookName]
		CurrentLogger().Infof("Running hook %q on %s...", hookName, n)
		err := n.executeHook(exe)
		if err != nil {
			return err
		}
	}

	return nil
}

// executeHook executes the given executable file path.
func (n *Note) executeHook(exe string) error {
	// We will write the JSON representation of the note to stding
	noteJson := n.FormatToJSON()

	// See https://stackoverflow.com/a/23167416
	subProcess := exec.Command(exe)
	stdin, err := subProcess.StdinPipe()
	if err != nil {
		return err
	}
	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr

	if err = subProcess.Start(); err != nil {
		return err
	}
	io.WriteString(stdin, noteJson)
	err = stdin.Close()
	if err != nil {
		return err
	}
	err = subProcess.Wait()
	if err != nil {
		return err
	}

	return nil
}

/* Helpers */

func IsExecOwner(mode os.FileMode) bool {
	return mode&0100 != 0
}

func IsExecOther(mode os.FileMode) bool {
	return mode&0001 != 0
}

func IsExecAny(mode os.FileMode) bool {
	return mode&0111 != 0
}

func IsExec(mode os.FileMode) bool {
	return IsExecOwner(mode) || IsExecOther(mode) || IsExecAny(mode)
}
