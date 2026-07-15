package report

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/goropikari/gomut/internal/gomut/result"
)

// HTMLReportData contains the metadata and records needed to render the HTML report.
type HTMLReportData struct {
	Root      string
	Target    result.Target
	StartedAt string
	Command   string
	Summary   result.Summary
	Records   []result.Record
}

// WriteHTML renders a self-contained HTML report for the provided mutation records.
func WriteHTML(w io.Writer, report HTMLReportData) error {
	view := buildHTMLReportView(report)

	tmpl, err := template.New("html-report").Parse(htmlTemplateSource)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, view)
}

type htmlReportView struct {
	Target        result.Target
	StartedAt     string
	Command       string
	Summary       result.Summary
	MutationScore string
	Records       []htmlMutationView
}

type htmlMutationView struct {
	File        string
	Line        int
	Link        string
	Kind        string
	KindLower   string
	Original    string
	Replacement string
	Result      string
	ResultClass string
	ResultLower string
	Message     string
	Excerpt     string
	Diff        string
}

func buildHTMLReportView(report HTMLReportData) htmlReportView {
	sourceCache := make(map[string]cachedSource)
	view := htmlReportView{
		Target:        report.Target,
		StartedAt:     report.StartedAt,
		Command:       report.Command,
		Summary:       report.Summary,
		MutationScore: formatMutationScore(report.Summary),
		Records:       make([]htmlMutationView, 0, len(report.Records)),
	}

	for _, record := range report.Records {
		mutation := record.Mutation
		resultLower := resultSlug(string(mutation.Result))
		excerpt, diff := buildMutationCodeViews(report.Root, mutation, sourceCache)
		view.Records = append(view.Records, htmlMutationView{
			File:        mutation.File,
			Line:        mutation.Line,
			Link:        fmt.Sprintf("%s#L%d", mutation.File, mutation.Line),
			Kind:        string(mutation.Kind),
			KindLower:   strings.ToLower(string(mutation.Kind)),
			Original:    mutation.Original,
			Replacement: mutation.Replacement,
			Result:      string(mutation.Result),
			ResultClass: "result-" + resultLower,
			ResultLower: resultLower,
			Message:     mutation.Message,
			Excerpt:     excerpt,
			Diff:        diff,
		})
	}

	return view
}

type cachedSource struct {
	raw   []byte
	lines []string
	ok    bool
}

func buildMutationCodeViews(root string, mutation result.MutationMetadata, cache map[string]cachedSource) (string, string) {
	if root == "" || mutation.File == "" || mutation.Line <= 0 {
		return "", ""
	}

	path := mutation.File
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}

	source, ok := loadSource(path, cache)
	if !ok {
		return "", ""
	}

	before := source.lines
	after := append([]string(nil), before...)

	if mutation.Start >= 0 && mutation.End >= mutation.Start && mutation.End <= len(source.raw) {
		mutated := make([]byte, 0, len(source.raw)-(mutation.End-mutation.Start)+len(mutation.Replacement))
		mutated = append(mutated, source.raw[:mutation.Start]...)
		mutated = append(mutated, mutation.Replacement...)
		mutated = append(mutated, source.raw[mutation.End:]...)
		after = normalizeSourceLines(mutated)
	} else if mutation.Original != "" {
		after = mutateLine(before, mutation.Line, mutation.Original, mutation.Replacement)
	}

	excerpt := renderSourceExcerpt(before, mutation.Line, 2)
	diff := renderUnifiedDiff(mutation.File, before, after, mutation.Line, 2)

	return excerpt, diff
}

func loadSource(path string, cache map[string]cachedSource) (cachedSource, bool) {
	if cached, ok := cache[path]; ok {
		return cached, cached.ok
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		cache[path] = cachedSource{ok: false}

		return cachedSource{}, false
	}

	cached := cachedSource{
		raw:   raw,
		lines: normalizeSourceLines(raw),
		ok:    true,
	}
	cache[path] = cached

	return cached, true
}

func normalizeSourceLines(src []byte) []string {
	text := strings.ReplaceAll(string(src), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")

	if text == "" {
		return nil
	}

	return strings.Split(text, "\n")
}

func mutateLine(lines []string, line int, original, replacement string) []string {
	if line <= 0 || line > len(lines) {
		return append([]string(nil), lines...)
	}

	mutated := append([]string(nil), lines...)
	mutated[line-1] = strings.Replace(mutated[line-1], original, replacement, 1)

	return mutated
}

func renderSourceExcerpt(lines []string, line, context int) string {
	start, end := lineBounds(len(lines), line, context)
	if start == 0 {
		return ""
	}

	var b strings.Builder

	for i := start; i <= end; i++ {
		marker := " "

		if i == line {
			marker = ">"
		}

		fmt.Fprintf(&b, "%s %4d | %s\n", marker, i, lines[i-1])
	}

	return strings.TrimRight(b.String(), "\n")
}

func renderUnifiedDiff(path string, before, after []string, line, context int) string {
	start, end := lineBounds(len(before), line, context)
	if start == 0 {
		return ""
	}

	var b strings.Builder

	fmt.Fprintf(&b, "--- a/%s\n", path)
	fmt.Fprintf(&b, "+++ b/%s\n", path)
	fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", start, end-start+1, start, end-start+1)

	for i := start; i <= end; i++ {
		if i == line {
			fmt.Fprintf(&b, "- %s\n", before[i-1])
			fmt.Fprintf(&b, "+ %s\n", after[i-1])

			continue
		}

		fmt.Fprintf(&b, "  %s\n", before[i-1])
	}

	return strings.TrimRight(b.String(), "\n")
}

func lineBounds(total, line, context int) (int, int) {
	if total == 0 || line <= 0 {
		return 0, 0
	}

	start := line - context
	if start < 1 {
		start = 1
	}

	end := line + context
	if end > total {
		end = total
	}

	return start, end
}

func formatMutationScore(summary result.Summary) string {
	denominator := summary.Total - summary.NotViable - summary.NotCovered
	if denominator <= 0 {
		return "0.0%"
	}

	score := float64(summary.Killed) / float64(denominator) * 100

	return fmt.Sprintf("%.1f%%", score)
}

func resultSlug(result string) string {
	slug := strings.ToLower(result)
	slug = strings.ReplaceAll(slug, " ", "-")

	return slug
}
