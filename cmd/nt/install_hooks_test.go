package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindGitRoot(t *testing.T) {
	t.Run("finds git root in repository", func(t *testing.T) {
		// Create a temporary git repository
		tmpDir := t.TempDir()
		cmd := exec.Command("git", "init")
		cmd.Dir = tmpDir
		err := cmd.Run()
		require.NoError(t, err)

		// Change to a subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		err = os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)

		err = os.Chdir(subDir)
		require.NoError(t, err)

		// Test finding git root
		gitRoot, err := findGitRoot()
		require.NoError(t, err)
		assert.Equal(t, tmpDir, gitRoot)
	})

	t.Run("returns error when not in git repository", func(t *testing.T) {
		tmpDir := t.TempDir()

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		_, err = findGitRoot()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
	})
}

func TestCheckExistingHooks(t *testing.T) {
	t.Run("passes when no hooks exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		hooksDir := filepath.Join(tmpDir, ".git", "hooks")
		err := os.MkdirAll(hooksDir, 0755)
		require.NoError(t, err)

		err = checkExistingHooks(hooksDir)
		assert.NoError(t, err)
	})

	t.Run("returns error when hooks exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		hooksDir := filepath.Join(tmpDir, ".git", "hooks")
		err := os.MkdirAll(hooksDir, 0755)
		require.NoError(t, err)

		// Create a pre-commit hook
		hookPath := filepath.Join(hooksDir, "pre-commit")
		err = os.WriteFile(hookPath, []byte("#!/bin/bash\n"), 0755)
		require.NoError(t, err)

		err = checkExistingHooks(hooksDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pre-commit")
	})

	t.Run("lists all existing hooks", func(t *testing.T) {
		tmpDir := t.TempDir()
		hooksDir := filepath.Join(tmpDir, ".git", "hooks")
		err := os.MkdirAll(hooksDir, 0755)
		require.NoError(t, err)

		// Create multiple hooks
		hooks := []string{"pre-commit", "post-commit", "pre-push"}
		for _, hook := range hooks {
			hookPath := filepath.Join(hooksDir, hook)
			err = os.WriteFile(hookPath, []byte("#!/bin/bash\n"), 0755)
			require.NoError(t, err)
		}

		err = checkExistingHooks(hooksDir)
		assert.Error(t, err)
		for _, hook := range hooks {
			assert.Contains(t, err.Error(), hook)
		}
	})
}

func TestGenerateHookScript(t *testing.T) {
	t.Run("generates pre-commit hook script", func(t *testing.T) {
		script := generateHookScript("pre-commit", "")
		assert.Contains(t, script, "#!/bin/bash")
		assert.Contains(t, script, "nt lint")
		assert.Contains(t, script, "nt add .")
		assert.Contains(t, script, "set -e")
	})

	t.Run("generates post-commit hook script", func(t *testing.T) {
		script := generateHookScript("post-commit", "")
		assert.Contains(t, script, "#!/bin/bash")
		assert.Contains(t, script, "nt commit")
		assert.Contains(t, script, "set -e")
	})

	t.Run("generates pre-push hook script with remote", func(t *testing.T) {
		script := generateHookScript("pre-push", "origin")
		assert.Contains(t, script, "#!/bin/bash")
		assert.Contains(t, script, "nt push origin")
		assert.Contains(t, script, "set -e")
	})

	t.Run("generates pre-push hook script with custom remote", func(t *testing.T) {
		script := generateHookScript("pre-push", "backup")
		assert.Contains(t, script, "nt push backup")
	})
}

func TestInstallHooks(t *testing.T) {
	t.Run("installs selected hooks", func(t *testing.T) {
		tmpDir := t.TempDir()
		hooksDir := filepath.Join(tmpDir, ".git", "hooks")

		selectedHooks := []string{"pre-commit", "post-commit"}
		err := installHooks(hooksDir, selectedHooks, "")
		require.NoError(t, err)

		// Verify hooks were created
		for _, hook := range selectedHooks {
			hookPath := filepath.Join(hooksDir, hook)
			assert.FileExists(t, hookPath)

			// Verify executable bit is set
			info, err := os.Stat(hookPath)
			require.NoError(t, err)
			assert.NotEqual(t, 0, info.Mode()&0111, "hook should be executable")

			// Verify content
			content, err := os.ReadFile(hookPath)
			require.NoError(t, err)
			assert.Contains(t, string(content), "#!/bin/bash")
		}
	})

	t.Run("installs pre-push with remote", func(t *testing.T) {
		tmpDir := t.TempDir()
		hooksDir := filepath.Join(tmpDir, ".git", "hooks")

		selectedHooks := []string{"pre-push"}
		err := installHooks(hooksDir, selectedHooks, "origin")
		require.NoError(t, err)

		hookPath := filepath.Join(hooksDir, "pre-push")
		content, err := os.ReadFile(hookPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "nt push origin")
	})

	t.Run("creates hooks directory if it doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		hooksDir := filepath.Join(tmpDir, ".git", "hooks")

		// Ensure directory doesn't exist
		_, err := os.Stat(hooksDir)
		assert.True(t, os.IsNotExist(err))

		selectedHooks := []string{"pre-commit"}
		err = installHooks(hooksDir, selectedHooks, "")
		require.NoError(t, err)

		// Verify directory was created
		info, err := os.Stat(hooksDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("hook scripts have proper line endings", func(t *testing.T) {
		tmpDir := t.TempDir()
		hooksDir := filepath.Join(tmpDir, ".git", "hooks")

		selectedHooks := []string{"pre-commit"}
		err := installHooks(hooksDir, selectedHooks, "")
		require.NoError(t, err)

		hookPath := filepath.Join(hooksDir, "pre-commit")
		content, err := os.ReadFile(hookPath)
		require.NoError(t, err)

		// Verify Unix line endings
		assert.NotContains(t, string(content), "\r\n")
		assert.Contains(t, string(content), "\n")
	})
}

func TestHookScriptContent(t *testing.T) {
	t.Run("pre-commit script runs lint before add", func(t *testing.T) {
		script := generateHookScript("pre-commit", "")
		lines := strings.Split(script, "\n")

		lintIdx := -1
		addIdx := -1
		for i, line := range lines {
			if strings.Contains(line, "nt lint") {
				lintIdx = i
			}
			if strings.Contains(line, "nt add") {
				addIdx = i
			}
		}

		assert.Greater(t, lintIdx, 0, "lint command should be present")
		assert.Greater(t, addIdx, 0, "add command should be present")
		assert.Less(t, lintIdx, addIdx, "lint should run before add")
	})
}

func TestInstallHooksEmptySelection(t *testing.T) {
	t.Run("no hooks selected installs nothing", func(t *testing.T) {
		tmpDir := t.TempDir()
		hooksDir := filepath.Join(tmpDir, ".git", "hooks")

		// Install empty list of hooks
		selectedHooks := []string{}
		err := installHooks(hooksDir, selectedHooks, "")
		require.NoError(t, err)

		// Verify hooks directory exists but is empty
		entries, err := os.ReadDir(hooksDir)
		require.NoError(t, err)
		assert.Equal(t, 0, len(entries), "no hooks should be installed")
	})
}
