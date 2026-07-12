package gomut

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

// HTMLReportData contains the metadata and records needed to render the HTML report.
type HTMLReportData struct {
	Target    Target
	StartedAt string
	Command   string
	Summary   Summary
	Records   []Record
}

// writeHTML renders a self-contained HTML report for the provided mutation records.
func writeHTML(w io.Writer, report HTMLReportData) error {
	view := buildHTMLReportView(report)

	tmpl, err := template.New("html-report").Parse(htmlTemplateSource)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, view)
}

type htmlReportView struct {
	Target        Target
	StartedAt     string
	Command       string
	Summary       Summary
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
}

func buildHTMLReportView(report HTMLReportData) htmlReportView {
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
		})
	}

	return view
}

func formatMutationScore(summary Summary) string {
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
