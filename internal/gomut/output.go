package gomut

import (
	"fmt"
	"io"
	"os"

	"github.com/goropikari/gomut/internal/gomut/report"
	"github.com/goropikari/gomut/internal/gomut/result"
)

func (r *Runner) openCandidateOutputs(cfg RunConfig) (io.Writer, io.Writer, io.Writer, func(), error) {
	outputs, err := buildCandidateOutputs(cfg, r.stdout)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return outputs.jsonl, outputs.html, outputs.sarif, outputs.cleanup, nil
}

type candidateOutputs struct {
	jsonl   io.Writer
	html    io.Writer
	sarif   io.Writer
	cleanup func()
}

func buildCandidateOutputs(cfg RunConfig, stdout io.Writer) (candidateOutputs, error) {
	var (
		outputs candidateOutputs
		cleanup func()
	)

	stdoutTaken := false

	var err error
	if outputs.sarif, cleanup, stdoutTaken, err = openConfiguredOutput(cfg.SARIFEnabled, cfg.SARIFPath, stdout, stdoutTaken); err != nil {
		return candidateOutputs{}, err
	}

	outputs.cleanup = chainCleanup(outputs.cleanup, cleanup)

	if outputs.html, cleanup, stdoutTaken, err = openConfiguredOutput(cfg.HTMLEnabled, cfg.HTMLPath, stdout, stdoutTaken); err != nil {
		if outputs.cleanup != nil {
			outputs.cleanup()
		}

		return candidateOutputs{}, err
	}

	outputs.cleanup = chainCleanup(outputs.cleanup, cleanup)

	if outputs.jsonl, cleanup, _, err = openJSONLOutput(cfg, stdout, stdoutTaken); err != nil {
		if outputs.cleanup != nil {
			outputs.cleanup()
		}

		return candidateOutputs{}, err
	}

	outputs.cleanup = chainCleanup(outputs.cleanup, cleanup)

	return outputs, nil
}

func openJSONLOutput(cfg RunConfig, stdout io.Writer, stdoutTaken bool) (io.Writer, func(), bool, error) {
	if cfg.OutputPath != "" {
		outputFile, err := openOutput(cfg.OutputPath)
		if err != nil {
			return nil, nil, false, err
		}

		return outputFile, func() {
			_ = outputFile.Close()
		}, false, nil
	}

	if shouldSuppressJSONLOutput(cfg, stdoutTaken) {
		return nil, nil, false, nil
	}

	return stdout, nil, true, nil
}

func shouldSuppressJSONLOutput(cfg RunConfig, stdoutTaken bool) bool {
	if cfg.SARIFEnabled && cfg.SARIFPath == "" {
		return true
	}

	if cfg.HTMLEnabled && cfg.HTMLPath == "" {
		return true
	}

	if stdoutTaken {
		return true
	}

	if !cfg.JSONLEnabled && (cfg.HTMLPath != "" || cfg.SARIFPath != "") {
		return true
	}

	return false
}

func openConfiguredOutput(enabled bool, path string, stdout io.Writer, stdoutTaken bool) (io.Writer, func(), bool, error) {
	if !enabled {
		return nil, nil, false, nil
	}

	if path == "" && stdoutTaken {
		return nil, nil, false, nil
	}

	if path == "" {
		return stdout, nil, true, nil
	}

	outputFile, err := openOutput(path)
	if err != nil {
		return nil, nil, false, err
	}

	return outputFile, func() {
		_ = outputFile.Close()
	}, false, nil
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

func (r *Runner) writeCandidateSARIF(cfg RunConfig, sarifWriter io.Writer, startedAt, command string, summary result.Summary, records []result.Record) error {
	if !cfg.SARIFEnabled {
		return nil
	}

	return report.WriteSARIF(sarifWriter, report.SARIFReportData{
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
