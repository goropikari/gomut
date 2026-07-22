package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goropikari/gomut/internal/gomut/result"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/goropikari/gomut/internal/gomut"
)

func TestCommandRunTypeFilter(t *testing.T) {
	t.Run("given a single result type, it writes only matching records", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--type", "not-covered"})

		// Assert
		require.NoError(t, err)

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		for _, record := range records {
			assert.Equal(t, result.MutationResultNotCovered, record.Mutation.Result)
		}

		last := records[len(records)-1]
		assert.Equal(t, len(records), last.Summary.Total)
		assert.Equal(t, len(records), last.Summary.NotCovered)
		assert.Zero(t, last.Summary.Killed)
		assert.Zero(t, last.Summary.Lived)
		assert.Zero(t, last.Summary.TimedOut)
		assert.Zero(t, last.Summary.NotViable)
		assert.Contains(t, stderr, "Mutation summary")
		assert.Contains(t, stderr, "  killed: 0")
		assert.Contains(t, stderr, "  lived: 0")
		assert.Contains(t, stderr, "  not covered: ")
		assert.Contains(t, stderr, "  timed out: 0")
		assert.Contains(t, stderr, "  not viable: 0")
	})

	t.Run("given comma-separated result types, it writes only matching records", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--type", "killed,lived"})

		// Assert
		require.NoError(t, err)

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		for _, record := range records {
			assert.Contains(t, []result.MutationResult{result.MutationResultKilled, result.MutationResultLived}, record.Mutation.Result)
		}

		last := records[len(records)-1]
		assert.Equal(t, len(records), last.Summary.Total)
		assert.Zero(t, last.Summary.NotCovered)
		assert.Zero(t, last.Summary.TimedOut)
		assert.Zero(t, last.Summary.NotViable)
		assert.Contains(t, stderr, "Mutation summary")
		assert.Contains(t, stderr, "  killed: ")
		assert.Contains(t, stderr, "  lived: ")
		assert.Contains(t, stderr, "  not covered: 0")
		assert.Contains(t, stderr, "  timed out: 0")
		assert.Contains(t, stderr, "  not viable: 0")
	})

	t.Run("given repeated result type flags, it writes only matching records", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--type", "killed", "--type", "timed-out"})

		// Assert
		require.NoError(t, err)

		records := decodeJSONLRecords(t, stdout)
		require.NotEmpty(t, records)

		for _, record := range records {
			assert.Contains(t, []result.MutationResult{result.MutationResultKilled, result.MutationResultTimedOut}, record.Mutation.Result)
		}

		last := records[len(records)-1]
		assert.Equal(t, len(records), last.Summary.Total)
		assert.Zero(t, last.Summary.Lived)
		assert.Zero(t, last.Summary.NotCovered)
		assert.Zero(t, last.Summary.NotViable)
		assert.Contains(t, stderr, "Mutation summary")
		assert.Contains(t, stderr, "  killed: ")
		assert.Contains(t, stderr, "  lived: 0")
		assert.Contains(t, stderr, "  not covered: 0")
		assert.Contains(t, stderr, "  not viable: 0")
	})

	t.Run("given an unknown result type, it fails before writing output", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"./sample", "--type", "unknown"})

		// Assert
		require.Error(t, err)
		assert.Empty(t, stdout)
		assert.Empty(t, stderr)
		assert.Contains(t, err.Error(), "unknown")
	})
}

func createResultFilterFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/filtertest\n\ngo 1.26\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sample"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(root, "sample", "sample.go"), []byte(`package sample

func IsAtLeast(age int) bool {
	return age >= 18
}

func Add(a, b int) int {
	return a + b
}

func KeepCommonBits(mask, flag uint8) uint8 {
	return mask & flag
}

func Greeting() string {
	return "hello"
}

func SumPositive(values []int) int {
	total := 0

	for _, value := range values {
		if value < 0 {
			break
		}

		if value == 0 {
			continue
		}

		total += value
	}

	return total
}

func Unused() bool {
	return true
}
`), 0o600))

	require.NoError(t, os.WriteFile(filepath.Join(root, "sample", "sample_test.go"), []byte(`package sample_test

import (
	"testing"

	"example.com/filtertest/sample"
)

func TestIsAtLeast(t *testing.T) {
	if !sample.IsAtLeast(20) {
		t.Fatal("expected adult input to be accepted")
	}
}

func TestAdd(t *testing.T) {
	if got := sample.Add(1, 2); got != 3 {
		t.Fatalf("expected sum to be 3, got %d", got)
	}
}

func TestKeepCommonBits(t *testing.T) {
	if got := sample.KeepCommonBits(0b1101, 0b1011); got != 0b1001 {
		t.Fatalf("expected common bits to be kept, got %08b", got)
	}
}

func TestGreeting(t *testing.T) {
	if got := sample.Greeting(); got != "hello" {
		t.Fatalf("expected greeting to be hello, got %q", got)
	}
}

func TestSumPositive(t *testing.T) {
	if got := sample.SumPositive([]int{1, 0, 2, -1, 5}); got != 3 {
		t.Fatalf("expected filtered sum to be 3, got %d", got)
	}
}
`), 0o600))

	return root
}

func runCommandInDir(t *testing.T, dir string, args []string) (string, string, error) {
	t.Helper()

	t.Chdir(dir)

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd := gomut.NewCommand(&stdout, &stderr)
	err := cmd.Run(context.Background(), args)

	return stdout.String(), stderr.String(), err
}

func decodeJSONLRecords(t *testing.T, output string) []result.Record {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	records := make([]result.Record, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		var record result.Record
		require.NoError(t, json.Unmarshal([]byte(line), &record))
		records = append(records, record)
	}

	return records
}
