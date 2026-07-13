package gomut

import (
	"context"
	"errors"
	"fmt"
	"gomut/internal/gomut/result"
	"io"
	"os"
	"strings"
	"time"
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

	root, cleanup, err := prepareRunRoot(ctx, originalRoot, r.stderr)
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

	r.reportExclusionNotices(notices)

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

	candidates, notices, err := DiscoverCandidatesWithExclusions(root, packages, cfg.Target, coverage, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("discover candidates: %w", err)
	}

	return candidates, notices, nil
}

func (r *Runner) runCandidates(ctx context.Context, root string, cfg RunConfig, candidates []result.Candidate) error {
	jsonlWriter, htmlWriter, cleanup, err := r.openCandidateOutputs(cfg)
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

	r.printSummary(summary, summary.Total)

	if len(candidates) == 0 {
		fmt.Fprintln(r.stderr, "No mutation candidates found.")
	}

	return nil
}

func (r *Runner) reportExclusionNotices(notices []ExclusionNotice) {
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
