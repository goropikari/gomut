package gomut_test

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		args, output, err := gomut.NormalizeTestArgs([]string{"--package", "./internal/gomut", "--jsonl"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"--package", "./internal/gomut"}, args)
		assert.Empty(t, output)
	})

	t.Run("given jsonl with a value, it captures the file path", func(t *testing.T) {
		// Arrange
		args, output, err := gomut.NormalizeTestArgs([]string{"--package", "./internal/gomut", "--jsonl", "mutations.jsonl"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"--package", "./internal/gomut"}, args)
		assert.Equal(t, "mutations.jsonl", output)
	})

	t.Run("given jsonl equals syntax, it captures the file path", func(t *testing.T) {
		// Arrange
		args, output, err := gomut.NormalizeTestArgs([]string{"--package", "./internal/gomut", "--jsonl=mutations.jsonl"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"--package", "./internal/gomut"}, args)
		assert.Equal(t, "mutations.jsonl", output)
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
}
