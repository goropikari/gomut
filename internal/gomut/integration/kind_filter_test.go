package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRunKindFilter(t *testing.T) {
	t.Run("given a single mutation kind, it writes only matching records", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--kind", "comparison_operator", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		for _, record := range records {
			assert.Equal(t, "comparison_operator", string(record.Mutation.Kind))
		}

		last := records[len(records)-1]
		assert.Equal(t, len(records), last.Summary.Total)
	})

	t.Run("given comma-separated and repeated mutation kinds, it writes only matching records", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--kind", "comparison_operator,return", "--kind", "nil_check", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		for _, record := range records {
			assert.Contains(t, []string{"comparison_operator", "return", "nil_check"}, string(record.Mutation.Kind))
		}

		last := records[len(records)-1]
		assert.Equal(t, len(records), last.Summary.Total)
	})

	t.Run("given an unknown mutation kind, it fails before running baseline tests", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--kind", "comparsion_operator", "--jsonl", "--progress=off"})

		// Assert
		require.Error(t, err)
		assert.Empty(t, stdout)
		assert.Empty(t, stderr)
		assert.Contains(t, err.Error(), "comparsion_operator")
		assert.Contains(t, err.Error(), "comparison_operator")
	})

	t.Run("given config kinds and a CLI kind, it prefers the CLI value", func(t *testing.T) {
		// Arrange
		root := createKindConfigFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--kind", "comparison_operator", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		for _, record := range records {
			assert.Equal(t, "comparison_operator", string(record.Mutation.Kind))
		}
	})
}

func createKindConfigFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/kindconfigtest\n\ngo 1.26\n"), 0o600))
	writeConfigFixturePackage(t, root, "sample")

	require.NoError(t, os.WriteFile(filepath.Join(root, ".gomut.yaml"), []byte(`target:
  mode: package
  value: ./sample
kind:
  - return
jsonl: default-kind.jsonl
`), 0o600))

	return root
}
