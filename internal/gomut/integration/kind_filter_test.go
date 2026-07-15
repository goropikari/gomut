package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRunKindFilter(t *testing.T) {
	t.Run("given no kind flags, it uses the standard kind set", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./sample", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		for _, record := range records {
			assert.Contains(t, []string{
				"comparison_operator",
				"logical_operator",
				"arithmetic_operator",
				"guard_clause",
				"return",
				"nil_check",
			}, string(record.Mutation.Kind))
		}
	})

	t.Run("given enable and disable flags, disable removes the requested kind", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./sample", "--enable", "bitwise_operator", "--disable", "return", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		seenBitwise := false

		for _, record := range records {
			assert.NotEqual(t, "return", string(record.Mutation.Kind))

			if string(record.Mutation.Kind) == "bitwise_operator" {
				seenBitwise = true
			}
		}

		assert.True(t, seenBitwise, "expected bitwise_operator mutations to be enabled")
	})

	t.Run("given a config file with all mode and disable entries, it excludes the disabled kinds", func(t *testing.T) {
		// Arrange
		root := createAllModeConfigFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./sample", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		seenComparison := false

		for _, record := range records {
			assert.NotEqual(t, "string_literal", string(record.Mutation.Kind))

			if string(record.Mutation.Kind) == "comparison_operator" {
				seenComparison = true
			}
		}

		assert.True(t, seenComparison, "expected comparison_operator mutations to be included")
	})

	t.Run("given config disable entries and a CLI enable, the CLI value wins for the requested kind", func(t *testing.T) {
		// Arrange
		root := createAllModeConfigFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./sample", "--enable", "string_literal", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		seenStringLiteral := false

		for _, record := range records {
			if string(record.Mutation.Kind) == "string_literal" {
				seenStringLiteral = true
			}
		}

		assert.True(t, seenStringLiteral, "expected string_literal mutations to be enabled by the CLI")
	})

	t.Run("given config kind settings and a CLI disable, the CLI value wins for the requested kind", func(t *testing.T) {
		// Arrange
		root := createKindConfigFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./sample", "--disable", "bitwise_operator", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		for _, record := range records {
			assert.NotEqual(t, "bitwise_operator", string(record.Mutation.Kind))
		}
	})

	t.Run("given an unknown mutation kind mode, it fails before running baseline tests", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./sample", "--mode", "experimental", "--jsonl", "--progress=off"})

		// Assert
		require.Error(t, err)
		assert.Empty(t, stdout)
		assert.Empty(t, stderr)
		assert.Contains(t, err.Error(), "experimental")
		assert.Contains(t, err.Error(), "standard")
		assert.Contains(t, err.Error(), "all")
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
  mode: standard
  enable:
    - bitwise_operator
  disable:
    - return
jsonl: default-kind.jsonl
`), 0o600))

	return root
}

func createAllModeConfigFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/allkindconfigtest\n\ngo 1.26\n"), 0o600))
	writeConfigFixturePackage(t, root, "sample")

	require.NoError(t, os.WriteFile(filepath.Join(root, ".gomut.yaml"), []byte(`target:
  mode: package
  value: ./sample
kind:
  mode: all
  disable:
    - string_literal
jsonl: all-kind.jsonl
`), 0o600))

	return root
}
