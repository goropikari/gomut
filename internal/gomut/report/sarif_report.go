package report

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/goropikari/gomut/internal/gomut/result"
)

// SARIFReportData contains the metadata and records needed to render a SARIF report.
type SARIFReportData struct {
	Target    result.Target
	StartedAt string
	Command   string
	Summary   result.Summary
	Records   []result.Record
}

// WriteSARIF renders a SARIF 2.1.0 log for the provided mutation records.
func WriteSARIF(w io.Writer, report SARIFReportData) error {
	view := buildSARIFLog(report)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	return encoder.Encode(view)
}

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results,omitempty"`
}

type sarifTool struct {
	Driver sarifToolDriver `json:"driver"`
}

type sarifToolDriver struct {
	Name           string            `json:"name"`
	InformationURI string            `json:"informationUri,omitempty"`
	Rules          []sarifRule       `json:"rules,omitempty"`
	Properties     map[string]string `json:"properties,omitempty"`
}

type sarifRule struct {
	ID               string            `json:"id"`
	Name             string            `json:"name,omitempty"`
	ShortDescription sarifMessage      `json:"shortDescription,omitempty"`
	HelpText         sarifMessage      `json:"help,omitempty"`
	Properties       map[string]string `json:"properties,omitempty"`
}

type sarifResult struct {
	RuleID     string            `json:"ruleId,omitempty"`
	Level      string            `json:"level,omitempty"`
	Message    sarifMessage      `json:"message"`
	Locations  []sarifLocation   `json:"locations,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine,omitempty"`
}

func buildSARIFLog(report SARIFReportData) sarifLog {
	rules := make([]sarifRule, 0, len(report.Records))
	ruleSeen := make(map[string]struct{})
	results := make([]sarifResult, 0, len(report.Records))

	for _, record := range report.Records {
		mutation := record.Mutation

		level, ok := sarifLevelForResult(mutation.Result)
		if !ok {
			continue
		}

		ruleID := string(mutation.Kind)
		if _, seen := ruleSeen[ruleID]; !seen {
			ruleSeen[ruleID] = struct{}{}
			rules = append(rules, sarifRule{
				ID:   ruleID,
				Name: ruleID,
				ShortDescription: sarifMessage{
					Text: mutationKindDescription(mutation.Kind),
				},
			})
		}

		results = append(results, sarifResult{
			RuleID: ruleID,
			Level:  level,
			Message: sarifMessage{
				Text: buildSARIFMessage(mutation),
			},
			Locations: []sarifLocation{
				{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{
							URI: filepath.ToSlash(mutation.File),
						},
						Region: sarifRegion{
							StartLine: mutation.Line,
							EndLine:   mutation.Line,
						},
					},
				},
			},
			Properties: map[string]string{
				"result":      string(mutation.Result),
				"original":    mutation.Original,
				"replacement": mutation.Replacement,
				"command":     report.Command,
				"started_at":  report.StartedAt,
			},
		})
	}

	return sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifToolDriver{
						Name:           "gomut",
						InformationURI: "https://github.com/goropikari/gomut",
						Rules:          rules,
						Properties: map[string]string{
							"target_mode":  string(report.Target.Mode),
							"target_value": report.Target.Value,
							"started_at":   report.StartedAt,
							"command":      report.Command,
							"total":        fmt.Sprint(report.Summary.Total),
						},
					},
				},
				Results: results,
			},
		},
	}
}

func sarifLevelForResult(mutationResult result.MutationResult) (string, bool) {
	switch mutationResult {
	case result.MutationResultLived:
		return "error", true
	case result.MutationResultNotCovered, result.MutationResultTimedOut:
		return "warning", true
	case result.MutationResultNotViable:
		return "note", true
	default:
		return "", false
	}
}

var mutationKindDescriptions = map[result.MutationKind]string{
	result.MutationKindComparisonOperator:   "Comparison operator mutation",
	result.MutationKindLogicalOperator:      "Logical operator mutation",
	result.MutationKindGuardClause:          "Guard clause mutation",
	result.MutationKindArithmeticOperator:   "Arithmetic operator mutation",
	result.MutationKindBitwiseOperator:      "Bitwise operator mutation",
	result.MutationKindShiftOperator:        "Shift operator mutation",
	result.MutationKindAssignmentArithmetic: "Assignment arithmetic mutation",
	result.MutationKindAssignmentShift:      "Assignment shift mutation",
	result.MutationKindControlFlow:          "Control flow mutation",
	result.MutationKindLoopControl:          "Loop control mutation",
	result.MutationKindAssignmentBitwise:    "Assignment bitwise mutation",
	result.MutationKindIncDec:               "Increment/decrement mutation",
	result.MutationKindReturn:               "Return mutation",
	result.MutationKindNilCheck:             "Nil check mutation",
	result.MutationKindBooleanLiteral:       "Boolean literal mutation",
	result.MutationKindIntegerLiteral:       "Integer literal mutation",
	result.MutationKindFloatLiteral:         "Float literal mutation",
	result.MutationKindRuneLiteral:          "Rune literal mutation",
	result.MutationKindUnaryNot:             "Unary not mutation",
	result.MutationKindUnaryMinus:           "Unary minus mutation",
	result.MutationKindUnaryBitwiseNot:      "Unary bitwise not mutation",
	result.MutationKindSwitchCondition:      "Switch condition mutation",
	result.MutationKindStringLiteral:        "String literal mutation",
}

func mutationKindDescription(kind result.MutationKind) string {
	if description, ok := mutationKindDescriptions[kind]; ok {
		return description
	}

	return "Mutation"
}

func buildSARIFMessage(mutation result.MutationMetadata) string {
	parts := []string{string(mutation.Result)}

	if mutation.Original != "" || mutation.Replacement != "" {
		parts = append(parts, fmt.Sprintf("Origin: %s | Replace: %s", mutation.Original, mutation.Replacement))
	}

	return strings.Join(parts, " | ")
}
