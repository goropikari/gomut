package gomut_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/goropikari/gomut/internal/gomut/result"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomut "github.com/goropikari/gomut/internal/gomut"
)

func TestResolveTarget(t *testing.T) {
	t.Run("given a package target, it returns package mode", func(t *testing.T) {
		// Arrange
		target, err := gomut.ResolveTarget("./internal/foo", "")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, result.Target{Mode: result.TargetModePackage, Value: "./internal/foo"}, target)
	})

	t.Run("given ./... as the target, it returns package mode", func(t *testing.T) {
		// Arrange
		target, err := gomut.ResolveTarget("./...", "")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, result.Target{Mode: result.TargetModePackage, Value: "./..."}, target)
	})

	t.Run("given a diff range, it returns diff mode", func(t *testing.T) {
		// Arrange
		target, err := gomut.ResolveTarget("", "HEAD~1..HEAD")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, result.Target{Mode: result.TargetModeDiff, Value: "HEAD~1..HEAD"}, target)
	})

	t.Run("given no target, it returns a helpful error", func(t *testing.T) {
		// Arrange
		target, err := gomut.ResolveTarget("", "")

		// Assert
		require.Error(t, err)
		assert.Empty(t, target)
		assert.Contains(t, err.Error(), "gomut test ./...")
	})
}

func TestParseDiffPatch(t *testing.T) {
	t.Run("given a single added file hunk, it records the file and changed lines", func(t *testing.T) {
		// Arrange
		patch := `diff --git a/foo.go b/foo.go
index 0000000..1111111 100644
--- a/foo.go
+++ b/foo.go
@@ -10,0 +11,2 @@
+x
+y
`

		// Act
		files, err := gomut.ParseDiffPatch(patch)

		// Assert
		require.NoError(t, err)
		require.Len(t, files, 1)
		assert.Equal(t, "foo.go", files[0])
		assert.True(t, gomut.DiffLineAllowed("foo.go", 11))
		assert.True(t, gomut.DiffLineAllowed("foo.go", 12))
		assert.False(t, gomut.DiffLineAllowed("foo.go", 9))
	})
}

func TestApplyMutation(t *testing.T) {
	t.Run("given a replacement range, it rewrites the file contents", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		file := filepath.Join(dir, "sample.go")
		require.NoError(t, os.WriteFile(file, []byte("package sample\n\nfunc add() int { return 1 + 2 }\n"), 0o600))

		candidate := result.Candidate{
			File:        "sample.go",
			Start:       42,
			End:         43,
			Replacement: "-",
		}

		// Act
		out, err := gomut.ApplyMutation(dir, candidate)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "package sample\n\nfunc add() int { return 1 - 2 }\n", string(out))
	})
}

func TestNormalizeTestArgs(t *testing.T) {
	t.Run("given jsonl without a value, it keeps stdout output", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "--jsonl"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.True(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
		assert.Empty(t, sarifOutput)
		assert.False(t, sarifEnabled)
	})

	t.Run("given jsonl with a value, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "--jsonl", "mutations.jsonl"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.JSONEq(t, `"mutations.jsonl"`, strconv.Quote(jsonlOutput))
		assert.True(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
		assert.Empty(t, sarifOutput)
		assert.False(t, sarifEnabled)
	})

	t.Run("given jsonl equals syntax, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "--jsonl=mutations.jsonl"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.JSONEq(t, `"mutations.jsonl"`, strconv.Quote(jsonlOutput))
		assert.True(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
		assert.Empty(t, sarifOutput)
		assert.False(t, sarifEnabled)
	})

	t.Run("given short jsonl syntax, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "-jsonl", "mutations.jsonl"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.JSONEq(t, `"mutations.jsonl"`, strconv.Quote(jsonlOutput))
		assert.True(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
		assert.Empty(t, sarifOutput)
		assert.False(t, sarifEnabled)
	})

	t.Run("given html without a value, it keeps stdout output", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "--html"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.False(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.True(t, htmlEnabled)
		assert.Empty(t, sarifOutput)
		assert.False(t, sarifEnabled)
	})

	t.Run("given html with a value, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "--html", "report.html"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.False(t, jsonlEnabled)
		assert.Equal(t, "report.html", htmlOutput)
		assert.True(t, htmlEnabled)
		assert.Empty(t, sarifOutput)
		assert.False(t, sarifEnabled)
	})

	t.Run("given short html syntax, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "-html", "report.html"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.False(t, jsonlEnabled)
		assert.Equal(t, "report.html", htmlOutput)
		assert.True(t, htmlEnabled)
		assert.Empty(t, sarifOutput)
		assert.False(t, sarifEnabled)
	})

	t.Run("given both jsonl and html outputs, it captures both file paths", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "--jsonl", "mutations.jsonl", "--html", "report.html"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.JSONEq(t, `"mutations.jsonl"`, strconv.Quote(jsonlOutput))
		assert.True(t, jsonlEnabled)
		assert.Equal(t, "report.html", htmlOutput)
		assert.True(t, htmlEnabled)
		assert.Empty(t, sarifOutput)
		assert.False(t, sarifEnabled)
	})

	t.Run("given sarif without a value, it keeps stdout output", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "--sarif"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.False(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
		assert.Empty(t, sarifOutput)
		assert.True(t, sarifEnabled)
	})

	t.Run("given sarif with a value, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "--sarif", "report.sarif"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.False(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
		assert.Equal(t, "report.sarif", sarifOutput)
		assert.True(t, sarifEnabled)
	})

	t.Run("given short sarif syntax, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "-sarif", "report.sarif"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.False(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
		assert.Equal(t, "report.sarif", sarifOutput)
		assert.True(t, sarifEnabled)
	})

	t.Run("given sarif equals syntax, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "--sarif=report.sarif"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.False(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
		assert.Equal(t, "report.sarif", sarifOutput)
		assert.True(t, sarifEnabled)
	})

	t.Run("given short sarif equals syntax, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := gomut.NormalizeTestArgs([]string{"./internal/gomut", "-sarif=report.sarif"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.False(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
		assert.Equal(t, "report.sarif", sarifOutput)
		assert.True(t, sarifEnabled)
	})
}

func TestRecordJSONIncludesMutationReplacementDetails(t *testing.T) {
	t.Run("given a mutation record, it serializes original and replacement", func(t *testing.T) {
		// Arrange
		record := result.Record{
			Target: result.Target{Mode: result.TargetModePackage, Value: "./sample"},
			Mutation: result.MutationMetadata{
				File:        "sample.go",
				Line:        18,
				Kind:        result.MutationKindLogicalOperator,
				Original:    "&&",
				Replacement: "||",
				Result:      result.MutationResultLived,
				Message:     "ok",
			},
		}

		// Act
		data, err := json.Marshal(record)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `{"target":{"mode":"package","value":"./sample"},"started_at":"","command":"","summary":{"total":0,"killed":0,"lived":0,"not_covered":0,"timed_out":0,"not_viable":0},"mutation":{"file":"sample.go","line":18,"kind":"logical_operator","original":"&&","replacement":"||","result":"LIVED","message":"ok"}}`, string(data))
	})

	t.Run("given a control flow mutation record, it serializes the control_flow kind", func(t *testing.T) {
		// Arrange
		record := result.Record{
			Target: result.Target{Mode: result.TargetModePackage, Value: "./sample"},
			Mutation: result.MutationMetadata{
				File:        "sample.go",
				Line:        12,
				Kind:        result.MutationKindControlFlow,
				Original:    "ready",
				Replacement: "!ready",
				Result:      result.MutationResultKilled,
				Message:     "killed by tests",
			},
		}

		// Act
		data, err := json.Marshal(record)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `{"target":{"mode":"package","value":"./sample"},"started_at":"","command":"","summary":{"total":0,"killed":0,"lived":0,"not_covered":0,"timed_out":0,"not_viable":0},"mutation":{"file":"sample.go","line":12,"kind":"control_flow","original":"ready","replacement":"!ready","result":"KILLED","message":"killed by tests"}}`, string(data))
	})

	t.Run("given a loop control mutation record, it serializes the loop_control kind", func(t *testing.T) {
		// Arrange
		record := result.Record{
			Target: result.Target{Mode: result.TargetModePackage, Value: "./sample"},
			Mutation: result.MutationMetadata{
				File:        "sample.go",
				Line:        24,
				Kind:        result.MutationKindLoopControl,
				Original:    "break",
				Replacement: "continue",
				Result:      result.MutationResultLived,
				Message:     "ok",
			},
		}

		// Act
		data, err := json.Marshal(record)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `{"target":{"mode":"package","value":"./sample"},"started_at":"","command":"","summary":{"total":0,"killed":0,"lived":0,"not_covered":0,"timed_out":0,"not_viable":0},"mutation":{"file":"sample.go","line":24,"kind":"loop_control","original":"break","replacement":"continue","result":"LIVED","message":"ok"}}`, string(data))
	})

	t.Run("given an assignment arithmetic mutation record, it serializes the assignment_arithmetic kind", func(t *testing.T) {
		// Arrange
		record := result.Record{
			Target: result.Target{Mode: result.TargetModePackage, Value: "./sample"},
			Mutation: result.MutationMetadata{
				File:        "sample.go",
				Line:        52,
				Kind:        result.MutationKindAssignmentArithmetic,
				Original:    "+=",
				Replacement: "-=",
				Result:      result.MutationResultLived,
				Message:     "ok",
			},
		}

		// Act
		data, err := json.Marshal(record)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `{"target":{"mode":"package","value":"./sample"},"started_at":"","command":"","summary":{"total":0,"killed":0,"lived":0,"not_covered":0,"timed_out":0,"not_viable":0},"mutation":{"file":"sample.go","line":52,"kind":"assignment_arithmetic","original":"+=","replacement":"-=","result":"LIVED","message":"ok"}}`, string(data))
	})

	t.Run("given an inc/dec mutation record, it serializes the inc_dec kind", func(t *testing.T) {
		// Arrange
		record := result.Record{
			Target: result.Target{Mode: result.TargetModePackage, Value: "./sample"},
			Mutation: result.MutationMetadata{
				File:        "sample.go",
				Line:        58,
				Kind:        result.MutationKindIncDec,
				Original:    "++",
				Replacement: "--",
				Result:      result.MutationResultLived,
				Message:     "ok",
			},
		}

		// Act
		data, err := json.Marshal(record)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `{"target":{"mode":"package","value":"./sample"},"started_at":"","command":"","summary":{"total":0,"killed":0,"lived":0,"not_covered":0,"timed_out":0,"not_viable":0},"mutation":{"file":"sample.go","line":58,"kind":"inc_dec","original":"++","replacement":"--","result":"LIVED","message":"ok"}}`, string(data))
	})

	t.Run("given a return mutation record, it serializes the return kind", func(t *testing.T) {
		// Arrange
		record := result.Record{
			Target: result.Target{Mode: result.TargetModePackage, Value: "./sample"},
			Mutation: result.MutationMetadata{
				File:        "sample.go",
				Line:        22,
				Kind:        result.MutationKindReturn,
				Original:    "true",
				Replacement: "false",
				Result:      result.MutationResultLived,
				Message:     "ok",
			},
		}

		// Act
		data, err := json.Marshal(record)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `{"target":{"mode":"package","value":"./sample"},"started_at":"","command":"","summary":{"total":0,"killed":0,"lived":0,"not_covered":0,"timed_out":0,"not_viable":0},"mutation":{"file":"sample.go","line":22,"kind":"return","original":"true","replacement":"false","result":"LIVED","message":"ok"}}`, string(data))
	})

	t.Run("given a nil check mutation record, it serializes the nil_check kind", func(t *testing.T) {
		// Arrange
		record := result.Record{
			Target: result.Target{Mode: result.TargetModePackage, Value: "./sample"},
			Mutation: result.MutationMetadata{
				File:        "sample.go",
				Line:        31,
				Kind:        result.MutationKindNilCheck,
				Original:    "!=",
				Replacement: "==",
				Result:      result.MutationResultLived,
				Message:     "ok",
			},
		}

		// Act
		data, err := json.Marshal(record)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `{"target":{"mode":"package","value":"./sample"},"started_at":"","command":"","summary":{"total":0,"killed":0,"lived":0,"not_covered":0,"timed_out":0,"not_viable":0},"mutation":{"file":"sample.go","line":31,"kind":"nil_check","original":"!=","replacement":"==","result":"LIVED","message":"ok"}}`, string(data))
	})

	t.Run("given a boolean literal mutation record, it serializes the boolean_literal kind", func(t *testing.T) {
		// Arrange
		record := result.Record{
			Target: result.Target{Mode: result.TargetModePackage, Value: "./sample"},
			Mutation: result.MutationMetadata{
				File:        "sample.go",
				Line:        40,
				Kind:        result.MutationKindBooleanLiteral,
				Original:    "true",
				Replacement: "false",
				Result:      result.MutationResultLived,
				Message:     "ok",
			},
		}

		// Act
		data, err := json.Marshal(record)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `{"target":{"mode":"package","value":"./sample"},"started_at":"","command":"","summary":{"total":0,"killed":0,"lived":0,"not_covered":0,"timed_out":0,"not_viable":0},"mutation":{"file":"sample.go","line":40,"kind":"boolean_literal","original":"true","replacement":"false","result":"LIVED","message":"ok"}}`, string(data))
	})
}
