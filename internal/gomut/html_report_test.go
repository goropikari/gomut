package gomut_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomut "gomut/internal/gomut"
)

func TestWriteHTML(t *testing.T) {
	t.Run("given mutation records, it renders a self-contained html report", func(t *testing.T) {
		// Arrange
		var output bytes.Buffer

		records := []gomut.Record{
			{
				Target:    gomut.Target{Mode: gomut.TargetModePackage, Value: "./sample"},
				StartedAt: "2026-07-12T01:02:03Z",
				Command:   "gomut test --package ./sample --html",
				Summary: gomut.Summary{
					Total:      2,
					Killed:     1,
					Lived:      1,
					NotCovered: 0,
					TimedOut:   0,
					NotViable:  0,
				},
				Mutation: gomut.MutationMetadata{
					File:        "sample.go",
					Line:        18,
					Kind:        gomut.MutationKindComparisonOperator,
					Original:    "==",
					Replacement: "!=",
					Result:      gomut.MutationResultKilled,
					Message:     "killed by tests",
				},
			},
			{
				Target:    gomut.Target{Mode: gomut.TargetModePackage, Value: "./sample"},
				StartedAt: "2026-07-12T01:02:03Z",
				Command:   "gomut test --package ./sample --html",
				Summary: gomut.Summary{
					Total:      2,
					Killed:     1,
					Lived:      1,
					NotCovered: 0,
					TimedOut:   0,
					NotViable:  0,
				},
				Mutation: gomut.MutationMetadata{
					File:        "sample.go",
					Line:        24,
					Kind:        gomut.MutationKindLogicalOperator,
					Original:    "&&",
					Replacement: "||",
					Result:      gomut.MutationResultLived,
					Message:     "survived",
				},
			},
		}

		// Act
		err := gomut.WriteHTML(&output, gomut.HTMLReportData{
			Target:    records[0].Target,
			StartedAt: records[0].StartedAt,
			Command:   records[0].Command,
			Summary:   records[0].Summary,
			Records:   records,
		})

		// Assert
		require.NoError(t, err)

		rendered := output.String()
		assert.Contains(t, rendered, "<!doctype html")
		assert.Contains(t, rendered, "2026-07-12T01:02:03Z")
		assert.Contains(t, rendered, "gomut test --package ./sample --html")
		assert.Contains(t, rendered, "sample.go")
		assert.Contains(t, rendered, "comparison_operator")
		assert.Contains(t, rendered, "logical_operator")
		assert.Contains(t, rendered, "killed by tests")
		assert.Contains(t, rendered, "survived")
		assert.Contains(t, rendered, "KILLED")
		assert.Contains(t, rendered, "LIVED")
	})
}
