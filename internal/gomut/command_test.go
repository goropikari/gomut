package gomut_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomut "gomut/internal/gomut"
)

func TestResolveTarget(t *testing.T) {
	t.Run("given a package target, it returns package mode", func(t *testing.T) {
		// Arrange
		target, err := gomut.ResolveTarget("./internal/foo", false, "")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, gomut.Target{Mode: gomut.TargetModePackage, Value: "./internal/foo"}, target)
	})

	t.Run("given the all flag, it returns all mode", func(t *testing.T) {
		// Arrange
		target, err := gomut.ResolveTarget("", true, "")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, gomut.Target{Mode: gomut.TargetModeAll, Value: "./..."}, target)
	})

	t.Run("given a diff range, it returns diff mode", func(t *testing.T) {
		// Arrange
		target, err := gomut.ResolveTarget("", false, "HEAD~1..HEAD")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, gomut.Target{Mode: gomut.TargetModeDiff, Value: "HEAD~1..HEAD"}, target)
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

		candidate := gomut.Candidate{
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
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, err := gomut.NormalizeTestArgs([]string{"--package", "./internal/gomut", "--jsonl"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"--package", "./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.True(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
	})

	t.Run("given jsonl with a value, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, err := gomut.NormalizeTestArgs([]string{"--package", "./internal/gomut", "--jsonl", "mutations.jsonl"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"--package", "./internal/gomut"}, args)
		assert.JSONEq(t, `"mutations.jsonl"`, strconv.Quote(jsonlOutput))
		assert.True(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
	})

	t.Run("given jsonl equals syntax, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, err := gomut.NormalizeTestArgs([]string{"--package", "./internal/gomut", "--jsonl=mutations.jsonl"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"--package", "./internal/gomut"}, args)
		assert.JSONEq(t, `"mutations.jsonl"`, strconv.Quote(jsonlOutput))
		assert.True(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.False(t, htmlEnabled)
	})

	t.Run("given html without a value, it keeps stdout output", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, err := gomut.NormalizeTestArgs([]string{"--package", "./internal/gomut", "--html"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"--package", "./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.False(t, jsonlEnabled)
		assert.Empty(t, htmlOutput)
		assert.True(t, htmlEnabled)
	})

	t.Run("given html with a value, it captures the file path", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, err := gomut.NormalizeTestArgs([]string{"--package", "./internal/gomut", "--html", "report.html"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"--package", "./internal/gomut"}, args)
		assert.Empty(t, jsonlOutput)
		assert.False(t, jsonlEnabled)
		assert.Equal(t, "report.html", htmlOutput)
		assert.True(t, htmlEnabled)
	})

	t.Run("given both jsonl and html outputs, it captures both file paths", func(t *testing.T) {
		// Arrange
		args, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, err := gomut.NormalizeTestArgs([]string{"--package", "./internal/gomut", "--jsonl", "mutations.jsonl", "--html", "report.html"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"--package", "./internal/gomut"}, args)
		assert.JSONEq(t, `"mutations.jsonl"`, strconv.Quote(jsonlOutput))
		assert.True(t, jsonlEnabled)
		assert.Equal(t, "report.html", htmlOutput)
		assert.True(t, htmlEnabled)
	})
}

func TestCommandRunHTMLOutput(t *testing.T) {
	t.Run("given html output without a path, it writes html to stdout", func(t *testing.T) {
		// Arrange
		root := createResultFilterFixture(t)

		// Act
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--html"})

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
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--html", htmlPath})

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
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--jsonl", jsonlPath, "--html", htmlPath})

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
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--jsonl", "--html", htmlPath})

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
		_, _, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--type", "lived", "--html", htmlPath})

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
		stdout, stderr, err := runCommandInDir(t, root, []string{"test", "--package", "./sample", "--html", htmlPath})

		// Assert
		require.Error(t, err)
		assert.Empty(t, stdout)
		assert.NotEmpty(t, stderr)

		_, statErr := os.Stat(htmlPath)
		assert.Error(t, statErr)
	})
}

func TestRecordJSONIncludesMutationReplacementDetails(t *testing.T) {
	t.Run("given a mutation record, it serializes original and replacement", func(t *testing.T) {
		// Arrange
		record := gomut.Record{
			Target: gomut.Target{Mode: gomut.TargetModePackage, Value: "./sample"},
			Mutation: gomut.MutationMetadata{
				File:        "sample.go",
				Line:        18,
				Kind:        gomut.MutationKindLogicalOperator,
				Original:    "&&",
				Replacement: "||",
				Result:      gomut.MutationResultLived,
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
		record := gomut.Record{
			Target: gomut.Target{Mode: gomut.TargetModePackage, Value: "./sample"},
			Mutation: gomut.MutationMetadata{
				File:        "sample.go",
				Line:        12,
				Kind:        gomut.MutationKindControlFlow,
				Original:    "ready",
				Replacement: "!ready",
				Result:      gomut.MutationResultKilled,
				Message:     "killed by tests",
			},
		}

		// Act
		data, err := json.Marshal(record)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `{"target":{"mode":"package","value":"./sample"},"started_at":"","command":"","summary":{"total":0,"killed":0,"lived":0,"not_covered":0,"timed_out":0,"not_viable":0},"mutation":{"file":"sample.go","line":12,"kind":"control_flow","original":"ready","replacement":"!ready","result":"KILLED","message":"killed by tests"}}`, string(data))
	})

	t.Run("given an assignment arithmetic mutation record, it serializes the assignment_arithmetic kind", func(t *testing.T) {
		// Arrange
		record := gomut.Record{
			Target: gomut.Target{Mode: gomut.TargetModePackage, Value: "./sample"},
			Mutation: gomut.MutationMetadata{
				File:        "sample.go",
				Line:        52,
				Kind:        gomut.MutationKindAssignmentArithmetic,
				Original:    "+=",
				Replacement: "-=",
				Result:      gomut.MutationResultLived,
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
		record := gomut.Record{
			Target: gomut.Target{Mode: gomut.TargetModePackage, Value: "./sample"},
			Mutation: gomut.MutationMetadata{
				File:        "sample.go",
				Line:        58,
				Kind:        gomut.MutationKindIncDec,
				Original:    "++",
				Replacement: "--",
				Result:      gomut.MutationResultLived,
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
		record := gomut.Record{
			Target: gomut.Target{Mode: gomut.TargetModePackage, Value: "./sample"},
			Mutation: gomut.MutationMetadata{
				File:        "sample.go",
				Line:        22,
				Kind:        gomut.MutationKindReturn,
				Original:    "true",
				Replacement: "false",
				Result:      gomut.MutationResultLived,
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
		record := gomut.Record{
			Target: gomut.Target{Mode: gomut.TargetModePackage, Value: "./sample"},
			Mutation: gomut.MutationMetadata{
				File:        "sample.go",
				Line:        31,
				Kind:        gomut.MutationKindNilCheck,
				Original:    "!=",
				Replacement: "==",
				Result:      gomut.MutationResultLived,
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
		record := gomut.Record{
			Target: gomut.Target{Mode: gomut.TargetModePackage, Value: "./sample"},
			Mutation: gomut.MutationMetadata{
				File:        "sample.go",
				Line:        40,
				Kind:        gomut.MutationKindBooleanLiteral,
				Original:    "true",
				Replacement: "false",
				Result:      gomut.MutationResultLived,
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
