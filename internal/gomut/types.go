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
	MutationKindComparisonOperator   MutationKind = "comparison_operator"
	MutationKindLogicalOperator      MutationKind = "logical_operator"
	MutationKindGuardClause          MutationKind = "guard_clause"
	MutationKindArithmeticOperator   MutationKind = "arithmetic_operator"
	MutationKindBitwiseOperator      MutationKind = "bitwise_operator"
	MutationKindShiftOperator        MutationKind = "shift_operator"
	MutationKindAssignmentArithmetic MutationKind = "assignment_arithmetic"
	MutationKindAssignmentShift      MutationKind = "assignment_shift"
	MutationKindControlFlow          MutationKind = "control_flow"
	MutationKindAssignmentBitwise    MutationKind = "assignment_bitwise"
	MutationKindIncDec               MutationKind = "inc_dec"
	MutationKindReturn               MutationKind = "return"
	MutationKindNilCheck             MutationKind = "nil_check"
	MutationKindBooleanLiteral       MutationKind = "boolean_literal"
	MutationKindIntegerLiteral       MutationKind = "integer_literal"
	MutationKindFloatLiteral         MutationKind = "float_literal"
	MutationKindRuneLiteral          MutationKind = "rune_literal"
	MutationKindUnaryNot             MutationKind = "unary_not"
	MutationKindUnaryMinus           MutationKind = "unary_minus"
	MutationKindUnaryBitwiseNot      MutationKind = "unary_bitwise_not"
	MutationKindSwitchCondition      MutationKind = "switch_condition"
	MutationKindStringLiteral        MutationKind = "string_literal"
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
	File        string         `json:"file"`
	Line        int            `json:"line"`
	Kind        MutationKind   `json:"kind"`
	Original    string         `json:"original"`
	Replacement string         `json:"replacement"`
	Result      MutationResult `json:"result"`
	Message     string         `json:"message"`
}

type Record struct {
	Target    Target           `json:"target"`
	StartedAt string           `json:"started_at"`
	Command   string           `json:"command"`
	Summary   Summary          `json:"summary"`
	Mutation  MutationMetadata `json:"mutation"`
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
	Target      Target
	Timeout     time.Duration
	OutputPath  string
	UseWorktree bool
}
