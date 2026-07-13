package report_test

import (
	"bytes"
	"gomut/internal/gomut/report"
	"gomut/internal/gomut/result"
	"html"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteHTML(t *testing.T) {
	t.Run("given mutation records, it renders a self-contained html report", func(t *testing.T) {
		// Arrange
		var output bytes.Buffer

		root := t.TempDir()
		source := "package sample\n\nfunc Example(v int) int {\n\tif v == 10 {\n\t\treturn v + 0\n\t}\n\n\treturn v\n}\n"
		filePath := filepath.Join(root, "sample.go")

		err := os.WriteFile(filePath, []byte(source), 0o600)
		require.NoError(t, err)

		comparisonStart := strings.Index(source, "==")
		require.NotEqual(t, -1, comparisonStart)

		records := []result.Record{
			{
				Target:    result.Target{Mode: result.TargetModePackage, Value: "./sample"},
				StartedAt: "2026-07-12T01:02:03Z",
				Command:   "gomut test --package ./sample --html",
				Summary: result.Summary{
					Total:      2,
					Killed:     1,
					Lived:      1,
					NotCovered: 0,
					TimedOut:   0,
					NotViable:  0,
				},
				Mutation: result.MutationMetadata{
					File:        "sample.go",
					Line:        4,
					Kind:        result.MutationKindComparisonOperator,
					Original:    "==",
					Replacement: "!=",
					Result:      result.MutationResultKilled,
					Message:     "killed by tests",
					Start:       comparisonStart,
					End:         comparisonStart + len("=="),
				},
			},
		}

		// Act
		err = report.WriteHTML(&output, report.HTMLReportData{
			Root:      root,
			Target:    records[0].Target,
			StartedAt: records[0].StartedAt,
			Command:   records[0].Command,
			Summary:   records[0].Summary,
			Records:   records,
		})

		// Assert
		require.NoError(t, err)

		rendered := html.UnescapeString(output.String())
		assert.Contains(t, rendered, "<!doctype html")
		assert.Contains(t, rendered, "2026-07-12T01:02:03Z")
		assert.Contains(t, rendered, "gomut test --package ./sample --html")
		assert.Contains(t, rendered, "sample.go")
		assert.Contains(t, rendered, "comparison_operator")
		assert.Contains(t, rendered, "killed by tests")
		assert.Contains(t, rendered, "KILLED")
		assert.Contains(t, rendered, "Source excerpt")
		assert.Contains(t, rendered, "Unified diff")
		assert.Contains(t, rendered, "func Example(v int) int {")
		assert.Contains(t, rendered, "if v == 10 {")
		assert.Contains(t, rendered, "return v + 0")
		assert.Contains(t, rendered, "--- a/sample.go")
		assert.Contains(t, rendered, "if v != 10 {")
	})
}
