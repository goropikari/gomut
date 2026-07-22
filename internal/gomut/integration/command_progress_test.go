package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRunProgressDisplay(t *testing.T) {
	t.Run("given progress enabled, it writes progress to stderr and keeps jsonl on stdout", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--jsonl", "--progress=on"})

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, stdout)
		assert.Contains(t, stderr, "Progress")
		assert.Contains(t, stderr, "Mutation summary")
	})

	t.Run("given auto progress in a buffered run, it stays quiet", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--jsonl", "--progress=auto"})

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, stdout)
		assert.NotContains(t, stderr, "Progress")
		assert.Contains(t, stderr, "Mutation summary")
	})

	t.Run("given auto progress in CI, it stays quiet", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)
		t.Setenv("CI", "true")

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--jsonl", "--progress=auto"})

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, stdout)
		assert.NotContains(t, stderr, "Progress")
		assert.Contains(t, stderr, "Mutation summary")
	})

	t.Run("given progress disabled, it does not write progress to stderr", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, stdout)
		assert.NotContains(t, stderr, "Progress")
		assert.Contains(t, stderr, "Mutation summary")
	})
}
