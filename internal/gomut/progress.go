package gomut

import (
	"fmt"
	"io"
	"time"
)

type ProgressMode string

const (
	ProgressModeAuto ProgressMode = "auto"
	ProgressModeOn   ProgressMode = "on"
	ProgressModeOff  ProgressMode = "off"
)

// ProgressConfig captures the runtime inputs needed to decide how progress is displayed.
type ProgressConfig struct {
	Mode        ProgressMode
	Writer      io.Writer
	Interactive bool
	CI          bool
	Total       int
}

// ProgressReporter provides the mutation-loop hooks used to emit progress updates.
type ProgressReporter interface {
	Enabled() bool
	Start(total int)
	Update(completed int)
	Finish()
}

type progressReporter struct {
	writer      io.Writer
	enabled     bool
	interactive bool
	total       int
	startedAt   time.Time
	last        int
	emitted     bool
	finished    bool
}

// NewProgressReporter returns a progress reporter for the current run configuration.
func NewProgressReporter(config ProgressConfig) ProgressReporter {
	reporter := &progressReporter{
		writer:      config.Writer,
		interactive: config.Interactive,
		total:       config.Total,
	}

	switch config.Mode {
	case ProgressModeOn:
		reporter.enabled = true
	case ProgressModeOff:
		reporter.enabled = false
	default:
		reporter.enabled = config.Interactive && !config.CI
	}

	if !reporter.enabled {
		return noopProgressReporter{}
	}

	return reporter
}

type noopProgressReporter struct{}

func (progressReporter) Enabled() bool {
	return true
}

func (p *progressReporter) Start(total int) {
	if total > 0 {
		p.total = total
	}

	p.startedAt = time.Now()
}

func (p *progressReporter) Update(completed int) {
	if p.writer == nil || p.total <= 0 {
		return
	}

	p.last = completed
	p.emitted = true

	line := formatProgressLine(completed, p.total, time.Since(p.startedAt))
	if p.interactive {
		fmt.Fprintf(p.writer, "\r\033[K%s", line)
		return
	}

	fmt.Fprintln(p.writer, line)
}

func (p *progressReporter) Finish() {
	if p.writer == nil || !p.emitted || p.finished {
		return
	}

	p.finished = true

	if p.interactive {
		fmt.Fprintln(p.writer)
	}
}

func (noopProgressReporter) Enabled() bool {
	return false
}

func (noopProgressReporter) Start(total int) {}

func (noopProgressReporter) Update(completed int) {}

func (noopProgressReporter) Finish() {}

func formatProgressLine(completed, total int, elapsed time.Duration) string {
	percent := 0
	if total > 0 {
		percent = completed * 100 / total
	}

	line := fmt.Sprintf("Progress: %d/%d (%d%%)", completed, total, percent)

	if eta := estimateRemaining(completed, total, elapsed); eta != "" {
		line += " eta " + eta
	}

	return line
}

func estimateRemaining(completed, total int, elapsed time.Duration) string {
	if completed <= 0 || total <= completed || elapsed <= 0 {
		return ""
	}

	remaining := time.Duration(int64(elapsed) * int64(total-completed) / int64(completed))
	if remaining < 0 {
		return ""
	}

	return remaining.Round(time.Second).String()
}
