package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRunSARIFOutput(t *testing.T) {
	t.Run("given sarif output without a path, it writes sarif to stdout", func(t *testing.T) {
		// Arrange
		root := createSarifFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./sample", "--sarif", "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr, "Mutation summary")

		log := decodeSARIFLog(t, stdout)
		require.Len(t, log.Runs, 1)
		require.NotEmpty(t, log.Runs[0].Results)

		first := log.Runs[0].Results[0]
		assert.NotEmpty(t, first.RuleID)
		assert.NotEmpty(t, first.Level)
		assert.NotEmpty(t, first.Locations)
		assert.Equal(t, "sample/sample.go", first.Locations[0].PhysicalLocation.ArtifactLocation.URI)
		assert.Positive(t, first.Locations[0].PhysicalLocation.Region.StartLine)
	})

	t.Run("given sarif output with a file path, it writes the report to that file", func(t *testing.T) {
		// Arrange
		root := createSarifFixture(t)
		sarifPath := filepath.Join(root, "report.sarif")

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "./sample", "--sarif", sarifPath, "--progress=off"})

		// Assert
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Mutation summary")

		data, readErr := os.ReadFile(sarifPath)
		require.NoError(t, readErr)
		log := decodeSARIFLog(t, string(data))
		require.Len(t, log.Runs, 1)
		require.NotEmpty(t, log.Runs[0].Results)
		assert.Equal(t, "2.1.0", log.Version)
	})
}

type sarifLogFixture struct {
	Version string            `json:"version"`
	Runs    []sarifRunFixture `json:"runs"`
}

type sarifRunFixture struct {
	Results []sarifResultFixture `json:"results"`
}

type sarifResultFixture struct {
	RuleID    string                 `json:"ruleId"`
	Level     string                 `json:"level"`
	Locations []sarifLocationFixture `json:"locations"`
}

type sarifLocationFixture struct {
	PhysicalLocation sarifPhysicalLocationFixture `json:"physicalLocation"`
}

type sarifPhysicalLocationFixture struct {
	ArtifactLocation sarifArtifactLocationFixture `json:"artifactLocation"`
	Region           sarifRegionFixture           `json:"region"`
}

type sarifArtifactLocationFixture struct {
	URI string `json:"uri"`
}

type sarifRegionFixture struct {
	StartLine int `json:"startLine"`
}

func decodeSARIFLog(t *testing.T, output string) sarifLogFixture {
	t.Helper()

	var log sarifLogFixture
	require.NoError(t, json.Unmarshal([]byte(output), &log))

	return log
}

func createSarifFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/sariftest\n\ngo 1.26\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sample"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(root, "sample", "sample.go"), []byte(`package sample

func AboveThreshold(value int) bool {
	return value > 10
}
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sample", "sample_test.go"), []byte(`package sample_test

import (
	"testing"

	"example.com/sariftest/sample"
)

func TestAboveThreshold(t *testing.T) {
	if got := sample.AboveThreshold(11); !got {
		t.Fatal("expected value above threshold to be accepted")
	}
}
`), 0o600))

	return root
}
