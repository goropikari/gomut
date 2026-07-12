package gomut_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomut "gomut/internal/gomut"
)

func TestExclusionFilterSkipFile(t *testing.T) {
	t.Run("given a matching generated file pattern, it excludes the file", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		filter, err := gomut.NewExclusionFilter(root, []string{"*_mock.go"})
		require.NoError(t, err)

		// Act
		excluded, reason := filter.SkipFile(filepath.ToSlash(filepath.Join("internal", "sample_mock.go")))

		// Assert
		assert.True(t, excluded)
		assert.NotEmpty(t, reason)
	})

	t.Run("given a non-matching file, it keeps the file", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		filter, err := gomut.NewExclusionFilter(root, []string{"*_mock.go"})
		require.NoError(t, err)

		// Act
		excluded, reason := filter.SkipFile(filepath.ToSlash(filepath.Join("internal", "sample.go")))

		// Assert
		assert.False(t, excluded)
		assert.Empty(t, reason)
	})
}

func TestExclusionFilterSkipCandidate(t *testing.T) {
	t.Run("given a function-level ignore comment, it excludes candidates inside the function", func(t *testing.T) {
		assertSkipCandidateFixture(t, `package sample

//gomut:ignore
func Ignored(value int) int {
	return value + 1
}

func Kept(value int) int {
	return value + 1
}
`, 5, 9)
	})

	t.Run("given a statement-level ignore comment, it excludes the annotated line or block", func(t *testing.T) {
		assertSkipCandidateFixture(t, `package sample

func BlockIgnored(value int) int {
	//gomut:ignore
	return value + 1
}

func Kept(value int) int {
	return value + 1
}
`, 5, 9)
	})
}

func assertSkipCandidateFixture(t *testing.T, source string, ignoredLine, keptLine int) {
	t.Helper()

	// Arrange
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/mut\n\ngo 1.26\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sample"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sample", "sample.go"), []byte(source), 0o600))
	filter, err := gomut.NewExclusionFilter(root, nil)
	require.NoError(t, err)

	ignoredCandidate := gomut.Candidate{
		File: filepath.ToSlash(filepath.Join("sample", "sample.go")),
		Line: ignoredLine,
		Kind: gomut.MutationKindArithmeticOperator,
	}
	keptCandidate := gomut.Candidate{
		File: filepath.ToSlash(filepath.Join("sample", "sample.go")),
		Line: keptLine,
		Kind: gomut.MutationKindArithmeticOperator,
	}

	// Act
	ignored, ignoredReason := filter.SkipCandidate(ignoredCandidate)
	kept, keptReason := filter.SkipCandidate(keptCandidate)

	// Assert
	assert.True(t, ignored)
	assert.NotEmpty(t, ignoredReason)
	assert.False(t, kept)
	assert.Empty(t, keptReason)
}
