package gomut

import (
	"fmt"
	"gomut/internal/gomut/report"
	"gomut/internal/gomut/result"
	"io"
	"os"
)

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

func (r *Runner) writeCandidateHTML(root string, cfg RunConfig, htmlWriter io.Writer, startedAt, command string, summary result.Summary, records []result.Record) error {
	if !cfg.HTMLEnabled {
		return nil
	}

	return report.WriteHTML(htmlWriter, report.HTMLReportData{
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

func (r *Runner) printSummary(summary result.Summary, total int) {
	fmt.Fprintln(r.stderr, "Mutation summary")
	fmt.Fprintf(r.stderr, "  total: %d\n", total)
	fmt.Fprintf(r.stderr, "  killed: %d\n", summary.Killed)
	fmt.Fprintf(r.stderr, "  lived: %d\n", summary.Lived)
	fmt.Fprintf(r.stderr, "  not covered: %d\n", summary.NotCovered)
	fmt.Fprintf(r.stderr, "  timed out: %d\n", summary.TimedOut)
	fmt.Fprintf(r.stderr, "  not viable: %d\n", summary.NotViable)
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
