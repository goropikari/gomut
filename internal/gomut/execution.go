package gomut

import (
	"context"
	"encoding/json"
	"gomut/internal/gomut/result"
	"io"
	"sync"
	"time"
)

type ExecuteMutationFunc func(ctx context.Context, root string, candidate result.Candidate, timeout time.Duration) (result.MutationResult, string, error)

type PrepareMutationRootFunc func(ctx context.Context, root string) (string, func() error, error)

type ExecutorConfig struct {
	Root                string
	Timeout             time.Duration
	Parallel            int
	Target              result.Target
	ResultFilter        result.MutationResultFilter
	ExecuteMutation     ExecuteMutationFunc
	PrepareMutationRoot PrepareMutationRootFunc
	ErrorReporter       func(candidate result.Candidate, err error)
}

type Executor struct {
	cfg ExecutorConfig
}

func NewExecutor(cfg ExecutorConfig) Executor {
	return Executor{cfg: cfg}
}

type candidateOutcome struct {
	index    int
	record   result.Record
	result   result.MutationResult
	included bool
}

func (e Executor) Run(ctx context.Context, candidates []result.Candidate, startedAt, command string, jsonlWriter io.Writer, progress ProgressReporter) (result.Summary, []result.Record, error) {
	if e.cfg.Parallel > 1 && len(candidates) > 1 {
		return e.runParallel(ctx, candidates, startedAt, command, jsonlWriter, progress)
	}

	return e.runSequential(ctx, candidates, startedAt, command, jsonlWriter, progress)
}

func (e Executor) runSequential(ctx context.Context, candidates []result.Candidate, startedAt, command string, jsonlWriter io.Writer, progress ProgressReporter) (result.Summary, []result.Record, error) {
	summary := result.Summary{}
	records := make([]result.Record, 0, len(candidates))
	completed := 0

	for _, candidate := range candidates {
		record, mutationResult, err := e.processCandidate(ctx, candidate, startedAt, command)
		if err != nil {
			e.reportError(candidate, err)

			mutationResult = result.MutationResultNotViable
			record = buildRecord(e.cfg.Target, startedAt, command, candidate, mutationResult, err.Error())
		}

		completed++
		if progress != nil {
			progress.Update(completed)
		}

		if !e.filterMatches(mutationResult) {
			continue
		}

		summary.Total++
		summary = result.UpdateSummary(summary, mutationResult)
		record.Summary = summary
		records = append(records, record)

		if jsonlWriter != nil {
			if err := writeJSONL(jsonlWriter, record); err != nil {
				return result.Summary{}, nil, err
			}
		}
	}

	return summary, records, nil
}

func (e Executor) runParallel(ctx context.Context, candidates []result.Candidate, startedAt, command string, jsonlWriter io.Writer, progress ProgressReporter) (result.Summary, []result.Record, error) {
	workerCount := normalizeParallelWorkerCount(e.cfg.Parallel, len(candidates))

	ordered, err := e.collectParallelCandidateOutcomes(ctx, candidates, startedAt, command, workerCount, progress)
	if err != nil {
		return result.Summary{}, nil, err
	}

	return e.finalizeParallelCandidateOutcomes(jsonlWriter, ordered)
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

func (e Executor) collectParallelCandidateOutcomes(ctx context.Context, candidates []result.Candidate, startedAt, command string, workerCount int, progress ProgressReporter) ([]candidateOutcome, error) {
	jobs := make(chan int)
	results := make(chan candidateOutcome, len(candidates))

	var wg sync.WaitGroup

	workerRoots := make([]string, 0, workerCount)
	workerCleanups := make([]func() error, 0, workerCount)

	for i := 0; i < workerCount; i++ {
		workerRoot, cleanup, err := e.cfg.PrepareMutationRoot(ctx, e.cfg.Root)
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
		go e.runParallelCandidateWorker(ctx, workerRoots[i], candidates, startedAt, command, jobs, results, &wg)
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
		if progress != nil {
			progress.Update(completed)
		}
	}

	return ordered, nil
}

func (e Executor) runParallelCandidateWorker(ctx context.Context, root string, candidates []result.Candidate, startedAt, command string, jobs <-chan int, results chan<- candidateOutcome, wg *sync.WaitGroup) {
	defer wg.Done()

	for index := range jobs {
		results <- e.parallelCandidateOutcome(ctx, root, candidates[index], index, startedAt, command)
	}
}

func (e Executor) parallelCandidateOutcome(ctx context.Context, root string, candidate result.Candidate, index int, startedAt, command string) candidateOutcome {
	if !candidate.Covered {
		record := buildRecord(e.cfg.Target, startedAt, command, candidate, result.MutationResultNotCovered, "line not covered by baseline tests")

		return candidateOutcome{
			index:    index,
			record:   record,
			result:   result.MutationResultNotCovered,
			included: e.filterMatches(result.MutationResultNotCovered),
		}
	}

	record, mutationResult, err := e.executeMutation(ctx, root, candidate, startedAt, command)
	if err != nil {
		e.reportError(candidate, err)

		mutationResult = result.MutationResultNotViable
		record = buildRecord(e.cfg.Target, startedAt, command, candidate, mutationResult, err.Error())
	}

	return candidateOutcome{
		index:    index,
		record:   record,
		result:   mutationResult,
		included: e.filterMatches(mutationResult),
	}
}

func (e Executor) finalizeParallelCandidateOutcomes(jsonlWriter io.Writer, ordered []candidateOutcome) (result.Summary, []result.Record, error) {
	summary := result.Summary{}
	records := make([]result.Record, 0, len(ordered))

	for i := range ordered {
		outcome := ordered[i]
		if !outcome.included {
			continue
		}

		summary.Total++
		summary = result.UpdateSummary(summary, outcome.result)
		outcome.record.Summary = summary
		records = append(records, outcome.record)

		if jsonlWriter != nil {
			if err := writeJSONL(jsonlWriter, outcome.record); err != nil {
				return result.Summary{}, nil, err
			}
		}
	}

	return summary, records, nil
}

func (e Executor) processCandidate(ctx context.Context, candidate result.Candidate, startedAt, command string) (result.Record, result.MutationResult, error) {
	if !candidate.Covered {
		record := buildRecord(e.cfg.Target, startedAt, command, candidate, result.MutationResultNotCovered, "line not covered by baseline tests")
		return record, result.MutationResultNotCovered, nil
	}

	candidateRoot, cleanup, err := e.cfg.PrepareMutationRoot(ctx, e.cfg.Root)
	if err != nil {
		return result.Record{}, "", err
	}
	defer func() {
		_ = cleanup()
	}()

	return e.executeMutation(ctx, candidateRoot, candidate, startedAt, command)
}

func (e Executor) executeMutation(ctx context.Context, root string, candidate result.Candidate, startedAt, command string) (result.Record, result.MutationResult, error) {
	mutationResult, message, err := e.cfg.ExecuteMutation(ctx, root, candidate, e.cfg.Timeout)
	if err != nil {
		return result.Record{}, "", err
	}

	record := buildRecord(e.cfg.Target, startedAt, command, candidate, mutationResult, message)

	return record, mutationResult, nil
}

func (e Executor) reportError(candidate result.Candidate, err error) {
	if e.cfg.ErrorReporter != nil {
		e.cfg.ErrorReporter(candidate, err)
	}
}

func (e Executor) filterMatches(mutationResult result.MutationResult) bool {
	return e.cfg.ResultFilter.Matches(mutationResult)
}

func buildRecord(target result.Target, startedAt, command string, candidate result.Candidate, mutationResult result.MutationResult, message string) result.Record {
	return result.Record{
		Target:    target,
		StartedAt: startedAt,
		Command:   command,
		Summary:   result.Summary{},
		Mutation: result.MutationMetadata{
			File:        candidate.File,
			Line:        candidate.Line,
			Kind:        candidate.Kind,
			Original:    candidate.Original,
			Replacement: candidate.Replacement,
			Result:      mutationResult,
			Message:     message,
			Start:       candidate.Start,
			End:         candidate.End,
		},
	}
}

func writeJSONL(w io.Writer, record result.Record) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(record)
}
