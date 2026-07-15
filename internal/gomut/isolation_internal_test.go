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
		runRoot, cleanup, err := prepareRunRoot(context.Background(), root, io.Discard, []string{"tmp", "internal/cache/**"})

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
