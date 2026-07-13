package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRunPackagePattern(t *testing.T) {
	t.Run("given a package pattern on the CLI, it expands to multiple packages", func(t *testing.T) {
		// Arrange
		root := createPackagePatternFixture(t, false)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./sample/...", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr, "Mutation summary")
		assert.NotContains(t, stderr, "Progress")

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		last := records[len(records)-1]
		assert.Equal(t, "./sample/...", last.Target.Value)
		assert.Positive(t, last.Summary.Total)

		files := map[string]struct{}{}
		for _, record := range records {
			files[record.Mutation.File] = struct{}{}
		}

		_, alphaOK := files[filepath.ToSlash(filepath.Join("sample", "alpha", "alpha.go"))]
		_, betaOK := files[filepath.ToSlash(filepath.Join("sample", "beta", "beta.go"))]

		assert.True(t, alphaOK, "expected alpha package mutations to be included")
		assert.True(t, betaOK, "expected beta package mutations to be included")
	})

	t.Run("given a package pattern in config, it uses the same expansion", func(t *testing.T) {
		// Arrange
		root := createPackagePatternFixture(t, true)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--jsonl", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr, "Mutation summary")
		assert.NotContains(t, stderr, "Progress")

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		last := records[len(records)-1]
		assert.Equal(t, "./sample/...", last.Target.Value)
		assert.Positive(t, last.Summary.Total)

		files := map[string]struct{}{}
		for _, record := range records {
			files[record.Mutation.File] = struct{}{}
		}

		_, alphaOK := files[filepath.ToSlash(filepath.Join("sample", "alpha", "alpha.go"))]
		_, betaOK := files[filepath.ToSlash(filepath.Join("sample", "beta", "beta.go"))]

		assert.True(t, alphaOK, "expected alpha package mutations to be included")
		assert.True(t, betaOK, "expected beta package mutations to be included")
	})

	t.Run("given an invalid package pattern, it fails with a useful error", func(t *testing.T) {
		// Arrange
		root := createPackagePatternFixture(t, false)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./missing/...", "--jsonl", "--progress=off"})

		// Assert
		require.Error(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Creating isolated temporary copy")
		assert.Contains(t, err.Error(), "list packages for target")
		assert.Contains(t, err.Error(), "./missing/...")
	})
}

func createPackagePatternFixture(t *testing.T, withConfig bool) string {
	t.Helper()

	root := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/patterntest\n\ngo 1.26\n"), 0o600))

	writePatternFixturePackage(t, root, "alpha")
	writePatternFixturePackage(t, root, "beta")

	if withConfig {
		require.NoError(t, os.WriteFile(filepath.Join(root, ".gomut.yaml"), []byte(`target:
  mode: package
  value: ./sample/...
jsonl: pattern.jsonl
`), 0o600))
	}

	return root
}

func writePatternFixturePackage(t *testing.T, root, pkg string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Join(root, "sample", pkg), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sample", pkg, pkg+".go"), []byte(`package `+pkg+`

func IsAtLeast(age int) bool {
	return age >= 18
}
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sample", pkg, pkg+"_test.go"), []byte(`package `+pkg+`

import "testing"

func TestIsAtLeast(t *testing.T) {
	if !IsAtLeast(20) {
		t.Fatal("expected adult input to be accepted")
	}
}
`), 0o600))
}
