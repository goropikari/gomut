package gomut_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/goropikari/gomut/internal/gomut/result"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomut "github.com/goropikari/gomut/internal/gomut"
)

func TestNormalizeDiffRange(t *testing.T) {
	t.Run("given a branch name, it expands to a triple-dot diff against HEAD", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		initGitRepo(t, root)

		writeGitFile(t, root, "base.go", "package sample\n\nfunc base() bool { return true }\n")
		runGit(t, root, "add", "base.go")
		runGit(t, root, "commit", "-m", "initial")
		runGit(t, root, "branch", "-M", "main")

		runGit(t, root, "checkout", "-b", "feature")
		writeGitFile(t, root, "feature.go", "package sample\n\nfunc feature() bool { return true }\n")
		runGit(t, root, "add", "feature.go")
		runGit(t, root, "commit", "-m", "feature change")

		runGit(t, root, "checkout", "main")
		writeGitFile(t, root, "main_only.go", "package sample\n\nfunc mainOnly() bool { return true }\n")
		runGit(t, root, "add", "main_only.go")
		runGit(t, root, "commit", "-m", "main change")
		runGit(t, root, "checkout", "feature")

		// Act
		got, err := gomut.NormalizeDiffRange(context.Background(), root, "main")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "main...HEAD", got)
	})

	t.Run("given an explicit range, it preserves the value", func(t *testing.T) {
		// Arrange
		got, err := gomut.NormalizeDiffRange(context.Background(), t.TempDir(), "HEAD~1..HEAD")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "HEAD~1..HEAD", got)
	})
}

func TestDiffFiles(t *testing.T) {
	t.Run("given a diff range with a nested go file, it returns the changed file", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		initGitRepo(t, root)

		require.NoError(t, os.MkdirAll(filepath.Join(root, "sample"), 0o755))
		writeGitFile(t, root, "go.mod", "module example.com/mut\n\ngo 1.26\n")
		writeGitFile(t, root, filepath.Join("sample", "sample.go"), "package sample\n\nfunc IsAdult(age int) bool {\n\tif age < 18 {\n\t\treturn false\n\t}\n\n\treturn true\n}\n")
		runGit(t, root, "add", "go.mod", filepath.Join("sample", "sample.go"))
		runGit(t, root, "commit", "-m", "initial")

		writeGitFile(t, root, filepath.Join("sample", "sample.go"), "package sample\n\nfunc IsAdult(age int) bool {\n\tif age < 21 {\n\t\treturn false\n\t}\n\n\treturn true\n}\n")
		runGit(t, root, "add", filepath.Join("sample", "sample.go"))
		runGit(t, root, "commit", "-m", "update sample")

		// Act
		files, err := gomut.DiffFiles(context.Background(), root, "HEAD~1..HEAD")

		// Assert
		require.NoError(t, err)
		require.Len(t, files, 1)
		assert.Equal(t, filepath.ToSlash(filepath.Join("sample", "sample.go")), files[0])
		assert.True(t, gomut.DiffLineAllowed(filepath.Join("sample", "sample.go"), 4))
	})
}

func TestDiscoverCandidatesWithDiffTarget(t *testing.T) {
	t.Run("given a changed nested file, it keeps candidates on changed lines", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		initGitRepo(t, root)

		require.NoError(t, os.MkdirAll(filepath.Join(root, "sample"), 0o755))
		writeGitFile(t, root, "go.mod", "module example.com/mut\n\ngo 1.26\n")
		writeGitFile(t, root, filepath.Join("sample", "sample.go"), "package sample\n\nfunc IsAdult(age int) bool {\n\tif age < 18 {\n\t\treturn false\n\t}\n\n\treturn true\n}\n")
		runGit(t, root, "add", "go.mod", filepath.Join("sample", "sample.go"))
		runGit(t, root, "commit", "-m", "initial")

		writeGitFile(t, root, filepath.Join("sample", "sample.go"), "package sample\n\nfunc IsAdult(age int) bool {\n\tif age < 21 {\n\t\treturn false\n\t}\n\n\treturn true\n}\n")
		runGit(t, root, "add", filepath.Join("sample", "sample.go"))
		runGit(t, root, "commit", "-m", "update sample")

		files, err := gomut.DiffFiles(context.Background(), root, "HEAD~1..HEAD")
		require.NoError(t, err)

		// Act
		candidates, err := gomut.DiscoverCandidates(root, []string{"example.com/mut/sample"}, result.Target{Mode: result.TargetModeDiff, Value: "HEAD~1..HEAD"}, map[string]result.FileCoverage{})

		// Assert
		require.NoError(t, err)
		require.NotEmpty(t, candidates)
		assert.Equal(t, filepath.ToSlash(filepath.Join("sample", "sample.go")), files[0])
	})
}

func initGitRepo(t *testing.T, root string) {
	t.Helper()

	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "Test User")
	runGit(t, root, "config", "user.email", "test@example.com")
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = root

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func writeGitFile(t *testing.T, root, name, contents string) {
	t.Helper()

	path := filepath.Join(root, name)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
}
