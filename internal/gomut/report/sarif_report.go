package report

import (
	"encoding/json"
	"fmt"
	"gomut/internal/gomut/result"
	"io"
	"path/filepath"
	"strings"
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

func mutationKindDescription(kind result.MutationKind) string {
	switch kind {
	case result.MutationKindComparisonOperator:
		return "Comparison operator mutation"
	case result.MutationKindLogicalOperator:
		return "Logical operator mutation"
	case result.MutationKindGuardClause:
		return "Guard clause mutation"
	case result.MutationKindArithmeticOperator:
		return "Arithmetic operator mutation"
	case result.MutationKindBitwiseOperator:
		return "Bitwise operator mutation"
	case result.MutationKindShiftOperator:
		return "Shift operator mutation"
	case result.MutationKindAssignmentArithmetic:
		return "Assignment arithmetic mutation"
	case result.MutationKindAssignmentShift:
		return "Assignment shift mutation"
	case result.MutationKindControlFlow:
		return "Control flow mutation"
	case result.MutationKindLoopControl:
		return "Loop control mutation"
	case result.MutationKindAssignmentBitwise:
		return "Assignment bitwise mutation"
	case result.MutationKindIncDec:
		return "Increment/decrement mutation"
	case result.MutationKindReturn:
		return "Return mutation"
	case result.MutationKindNilCheck:
		return "Nil check mutation"
	case result.MutationKindBooleanLiteral:
		return "Boolean literal mutation"
	case result.MutationKindIntegerLiteral:
		return "Integer literal mutation"
	case result.MutationKindFloatLiteral:
		return "Float literal mutation"
	case result.MutationKindRuneLiteral:
		return "Rune literal mutation"
	case result.MutationKindUnaryNot:
		return "Unary not mutation"
	case result.MutationKindUnaryMinus:
		return "Unary minus mutation"
	case result.MutationKindUnaryBitwiseNot:
		return "Unary bitwise not mutation"
	case result.MutationKindSwitchCondition:
		return "Switch condition mutation"
	case result.MutationKindStringLiteral:
		return "String literal mutation"
	default:
		return "Mutation"
	}
}

func buildSARIFMessage(mutation result.MutationMetadata) string {
	parts := []string{string(mutation.Result)}

	if mutation.Original != "" || mutation.Replacement != "" {
		parts = append(parts, fmt.Sprintf("Origin: %s | Replace: %s", mutation.Original, mutation.Replacement))
	}

	return strings.Join(parts, " | ")
}
