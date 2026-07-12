package gomut

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Runner struct {
	stdout              io.Writer
	stderr              io.Writer
	executeMutationFunc func(ctx context.Context, root string, candidate Candidate, timeout time.Duration) (MutationResult, string, error)
}

type candidateOutcome struct {
	index    int
	record   Record
	result   MutationResult
	included bool
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

	packages, err := r.resolvePackages(ctx, originalRoot, root, cfg.Target)
	if err != nil {
		return fmt.Errorf("resolve packages: %w", err)
	}

	if len(packages) == 0 {
		return errors.New("no packages matched target")
	}

	coverage, err := r.runBaseline(ctx, root, packages)
	if err != nil {
		return fmt.Errorf("run baseline: %w", err)
	}

	candidates, err := DiscoverCandidates(root, packages, cfg.Target, coverage)
	if err != nil {
		return fmt.Errorf("discover candidates: %w", err)
	}

	return r.runCandidates(ctx, root, cfg, candidates)
}

func (r *Runner) runCandidates(ctx context.Context, root string, cfg RunConfig, candidates []Candidate) error {
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

func (r *Runner) openCandidateOutputs(cfg RunConfig) (io.Writer, io.Writer, func(), error) {
	var cleanup func()

	jsonlWriter, jsonlCleanup, err := openJSONLOutput(cfg, r.stdout)
	if err != nil {
		return nil, nil, nil, err
	}

	if jsonlCleanup != nil {
		cleanup = chainCleanup(cleanup, jsonlCleanup)
	}

	htmlWriter, htmlCleanup, err := openHTMLOutput(cfg, r.stdout)
	if err != nil {
		return nil, nil, nil, err
	}

	if htmlCleanup != nil {
		cleanup = chainCleanup(cleanup, htmlCleanup)
	}

	return jsonlWriter, htmlWriter, cleanup, nil
}

func openJSONLOutput(cfg RunConfig, stdout io.Writer) (io.Writer, func(), error) {
	if cfg.OutputPath != "" {
		outputFile, err := openOutput(cfg.OutputPath)
		if err != nil {
			return nil, nil, err
		}

		return outputFile, func() {
			_ = outputFile.Close()
		}, nil
	}

	if cfg.HTMLEnabled && cfg.HTMLPath == "" {
		return nil, nil, nil
	}

	if cfg.HTMLPath != "" && !cfg.JSONLEnabled {
		return nil, nil, nil
	}

	return stdout, nil, nil
}

func openHTMLOutput(cfg RunConfig, stdout io.Writer) (io.Writer, func(), error) {
	if !cfg.HTMLEnabled {
		return nil, nil, nil
	}

	if cfg.HTMLPath == "" {
		return stdout, nil, nil
	}

	outputFile, err := openOutput(cfg.HTMLPath)
	if err != nil {
		return nil, nil, err
	}

	return outputFile, func() {
		_ = outputFile.Close()
	}, nil
}

func chainCleanup(existing, next func()) func() {
	if existing == nil {
		return next
	}

	if next == nil {
		return existing
	}

	return func() {
		existing()
		next()
	}
}

func (r *Runner) runCandidateLoop(ctx context.Context, root string, cfg RunConfig, candidates []Candidate, startedAt, command string, jsonlWriter io.Writer, progress ProgressReporter) (Summary, []Record, error) {
	if cfg.Parallel > 1 && len(candidates) > 1 {
		return r.runCandidateLoopParallel(ctx, root, cfg, candidates, startedAt, command, jsonlWriter, progress)
	}

	return r.runCandidateLoopSequential(ctx, root, cfg, candidates, startedAt, command, jsonlWriter, progress)
}

func (r *Runner) runCandidateLoopSequential(ctx context.Context, root string, cfg RunConfig, candidates []Candidate, startedAt, command string, jsonlWriter io.Writer, progress ProgressReporter) (Summary, []Record, error) {
	summary := Summary{}
	records := make([]Record, 0, len(candidates))
	completed := 0

	for _, candidate := range candidates {
		record, result, err := r.processCandidate(ctx, root, cfg, candidate, startedAt, command)
		if err != nil {
			r.reportCandidateError(candidate, err)

			result = MutationResultNotViable
			record = r.buildRecord(cfg.Target, startedAt, command, candidate, result, err.Error())
		}

		completed++
		progress.Update(completed)

		if !cfg.ResultFilter.Matches(result) {
			continue
		}

		summary.Total++
		summary = updateSummary(summary, result)
		record.Summary = summary
		records = append(records, record)

		if jsonlWriter != nil {
			if err := writeJSONL(jsonlWriter, record); err != nil {
				return Summary{}, nil, err
			}
		}
	}

	return summary, records, nil
}

func (r *Runner) runCandidateLoopParallel(ctx context.Context, root string, cfg RunConfig, candidates []Candidate, startedAt, command string, jsonlWriter io.Writer, progress ProgressReporter) (Summary, []Record, error) {
	workerCount := normalizeParallelWorkerCount(cfg.Parallel, len(candidates))

	ordered, err := r.collectParallelCandidateOutcomes(ctx, root, cfg, candidates, startedAt, command, workerCount, progress)
	if err != nil {
		return Summary{}, nil, err
	}

	return r.finalizeParallelCandidateOutcomes(jsonlWriter, ordered)
}

func normalizeParallelWorkerCount(value, total int) int {
	if value <= 0 {
		value = 1
	}

	if value > total {
		return total
	}

	return value
}

func (r *Runner) collectParallelCandidateOutcomes(ctx context.Context, root string, cfg RunConfig, candidates []Candidate, startedAt, command string, workerCount int, progress ProgressReporter) ([]candidateOutcome, error) {
	jobs := make(chan int)
	results := make(chan candidateOutcome, len(candidates))

	var wg sync.WaitGroup

	workerRoots := make([]string, 0, workerCount)
	workerCleanups := make([]func() error, 0, workerCount)

	for i := 0; i < workerCount; i++ {
		workerRoot, cleanup, err := prepareMutationRoot(ctx, root)
		if err != nil {
			for _, workerCleanup := range workerCleanups {
				_ = workerCleanup()
			}

			return nil, err
		}

		workerRoots = append(workerRoots, workerRoot)
		workerCleanups = append(workerCleanups, cleanup)
	}

	defer func() {
		for _, workerCleanup := range workerCleanups {
			_ = workerCleanup()
		}
	}()

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go r.runParallelCandidateWorker(ctx, workerRoots[i], cfg, candidates, startedAt, command, jobs, results, &wg)
	}

	go func() {
		for index := range candidates {
			jobs <- index
		}

		close(jobs)
		wg.Wait()
		close(results)
	}()

	ordered := make([]candidateOutcome, len(candidates))
	completed := 0

	for outcome := range results {
		ordered[outcome.index] = outcome
		completed++
		progress.Update(completed)
	}

	return ordered, nil
}

func (r *Runner) runParallelCandidateWorker(ctx context.Context, root string, cfg RunConfig, candidates []Candidate, startedAt, command string, jobs <-chan int, results chan<- candidateOutcome, wg *sync.WaitGroup) {
	defer wg.Done()

	for index := range jobs {
		results <- r.parallelCandidateOutcome(ctx, root, cfg, candidates[index], index, startedAt, command)
	}
}

func (r *Runner) parallelCandidateOutcome(ctx context.Context, root string, cfg RunConfig, candidate Candidate, index int, startedAt, command string) candidateOutcome {
	if !candidate.Covered {
		record := r.buildRecord(cfg.Target, startedAt, command, candidate, MutationResultNotCovered, "line not covered by baseline tests")

		return candidateOutcome{
			index:    index,
			record:   record,
			result:   MutationResultNotCovered,
			included: cfg.ResultFilter.Matches(MutationResultNotCovered),
		}
	}

	record, result, err := r.processCandidateInRoot(ctx, root, cfg, candidate, startedAt, command)
	if err != nil {
		r.reportCandidateError(candidate, err)

		result = MutationResultNotViable
		record = r.buildRecord(cfg.Target, startedAt, command, candidate, result, err.Error())
	}

	return candidateOutcome{
		index:    index,
		record:   record,
		result:   result,
		included: cfg.ResultFilter.Matches(result),
	}
}

func (r *Runner) processCandidateInRoot(ctx context.Context, root string, cfg RunConfig, candidate Candidate, startedAt, command string) (Record, MutationResult, error) {
	result, message, err := r.executeMutation(ctx, root, candidate, cfg.Timeout)
	if err != nil {
		return Record{}, "", err
	}

	record := r.buildRecord(cfg.Target, startedAt, command, candidate, result, message)

	return record, result, nil
}

func (r *Runner) finalizeParallelCandidateOutcomes(jsonlWriter io.Writer, ordered []candidateOutcome) (Summary, []Record, error) {
	summary := Summary{}
	records := make([]Record, 0, len(ordered))

	for i := range ordered {
		outcome := ordered[i]
		if !outcome.included {
			continue
		}

		summary.Total++
		summary = updateSummary(summary, outcome.result)
		outcome.record.Summary = summary
		records = append(records, outcome.record)

		if jsonlWriter != nil {
			if err := writeJSONL(jsonlWriter, outcome.record); err != nil {
				return Summary{}, nil, err
			}
		}
	}

	return summary, records, nil
}

func (r *Runner) reportCandidateError(candidate Candidate, err error) {
	if r.stderr == nil {
		return
	}

	fmt.Fprintf(r.stderr, "mutation execution error for %s:%d: %v\n", candidate.File, candidate.Line, err)
}

func (r *Runner) writeCandidateHTML(root string, cfg RunConfig, htmlWriter io.Writer, startedAt, command string, summary Summary, records []Record) error {
	if !cfg.HTMLEnabled {
		return nil
	}

	return WriteHTML(htmlWriter, HTMLReportData{
		Root:      root,
		Target:    cfg.Target,
		StartedAt: startedAt,
		Command:   command,
		Summary:   summary,
		Records:   records,
	})
}

func openOutput(path string) (*os.File, error) {
	if path == "" {
		return nil, nil
	}

	outputFile, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return outputFile, nil
}

func (r *Runner) processCandidate(ctx context.Context, root string, cfg RunConfig, candidate Candidate, startedAt, command string) (Record, MutationResult, error) {
	if !candidate.Covered {
		record := r.buildRecord(cfg.Target, startedAt, command, candidate, MutationResultNotCovered, "line not covered by baseline tests")
		return record, MutationResultNotCovered, nil
	}

	candidateRoot, cleanup, err := prepareMutationRoot(ctx, root)
	if err != nil {
		return Record{}, "", err
	}
	defer func() {
		_ = cleanup()
	}()

	return r.processCandidateInRoot(ctx, candidateRoot, cfg, candidate, startedAt, command)
}

func (r *Runner) buildRecord(target Target, startedAt, command string, candidate Candidate, result MutationResult, message string) Record {
	return Record{
		Target:    target,
		StartedAt: startedAt,
		Command:   command,
		Mutation: MutationMetadata{
			File:        candidate.File,
			Line:        candidate.Line,
			Kind:        candidate.Kind,
			Original:    candidate.Original,
			Replacement: candidate.Replacement,
			Result:      result,
			Message:     message,
			Start:       candidate.Start,
			End:         candidate.End,
		},
	}
}

func updateSummary(summary Summary, result MutationResult) Summary {
	switch result {
	case MutationResultKilled:
		summary.Killed++
	case MutationResultLived:
		summary.Lived++
	case MutationResultNotCovered:
		summary.NotCovered++
	case MutationResultTimedOut:
		summary.TimedOut++
	case MutationResultNotViable:
		summary.NotViable++
	}

	return summary
}

func (r *Runner) printSummary(summary Summary, total int) {
	fmt.Fprintln(r.stderr, "Mutation summary")
	fmt.Fprintf(r.stderr, "  total: %d\n", total)
	fmt.Fprintf(r.stderr, "  killed: %d\n", summary.Killed)
	fmt.Fprintf(r.stderr, "  lived: %d\n", summary.Lived)
	fmt.Fprintf(r.stderr, "  not covered: %d\n", summary.NotCovered)
	fmt.Fprintf(r.stderr, "  timed out: %d\n", summary.TimedOut)
	fmt.Fprintf(r.stderr, "  not viable: %d\n", summary.NotViable)
}

func (r *Runner) resolvePackages(ctx context.Context, originalRoot, root string, target Target) ([]string, error) {
	switch target.Mode {
	case TargetModePackage:
		return []string{target.Value}, nil
	case TargetModeAll:
		packages, err := listPackages(ctx, root, "./...")
		if err != nil {
			return nil, fmt.Errorf("list packages for all target: %w", err)
		}

		return packages, nil
	case TargetModeDiff:
		files, err := DiffFiles(ctx, originalRoot, target.Value)
		if err != nil {
			return nil, fmt.Errorf("collect diff files: %w", err)
		}

		packages, err := packageDirsFromFiles(root, files)
		if err != nil {
			return nil, fmt.Errorf("resolve packages from diff files: %w", err)
		}

		return packages, nil
	default:
		return nil, fmt.Errorf("unsupported target mode %q", target.Mode)
	}
}

func (r *Runner) runBaseline(ctx context.Context, root string, packages []string) (map[string]FileCoverage, error) {
	merged := map[string]FileCoverage{}

	modulePath, err := modulePath(root)
	if err != nil {
		return nil, err
	}

	for _, pkg := range packages {
		coverProfile := filepath.Join(os.TempDir(), strings.ReplaceAll("gomut-"+strings.ReplaceAll(pkg, "/", "-"), "...", "all")+".cover")
		cmd := exec.CommandContext(ctx, "go", "test", "-coverprofile", coverProfile, pkg)
		cmd.Dir = root
		cmd.Env = goCommandEnv()

		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("baseline go test failed for %s: %w\n%s", pkg, err, string(out))
		}

		coverage, err := readCoverage(root, coverProfile, modulePath)
		if err != nil {
			return nil, err
		}

		for file, cov := range coverage {
			merged[file] = mergeCoverage(merged[file], cov)
		}
	}

	return merged, nil
}

func mergeCoverage(dst, src FileCoverage) FileCoverage {
	dst.Ranges = append(dst.Ranges, src.Ranges...)
	return dst
}

func (r *Runner) executeMutation(ctx context.Context, root string, candidate Candidate, timeout time.Duration) (MutationResult, string, error) {
	if r.executeMutationFunc != nil {
		return r.executeMutationFunc(ctx, root, candidate, timeout)
	}

	mutated, err := ApplyMutation(root, candidate)
	if err != nil {
		return MutationResultNotViable, err.Error(), nil
	}

	pkg := candidate.PackagePath
	if pkg == "" {
		pkg, err = packageForFile(root, candidate.File)
		if err != nil {
			return MutationResultNotViable, err.Error(), nil
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
		return MutationResultTimedOut, "mutation test timed out", nil
	}

	if err != nil {
		if looksLikeNotViable(string(out)) {
			return MutationResultNotViable, trimOutput(out), nil
		}

		return MutationResultKilled, trimOutput(out), nil
	}

	return MutationResultLived, trimOutput(out), nil
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

func isInteractiveWriter(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func isCIEnvironment() bool {
	return os.Getenv("CI") != ""
}
