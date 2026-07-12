package gomut

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

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

		// Act
		runRoot, cleanup, err := prepareRunRoot(context.Background(), root, io.Discard)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, cleanup)

		copied, err := os.ReadFile(filepath.Join(runRoot, "main.go"))
		require.NoError(t, err)
		assert.Equal(t, "package main\n\nfunc main() {}\n", string(copied))

		_, err = os.Stat(filepath.Join(runRoot, ".git"))
		assert.ErrorIs(t, err, os.ErrNotExist)

		require.NoError(t, os.WriteFile(filepath.Join(runRoot, "main.go"), []byte("package main\n\nvar mutated = true\n"), 0o600))

		original, err := os.ReadFile(filepath.Join(root, "main.go"))
		require.NoError(t, err)
		assert.Equal(t, "package main\n\nfunc main() {}\n", string(original))

		require.NoError(t, cleanup())

		_, err = os.Stat(runRoot)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})
}
