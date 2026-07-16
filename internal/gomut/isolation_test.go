package gomut_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/goropikari/gomut/internal/gomut"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareRunRoot(t *testing.T) {
	t.Run("given a repository root, it copies the tree into an isolated directory", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/isolation\n\ngo 1.26\n"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o600))
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o600))
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".agents"), 0o555))
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".cache"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".codex"), 0o555))
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".worktree"), 0o755))

		// Act
		runRoot, cleanup, err := gomut.PrepareRunRoot(context.Background(), root, io.Discard)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, cleanup)

		copied, err := os.ReadFile(filepath.Join(runRoot, "main.go"))
		require.NoError(t, err)
		assert.Equal(t, "package main\n\nfunc main() {}\n", string(copied))

		_, err = os.Stat(filepath.Join(runRoot, ".git"))
		require.ErrorIs(t, err, os.ErrNotExist)

		_, err = os.Stat(filepath.Join(runRoot, ".agents"))
		require.ErrorIs(t, err, os.ErrNotExist)

		_, err = os.Stat(filepath.Join(runRoot, ".cache"))
		require.ErrorIs(t, err, os.ErrNotExist)

		_, err = os.Stat(filepath.Join(runRoot, ".codex"))
		require.ErrorIs(t, err, os.ErrNotExist)

		_, err = os.Stat(filepath.Join(runRoot, ".worktree"))
		require.ErrorIs(t, err, os.ErrNotExist)

		require.NoError(t, os.WriteFile(filepath.Join(runRoot, "main.go"), []byte("package main\n\nvar mutated = true\n"), 0o600))

		original, err := os.ReadFile(filepath.Join(root, "main.go"))
		require.NoError(t, err)
		assert.Equal(t, "package main\n\nfunc main() {}\n", string(original))

		require.NoError(t, cleanup())

		_, err = os.Stat(runRoot)
		require.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestPrepareRunRootWithCopyExclude(t *testing.T) {
	t.Run("given configured copy excludes, it skips matching entries", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o600))
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".cache"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(root, "tmp"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "tmp", "scratch.txt"), []byte("scratch"), 0o600))
		require.NoError(t, os.MkdirAll(filepath.Join(root, "internal", "cache", "nested"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "internal", "cache", "nested", "data.txt"), []byte("data"), 0o600))
		require.NoError(t, os.MkdirAll(filepath.Join(root, "internal", "keep"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "internal", "keep", "keep.go"), []byte("package keep\n"), 0o600))

		// Act
		runRoot, cleanup, err := gomut.PrepareRunRootWithCopyExclude(context.Background(), root, io.Discard, []string{"tmp", "internal/cache/**"})

		// Assert
		require.NoError(t, err)
		defer func() {
			require.NoError(t, cleanup())
		}()

		assert.NoFileExists(t, filepath.Join(runRoot, ".cache"))
		assert.NoFileExists(t, filepath.Join(runRoot, "tmp", "scratch.txt"))
		assert.NoFileExists(t, filepath.Join(runRoot, "internal", "cache", "nested", "data.txt"))
		assert.FileExists(t, filepath.Join(runRoot, "main.go"))
		assert.FileExists(t, filepath.Join(runRoot, "internal", "keep", "keep.go"))
	})
}
