package gomut

import "time"

type TargetMode string

const (
	TargetModePackage TargetMode = "package"
	TargetModeAll     TargetMode = "all"
	TargetModeDiff    TargetMode = "diff"
)

type MutationKind string

const (
	MutationKindComparisonOperator MutationKind = "comparison_operator"
	MutationKindLogicalOperator     MutationKind = "logical_operator"
	MutationKindGuardClause         MutationKind = "guard_clause"
	MutationKindArithmeticOperator   MutationKind = "arithmetic_operator"
	MutationKindControlFlow         MutationKind = "control_flow"
	MutationKindAssignmentBitwise   MutationKind = "assignment_bitwise"
	MutationKindReturn              MutationKind = "return"
)

type MutationResult string

const (
	MutationResultKilled     MutationResult = "KILLED"
	MutationResultLived      MutationResult = "LIVED"
	MutationResultNotCovered MutationResult = "NOT COVERED"
	MutationResultTimedOut   MutationResult = "TIMED OUT"
	MutationResultNotViable  MutationResult = "NOT VIABLE"
)

type Target struct {
	Mode  TargetMode `json:"mode"`
	Value string     `json:"value"`
}

type Summary struct {
	Total      int `json:"total"`
	Killed     int `json:"killed"`
	Lived      int `json:"lived"`
	NotCovered int `json:"not_covered"`
	TimedOut   int `json:"timed_out"`
	NotViable  int `json:"not_viable"`
}

type MutationMetadata struct {
	File    string         `json:"file"`
	Line    int            `json:"line"`
	Kind    MutationKind   `json:"kind"`
	Result  MutationResult `json:"result"`
	Message string         `json:"message"`
}

type Record struct {
	Target    Target           `json:"target"`
	StartedAt string            `json:"started_at"`
	Command   string            `json:"command"`
	Summary   Summary           `json:"summary"`
	Mutation  MutationMetadata  `json:"mutation"`
}

type Candidate struct {
	File        string
	Line        int
	Kind        MutationKind
	Original    string
	Replacement string
	Start       int
	End         int
	PackagePath string
	Covered     bool
}

type FileCoverage struct {
	Ranges []CoverageRange
}

type CoverageRange struct {
	StartLine int
	EndLine   int
	Covered   bool
}

type Result struct {
	Candidate Candidate
	Result    MutationResult
	Message   string
}

type RunConfig struct {
	Target     Target
	Timeout    time.Duration
	OutputPath string
}
