package result

import (
	"fmt"
	"strings"
)

var allMutationKinds = []MutationKind{
	MutationKindComparisonOperator,
	MutationKindLogicalOperator,
	MutationKindGuardClause,
	MutationKindArithmeticOperator,
	MutationKindBitwiseOperator,
	MutationKindShiftOperator,
	MutationKindAssignmentArithmetic,
	MutationKindAssignmentShift,
	MutationKindControlFlow,
	MutationKindAssignmentBitwise,
	MutationKindIncDec,
	MutationKindReturn,
	MutationKindNilCheck,
	MutationKindBooleanLiteral,
	MutationKindIntegerLiteral,
	MutationKindFloatLiteral,
	MutationKindRuneLiteral,
	MutationKindUnaryNot,
	MutationKindUnaryMinus,
	MutationKindUnaryBitwiseNot,
	MutationKindSwitchCondition,
	MutationKindStringLiteral,
}

// AllMutationKinds returns the supported mutation kinds in declaration order.
func AllMutationKinds() []MutationKind {
	return append([]MutationKind(nil), allMutationKinds...)
}

// MutationKindFilter defines the allowed mutation kinds for discovery.
type MutationKindFilter struct {
	allowed map[MutationKind]struct{}
}

// ParseMutationKindFilter converts raw CLI or config values into a mutation kind filter.
func ParseMutationKindFilter(values []string) (MutationKindFilter, error) {
	allowed := map[MutationKind]struct{}{}
	invalid := map[string]struct{}{}
	orderedInvalid := make([]string, 0)

	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(token)
			if trimmed == "" {
				continue
			}

			kind := MutationKind(trimmed)
			if !isKnownMutationKind(kind) {
				if _, seen := invalid[trimmed]; !seen {
					invalid[trimmed] = struct{}{}
					orderedInvalid = append(orderedInvalid, trimmed)
				}

				continue
			}

			allowed[kind] = struct{}{}
		}
	}

	if len(orderedInvalid) > 0 {
		return MutationKindFilter{}, fmt.Errorf(
			"unknown mutation kind%s: %s (available: %s)",
			pluralSuffix(len(orderedInvalid)),
			strings.Join(orderedInvalid, ", "),
			strings.Join(allMutationKindStrings(), ", "),
		)
	}

	return MutationKindFilter{allowed: allowed}, nil
}

// Matches reports whether the given kind should be included in discovery output.
func (f MutationKindFilter) Matches(kind MutationKind) bool {
	if len(f.allowed) == 0 {
		return true
	}

	_, ok := f.allowed[kind]

	return ok
}

func isKnownMutationKind(kind MutationKind) bool {
	for _, candidate := range allMutationKinds {
		if candidate == kind {
			return true
		}
	}

	return false
}

func allMutationKindStrings() []string {
	values := make([]string, 0, len(allMutationKinds))
	for _, kind := range allMutationKinds {
		values = append(values, string(kind))
	}

	return values
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}

	return "s"
}
