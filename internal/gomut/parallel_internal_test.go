package gomut

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTestRunConfigParallel(t *testing.T) {
	t.Run("given config parallel and no CLI worker count, it uses config value", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, ".gomut.yaml"), []byte("parallel: 4\n"), 0o600))
		t.Chdir(root)

		command := NewCommand(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
		cmd := command.newTestCommand()
		require.NoError(t, cmd.Flags().Set("package", "./sample"))

		// Act
		cfg, err := command.buildTestRunConfig(cmd)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 4, cfg.Parallel)
	})

	t.Run("given a CLI worker count, it overrides config", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, ".gomut.yaml"), []byte("parallel: 4\n"), 0o600))
		t.Chdir(root)

		command := NewCommand(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
		cmd := command.newTestCommand()
		require.NoError(t, cmd.Flags().Set("package", "./sample"))
		require.NoError(t, cmd.Flags().Set("parallel", "2"))

		// Act
		cfg, err := command.buildTestRunConfig(cmd)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 2, cfg.Parallel)
	})

	t.Run("given no worker count anywhere, it defaults to CPU cores", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		t.Chdir(root)

		command := NewCommand(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
		cmd := command.newTestCommand()
		require.NoError(t, cmd.Flags().Set("package", "./sample"))

		// Act
		cfg, err := command.buildTestRunConfig(cmd)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, runtime.NumCPU(), cfg.Parallel)
	})
}

func TestRunnerRunCandidateLoopParallel(t *testing.T) {
	t.Run("given parallel workers, it starts multiple candidates before releasing output", func(t *testing.T) {
		// Arrange
		var (
			stdout bytes.Buffer
			stderr bytes.Buffer
		)

		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "placeholder.txt"), []byte("ok"), 0o600))

		started := make(chan string, 3)
		release := make(chan struct{})
		inFlight := new(int32)
		maxFlight := new(int32)
		runner := newParallelRunnerFixture(started, release, inFlight, maxFlight, &stdout, &stderr)
		candidates := parallelCandidates()
		cfg := RunConfig{Parallel: 2, ResultFilter: MutationResultFilter{}}
		progress := NewProgressReporter(ProgressConfig{Mode: ProgressModeOff, Writer: &stderr, Interactive: false, CI: true, Total: len(candidates)})
		jsonl := &bytes.Buffer{}

		var (
			summary Summary
			records []Record
			runErr  error
		)

		done := make(chan struct{})

		// Act
		go func() {
			summary, records, runErr = runner.runCandidateLoop(context.Background(), root, cfg, candidates, "2026-07-12T00:00:00Z", "gomut test --parallel 2", jsonl, progress)

			close(done)
		}()

		require.Equal(t, "a.go", waitForStart(t, started))
		require.Equal(t, "b.go", waitForStart(t, started))

		close(release)

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("runCandidateLoop did not finish")
		}

		// Assert
		require.NoError(t, runErr)
		assert.Equal(t, 3, summary.Total)
		assert.Equal(t, 2, summary.Killed)
		assert.Equal(t, 1, summary.NotViable)
		assert.Contains(t, stderr.String(), "mutation execution error for b.go:11")
		require.Len(t, records, 3)
		assert.Equal(t, "a.go", records[0].Mutation.File)
		assert.Equal(t, "b.go", records[1].Mutation.File)
		assert.Equal(t, "c.go", records[2].Mutation.File)
		assert.Equal(t, 1, records[0].Summary.Total)
		assert.Equal(t, 2, records[1].Summary.Total)
		assert.Equal(t, 3, records[2].Summary.Total)
		assert.Equal(t, int32(2), atomic.LoadInt32(maxFlight))

		lines := bytes.Split(bytes.TrimSpace(jsonl.Bytes()), []byte{'\n'})
		require.Len(t, lines, 3)

		for i, line := range lines {
			var record Record
			require.NoError(t, json.Unmarshal(line, &record))
			assert.Equal(t, records[i].Mutation.File, record.Mutation.File)
			assert.Equal(t, records[i].Summary.Total, record.Summary.Total)
		}
	})
}

func waitForStart(t *testing.T, started <-chan string) string {
	t.Helper()

	select {
	case candidate := <-started:
		return candidate
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for candidate start")
		return ""
	}
}

func newParallelRunnerFixture(started chan<- string, release <-chan struct{}, inFlight, maxFlight *int32, stdout, stderr *bytes.Buffer) *Runner {
	return &Runner{
		stdout: stdout,
		stderr: stderr,
		executeMutationFunc: func(ctx context.Context, root string, candidate Candidate, timeout time.Duration) (MutationResult, string, error) {
			active := atomic.AddInt32(inFlight, 1)
			observeMaxFlight(active, maxFlight)

			started <- candidate.File

			if err := waitForParallelRelease(ctx, release); err != nil {
				return "", "", err
			}

			defer atomic.AddInt32(inFlight, -1)

			return parallelMutationResult(candidate.File)
		},
	}
}

func observeMaxFlight(active int32, maxFlight *int32) {
	for {
		previous := atomic.LoadInt32(maxFlight)
		if active <= previous || atomic.CompareAndSwapInt32(maxFlight, previous, active) {
			return
		}
	}
}

func waitForParallelRelease(ctx context.Context, release <-chan struct{}) error {
	select {
	case <-release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func parallelMutationResult(file string) (MutationResult, string, error) {
	switch file {
	case "a.go":
		time.Sleep(100 * time.Millisecond)
		return MutationResultKilled, "a", nil
	case "b.go":
		return MutationResultNotViable, "b", fmt.Errorf("boom")
	default:
		time.Sleep(10 * time.Millisecond)
		return MutationResultKilled, "c", nil
	}
}

func parallelCandidates() []Candidate {
	return []Candidate{
		{
			File:        "a.go",
			Line:        10,
			Kind:        MutationKindComparisonOperator,
			Original:    "==",
			Replacement: "!=",
			PackagePath: "./sample",
			Covered:     true,
		},
		{
			File:        "b.go",
			Line:        11,
			Kind:        MutationKindComparisonOperator,
			Original:    "==",
			Replacement: "!=",
			PackagePath: "./sample",
			Covered:     true,
		},
		{
			File:        "c.go",
			Line:        12,
			Kind:        MutationKindComparisonOperator,
			Original:    "==",
			Replacement: "!=",
			PackagePath: "./sample",
			Covered:     true,
		},
	}
}
