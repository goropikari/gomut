package gomut

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareWorktree(t *testing.T) {
	t.Run("given a git repository, it creates and removes a detached worktree", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		runGit(t, root, "init")
		runGit(t, root, "config", "user.email", "test@example.com")
		runGit(t, root, "config", "user.name", "Test User")

		require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/worktree\n\ngo 1.26\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o600))
		runGit(t, root, "add", ".")
		runGit(t, root, "commit", "-m", "init")

		// Act
		worktreeRoot, cleanup, err := prepareWorktree(context.Background(), root)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, cleanup)

		info, err := os.Stat(worktreeRoot)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		insideCmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
		insideCmd.Dir = worktreeRoot

		inside, err := insideCmd.CombinedOutput()
		require.NoError(t, err, string(inside))

		require.NoError(t, cleanup())

		_, err = os.Stat(worktreeRoot)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
