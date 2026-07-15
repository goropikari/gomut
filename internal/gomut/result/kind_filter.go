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
	MutationKindLoopControl,
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

var standardMutationKinds = []MutationKind{
	MutationKindComparisonOperator,
	MutationKindLogicalOperator,
	MutationKindArithmeticOperator,
	MutationKindGuardClause,
	MutationKindReturn,
	MutationKindNilCheck,
}

// AllMutationKinds returns the supported mutation kinds in declaration order.
func AllMutationKinds() []MutationKind {
	return append([]MutationKind(nil), allMutationKinds...)
}

// StandardMutationKinds returns the default standard kind set in declaration order.
func StandardMutationKinds() []MutationKind {
	return append([]MutationKind(nil), standardMutationKinds...)
}

// MutationKindMode defines the base mutation kind selection mode.
type MutationKindMode string

const (
	MutationKindModeStandard MutationKindMode = "standard"
	MutationKindModeAll      MutationKindMode = "all"
)

// MutationKindFilter defines the allowed mutation kinds for discovery.
type MutationKindFilter struct {
	allowed map[MutationKind]struct{}
}

// ParseMutationKindFilter converts kind mode, enable, and disable values into a mutation kind filter.
func ParseMutationKindFilter(mode string, enable, disable []string) (MutationKindFilter, error) {
	selected, err := parseMutationKinds(mode, enable, disable)
	if err != nil {
		return MutationKindFilter{}, err
	}

	return MutationKindFilter{allowed: selected}, nil
}

// Matches reports whether the given kind should be included in discovery output.
func (f MutationKindFilter) Matches(kind MutationKind) bool {
	if f.allowed == nil {
		return true
	}

	_, ok := f.allowed[kind]

	return ok
}

func parseMutationKinds(mode string, enable, disable []string) (map[MutationKind]struct{}, error) {
	base, err := parseMutationKindMode(mode)
	if err != nil {
		return nil, err
	}

	allowed := make(map[MutationKind]struct{}, len(base))
	for _, kind := range base {
		allowed[kind] = struct{}{}
	}

	enableKinds, invalidEnable := parseMutationKindList(enable)

	disableKinds, invalidDisable := parseMutationKindList(disable)

	invalid := append([]string(nil), invalidEnable...)
	invalid = append(invalid, invalidDisable...)

	if len(invalid) > 0 {
		return nil, fmt.Errorf(
			"unknown mutation kind%s: %s (available: %s)",
			pluralSuffix(len(invalid)),
			strings.Join(invalid, ", "),
			strings.Join(allMutationKindStrings(), ", "),
		)
	}

	for _, kind := range enableKinds {
		allowed[kind] = struct{}{}
	}

	for _, kind := range disableKinds {
		delete(allowed, kind)
	}

	return allowed, nil
}

func parseMutationKindMode(mode string) ([]MutationKind, error) {
	trimmed := strings.ToLower(strings.TrimSpace(mode))
	switch MutationKindMode(trimmed) {
	case "", MutationKindModeStandard:
		return StandardMutationKinds(), nil
	case MutationKindModeAll:
		return AllMutationKinds(), nil
	default:
		return nil, fmt.Errorf("unknown mutation kind mode: %s (available: standard, all)", mode)
	}
}

func parseMutationKindList(values []string) ([]MutationKind, []string) {
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

	kinds := make([]MutationKind, 0, len(allowed))
	for _, kind := range allMutationKinds {
		if _, ok := allowed[kind]; ok {
			kinds = append(kinds, kind)
		}
	}

	return kinds, orderedInvalid
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
