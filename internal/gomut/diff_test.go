package gomut_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomut "gomut/internal/gomut"
)

func TestNormalizeDiffRange(t *testing.T) {
	t.Run("given a branch name, it expands to a triple-dot diff against HEAD", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		initGitRepo(t, root)

		writeGitFile(t, root, "base.go", "package sample\n\nfunc base() bool { return true }\n")
		runGit(t, root, "add", "base.go")
		runGit(t, root, "commit", "-m", "initial")
		runGit(t, root, "branch", "-M", "main")

		runGit(t, root, "checkout", "-b", "feature")
		writeGitFile(t, root, "feature.go", "package sample\n\nfunc feature() bool { return true }\n")
		runGit(t, root, "add", "feature.go")
		runGit(t, root, "commit", "-m", "feature change")

		runGit(t, root, "checkout", "main")
		writeGitFile(t, root, "main_only.go", "package sample\n\nfunc mainOnly() bool { return true }\n")
		runGit(t, root, "add", "main_only.go")
		runGit(t, root, "commit", "-m", "main change")
		runGit(t, root, "checkout", "feature")

		// Act
		got, err := gomut.NormalizeDiffRange(context.Background(), root, "main")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "main...HEAD", got)
	})

	t.Run("given an explicit range, it preserves the value", func(t *testing.T) {
		// Arrange
		got, err := gomut.NormalizeDiffRange(context.Background(), t.TempDir(), "HEAD~1..HEAD")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "HEAD~1..HEAD", got)
	})
}

func initGitRepo(t *testing.T, root string) {
	t.Helper()

	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "Test User")
	runGit(t, root, "config", "user.email", "test@example.com")
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = root

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func writeGitFile(t *testing.T, root, name, contents string) {
	t.Helper()

	path := filepath.Join(root, name)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
}
