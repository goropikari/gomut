package gomut_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRunConfig(t *testing.T) {
	t.Run("given a default config file, it runs with the config values", func(t *testing.T) {
		// Arrange
		root := createConfigFixture(t)
		jsonlPath := filepath.Join(root, "default-config.jsonl")

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test"})

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Progress")
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, mustReadFile(t, jsonlPath))
		require.NotEmpty(t, records)
		assert.Equal(t, "./sample", records[0].Target.Value)
	})

	t.Run("given an explicit config file path, it loads settings from that file", func(t *testing.T) {
		// Arrange
		root := createConfigFixture(t)
		jsonlPath := filepath.Join(root, "explicit-config.jsonl")

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--config", filepath.Join("configs", "gomut.yaml")})

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, mustReadFile(t, jsonlPath))
		require.NotEmpty(t, records)
		assert.Equal(t, "./alt", records[0].Target.Value)
	})

	t.Run("given config values and overriding CLI flags, it uses the flags", func(t *testing.T) {
		// Arrange
		root := createConfigFixture(t)
		jsonlPath := filepath.Join(root, "override.jsonl")

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./alt", "--jsonl", jsonlPath})

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, mustReadFile(t, jsonlPath))
		require.NotEmpty(t, records)
		assert.Equal(t, "./alt", records[0].Target.Value)
	})

	t.Run("given config progress and an explicit progress flag, it uses the flag", func(t *testing.T) {
		// Arrange
		root := createConfigFixture(t)
		jsonlPath := filepath.Join(root, "progress-override.jsonl")

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--jsonl", jsonlPath, "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.NotContains(t, stderr, "Progress")
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, mustReadFile(t, jsonlPath))
		require.NotEmpty(t, records)
		assert.Equal(t, "./sample", records[0].Target.Value)
	})

	t.Run("given no config file, it still runs with CLI flags", func(t *testing.T) {
		// Arrange
		root := createNoConfigFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--jsonl"})

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, stdout)
		assert.Contains(t, stderr, "Mutation summary")
	})

	t.Run("given a malformed config file, it fails with a clear error", func(t *testing.T) {
		// Arrange
		root := createConfigFixture(t)
		require.NoError(t, os.WriteFile(filepath.Join(root, ".gomut.yaml"), []byte("target:\n  mode: [\n"), 0o600))

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test"})

		// Assert
		require.Error(t, err)
		assert.Empty(t, stdout)
		assert.Empty(t, stderr)
		assert.Contains(t, strings.ToLower(err.Error()), "config")
	})
}

func createConfigFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/configtest\n\ngo 1.26\n"), 0o600))

	writeConfigFixturePackage(t, root, "sample")
	writeConfigFixturePackage(t, root, "alt")

	require.NoError(t, os.MkdirAll(filepath.Join(root, "configs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gomut.yaml"), []byte(`target:
  mode: package
  value: ./sample
timeout: 10s
progress: on
jsonl: default-config.jsonl
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "configs", "gomut.yaml"), []byte(`target:
  mode: package
  value: ./alt
timeout: 20s
progress: off
jsonl: explicit-config.jsonl
`), 0o600))

	return root
}

func createNoConfigFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/noconfigtest\n\ngo 1.26\n"), 0o600))
	writeConfigFixturePackage(t, root, "sample")

	return root
}

func writeConfigFixturePackage(t *testing.T, root, pkg string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Join(root, pkg), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, pkg, pkg+".go"), []byte(`package `+pkg+`

func IsAtLeast(age int) bool {
	return age >= 18
}

func Double(value int) int {
	return value + value
}
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, pkg, pkg+"_test.go"), []byte(`package `+pkg+`

import "testing"

func TestIsAtLeast(t *testing.T) {
	if !IsAtLeast(20) {
		t.Fatal("expected adult input to be accepted")
	}
}

func TestDouble(t *testing.T) {
	if got := Double(2); got != 4 {
		t.Fatalf("expected double to be 4, got %d", got)
	}
}
`), 0o600))
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(data)
}
