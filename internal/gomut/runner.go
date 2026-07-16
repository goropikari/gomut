package gomut

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/goropikari/gomut/internal/gomut/result"
)

type Runner struct {
	stdout              io.Writer
	stderr              io.Writer
	executeMutationFunc func(ctx context.Context, root string, candidate result.Candidate, timeout time.Duration) (result.MutationResult, string, error)
}

func NewRunner(stdout, stderr io.Writer) *Runner {
	return &Runner{stdout: stdout, stderr: stderr}
}

func (r *Runner) Run(ctx context.Context, cfg RunConfig) (err error) {
	originalRoot, err := os.Getwd()
	if err != nil {
		return err
	}

	root, cleanup, err := prepareRunRoot(ctx, originalRoot, r.stderr, cfg.IsolationCopyExclude)
	if err != nil {
		return err
	}

	defer func() {
		if cleanup == nil {
			return
		}

		if cleanupErr := cleanup(); cleanupErr != nil && err == nil {
			err = cleanupErr
		}
	}()

	candidates, notices, err := r.discoverCandidates(ctx, originalRoot, root, cfg)
	if err != nil {
		return err
	}

	r.reportExclusionNotices(notices, cfg.Verbose)

	return r.runCandidates(ctx, root, cfg, candidates)
}

func (r *Runner) discoverCandidates(ctx context.Context, originalRoot, root string, cfg RunConfig) ([]result.Candidate, []ExclusionNotice, error) {
	packages, err := resolvePackages(ctx, originalRoot, root, cfg.Target)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve packages: %w", err)
	}

	if len(packages) == 0 {
		return nil, nil, errors.New("no packages matched target")
	}

	coverage, err := runBaseline(ctx, root, packages)
	if err != nil {
		return nil, nil, fmt.Errorf("run baseline: %w", err)
	}

	filter, err := NewExclusionFilter(root, cfg.Exclude)
	if err != nil {
		return nil, nil, fmt.Errorf("build exclusion filter: %w", err)
	}

	candidates, notices, err := DiscoverCandidatesWithExclusions(root, packages, cfg.Target, coverage, filter, cfg.KindFilter)
	if err != nil {
		return nil, nil, fmt.Errorf("discover candidates: %w", err)
	}

	return candidates, notices, nil
}

func (r *Runner) runCandidates(ctx context.Context, root string, cfg RunConfig, candidates []result.Candidate) error {
	jsonlWriter, htmlWriter, sarifWriter, cleanup, err := r.openCandidateOutputs(cfg)
	if err != nil {
		return err
	}

	if cleanup != nil {
		defer cleanup()
	}

	startedAt := time.Now().Format(time.RFC3339)
	command := strings.Join(os.Args, " ")
	progress := NewProgressReporter(ProgressConfig{
		Mode:        cfg.ProgressMode,
		Writer:      r.stderr,
		Interactive: isInteractiveWriter(r.stderr),
		CI:          isCIEnvironment(),
		Total:       len(candidates),
	})

	progress.Start(len(candidates))
	defer progress.Finish()

	summary, records, err := r.runCandidateLoop(ctx, root, cfg, candidates, startedAt, command, jsonlWriter, progress)
	if err != nil {
		return err
	}

	progress.Finish()

	if err := r.writeCandidateHTML(root, cfg, htmlWriter, startedAt, command, summary, records); err != nil {
		return err
	}

	if err := r.writeCandidateSARIF(cfg, sarifWriter, startedAt, command, summary, records); err != nil {
		return err
	}

	r.printSummary(summary, summary.Total)

	if len(candidates) == 0 {
		fmt.Fprintln(r.stderr, "No mutation candidates found.")
	}

	return nil
}

func (r *Runner) reportExclusionNotices(notices []ExclusionNotice, verbose bool) {
	if !verbose {
		return
	}

	for _, notice := range notices {
		if notice.Line > 0 {
			fmt.Fprintf(r.stderr, "excluded %s:%d: %s\n", notice.File, notice.Line, notice.Reason)
			continue
		}

		fmt.Fprintf(r.stderr, "excluded %s: %s\n", notice.File, notice.Reason)
	}
}

func (r *Runner) runCandidateLoop(ctx context.Context, root string, cfg RunConfig, candidates []result.Candidate, startedAt, command string, jsonlWriter io.Writer, progress ProgressReporter) (result.Summary, []result.Record, error) {
	executor := NewExecutor(ExecutorConfig{
		Root:                root,
		Timeout:             cfg.Timeout,
		Parallel:            cfg.Parallel,
		Target:              cfg.Target,
		ResultFilter:        cfg.ResultFilter,
		ExecuteMutation:     r.executeMutation,
		PrepareMutationRoot: prepareMutationRoot,
		ErrorReporter:       r.reportCandidateError,
	})

	return executor.Run(ctx, candidates, startedAt, command, jsonlWriter, progress)
}

func (r *Runner) reportCandidateError(candidate result.Candidate, err error) {
	if r.stderr == nil {
		return
	}

	fmt.Fprintf(r.stderr, "mutation execution error for %s:%d: %v\n", candidate.File, candidate.Line, err)
}

func (r *Runner) executeMutation(ctx context.Context, root string, candidate result.Candidate, timeout time.Duration) (result.MutationResult, string, error) {
	if r.executeMutationFunc != nil {
		return r.executeMutationFunc(ctx, root, candidate, timeout)
	}

	mutated, err := ApplyMutation(root, candidate)
	if err != nil {
		return result.MutationResultNotViable, err.Error(), nil
	}

	pkg := candidate.PackagePath
	if pkg == "" {
		pkg, err = packageForFile(root, candidate.File)
		if err != nil {
			return result.MutationResultNotViable, err.Error(), nil
		}
	}

	path := resolveSourcePath(root, candidate.File)

	backup, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	if err := writeFilePreserveMode(path, mutated); err != nil {
		return "", "", err
	}
	defer func() {
		_ = writeFilePreserveMode(path, backup)
	}()

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "go", "test", pkg)
	cmd.Dir = root
	cmd.Env = goCommandEnv()
	out, err := cmd.CombinedOutput()

	if cmdCtx.Err() == context.DeadlineExceeded {
		return result.MutationResultTimedOut, "mutation test timed out", nil
	}

	if err != nil {
		if looksLikeNotViable(string(out)) {
			return result.MutationResultNotViable, trimOutput(out), nil
		}

		return result.MutationResultKilled, trimOutput(out), nil
	}

	return result.MutationResultLived, trimOutput(out), nil
}

func looksLikeNotViable(output string) bool {
	needles := []string{
		"build failed",
		"syntax error",
		"undefined:",
		"cannot use",
		"mismatched types",
		"too many errors",
	}
	for _, needle := range needles {
		if strings.Contains(output, needle) {
			return true
		}
	}

	return false
}

func trimOutput(out []byte) string {
	text := strings.TrimSpace(string(out))
	if text == "" {
		return "tests passed"
	}

	return text
}
