package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRunExclusionVerbose(t *testing.T) {
	t.Run("given the default verbosity, it suppresses exclusion notices", func(t *testing.T) {
		// Arrange
		root := createExclusionNoticeFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./sample", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.NotContains(t, stderr, "excluded by pattern")
		assert.Contains(t, stderr, "Mutation summary")
	})

	t.Run("given verbose mode, it prints exclusion notices", func(t *testing.T) {
		// Arrange
		root := createExclusionNoticeFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./sample", "--jsonl", "--progress=off", "--verbose"})

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "excluded by pattern")
		assert.Contains(t, stderr, "Mutation summary")
	})
}

func createExclusionNoticeFixture(t *testing.T) string {
	t.Helper()

	root := createResultFilterFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gomut.yaml"), []byte(`exclude:
  - sample.go
`), 0o600))

	return root
}
