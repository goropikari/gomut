package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRunHTMLOutput(t *testing.T) {
	t.Run("given html output without a path, it writes html to stdout", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--html"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, strings.ToLower(stdout), "<!doctype html")
		assert.Contains(t, strings.ToLower(stdout), "<html")
		assert.Contains(t, stderr, "Mutation summary")
	})

	t.Run("given html output with a file path, it writes the report to that file", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)
		htmlPath := filepath.Join(root, "report.html")

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--html", htmlPath})

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Mutation summary")

		data, readErr := os.ReadFile(htmlPath)
		require.NoError(t, readErr)
		assert.Contains(t, strings.ToLower(string(data)), "<!doctype html")
		assert.Contains(t, string(data), "sample.go")
	})

	t.Run("given jsonl and html output paths, it writes both outputs", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)
		jsonlPath := filepath.Join(root, "mutations.jsonl")
		htmlPath := filepath.Join(root, "report.html")

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--jsonl", jsonlPath, "--html", htmlPath})

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Mutation summary")

		jsonlData, readJSONLErr := os.ReadFile(jsonlPath)
		require.NoError(t, readJSONLErr)
		assert.NotEmpty(t, decodeJSONLRecords(t, string(jsonlData)))

		htmlData, readHTMLErr := os.ReadFile(htmlPath)
		require.NoError(t, readHTMLErr)
		assert.Contains(t, strings.ToLower(string(htmlData)), "<html")
	})

	t.Run("given stdout jsonl and html output path, it writes jsonl to stdout and html to the file", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)
		htmlPath := filepath.Join(root, "report.html")

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--jsonl", "--html", htmlPath})

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, stdout)
		assert.Contains(t, stderr, "Mutation summary")

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		data, readErr := os.ReadFile(htmlPath)
		require.NoError(t, readErr)
		assert.Contains(t, strings.ToLower(string(data)), "<html")
	})

	t.Run("given a type filter and html output, it reflects the filtered results", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)
		htmlPath := filepath.Join(root, "filtered-report.html")

		// Act
		_, _, err := runCommandInDir(t, root, []string{"./sample", "--type", "lived", "--html", htmlPath})

		// Assert
		require.NoError(t, err)

		data, readErr := os.ReadFile(htmlPath)
		require.NoError(t, readErr)
		assert.Contains(t, strings.ToLower(string(data)), "<html")
		assert.Contains(t, string(data), "LIVED")
	})

	t.Run("given an invalid html output path, it fails before creating the report", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)
		htmlPath := filepath.Join(root, "missing", "report.html")

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--html", htmlPath})

		// Assert
		require.Error(t, err)
		assert.Empty(t, stdout)
		assert.NotEmpty(t, stderr)

		_, statErr := os.Stat(htmlPath)
		assert.Error(t, statErr)
	})
}

func TestCommandRunRejectsUnexpectedArguments(t *testing.T) {
	t.Run("given an extra positional argument, it fails before running mutations", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "unexpected"})

		// Assert
		require.Error(t, err)
		assert.Empty(t, stdout)
		assert.Empty(t, stderr)
		assert.Contains(t, err.Error(), "unexpected arguments")
		assert.Contains(t, err.Error(), "unexpected")
	})
}
