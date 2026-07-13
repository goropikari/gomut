package gomut_test

import (
	"gomut/internal/gomut/result"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomut "gomut/internal/gomut"
)

func TestDiscoverCandidates(t *testing.T) {
	t.Run("given a package with control flow and existing mutations, it discovers all supported kinds", func(t *testing.T) {
		// Arrange
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/mut\n\ngo 1.26\n"), 0o600))
		require.NoError(t, os.MkdirAll(filepath.Join(root, "sample"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "sample", "sample.go"), []byte(`package sample

import "errors"

func CanPublish(age int, approved bool) bool {
	if age >= 18 && approved {
		return true
	}

	return false
}

func ValidateQuantity(quantity int) error {
	if quantity < 0 {
		err := errors.New("quantity must be non-negative")
		_ = err

		return err
	}

	return nil
}

func IsReady(ready bool) bool {
	if ready {
		return true
	}

	return false
}

func Add(a, b int) int {
	return a + b
}

func SetMask(mask uint8) uint8 {
	mask |= 1
	return mask
}

func KeepCommonBits(mask, flag uint8) uint8 {
	return mask & flag
}

func MergeFlags(mask, flag uint8) uint8 {
	return mask | flag
}

func ClearFlagBits(mask, flag uint8) uint8 {
	return mask &^ flag
}

func ShiftLeft(value uint8) uint8 {
	return value << 1
}

func ShiftRight(value uint8) uint8 {
	return value >> 1
}

func ShiftCounter(value uint8) uint8 {
	value <<= 1
	return value
}

func NegateScore(score int) int {
	return -score
}

func InvertBits(value uint8) uint8 {
	return ^value
}

func AddBonus(score, bonus int) int {
	score += bonus
	return score
}

func AdvanceCount(count int) int {
	count++
	return count
}

func DefaultRetryLimit() int {
	return 3
}

func Greeting() string {
	return "hello"
}

func DefaultTaxRate() float64 {
	return 0.1
}

func DefaultGrade() rune {
	return 'A'
}

func IsBlocked(approved bool) bool {
	return !approved
}

func ApprovalLabel(approved bool) string {
	switch approved {
	case true:
		return "approved"
	default:
		return "blocked"
	}
}

func HasNickname(nickname *string) bool {
	return nickname != nil
}

func AlwaysEnabled() bool {
	return true
}

func AlwaysDisabled() bool {
	return false
}
`), 0o600))

		// Act
		candidates, err := gomut.DiscoverCandidates(root, []string{"./sample"}, result.Target{Mode: result.TargetModePackage, Value: "./sample"}, map[string]result.FileCoverage{})

		// Assert
		require.NoError(t, err)
		require.NotEmpty(t, candidates)

		kinds := map[result.MutationKind]bool{}

		var controlFlowCandidate *result.Candidate

		for i := range candidates {
			candidate := candidates[i]

			kinds[candidate.Kind] = true
			if candidate.Kind == result.MutationKindControlFlow && candidate.Original == "ready" {
				controlFlowCandidate = &candidates[i]
			}
		}

		assert.True(t, kinds[result.MutationKindComparisonOperator], "expected comparison operator mutation to remain available")
		assert.True(t, kinds[result.MutationKindLogicalOperator], "expected logical operator mutation to remain available")
		assert.True(t, kinds[result.MutationKindArithmeticOperator], "expected arithmetic operator mutation to remain available")
		assert.True(t, kinds[result.MutationKindBitwiseOperator], "expected bitwise operator mutation to be discovered")
		assert.True(t, kinds[result.MutationKindShiftOperator], "expected shift operator mutation to be discovered")
		assert.True(t, kinds[result.MutationKindAssignmentArithmetic], "expected assignment arithmetic mutation to be discovered")
		assert.True(t, kinds[result.MutationKindAssignmentShift], "expected assignment shift mutation to be discovered")
		assert.True(t, kinds[result.MutationKindAssignmentBitwise], "expected assignment bitwise mutation to be discovered")
		assert.True(t, kinds[result.MutationKindIncDec], "expected inc/dec mutation to be discovered")
		assert.True(t, kinds[result.MutationKindGuardClause], "expected guard clause mutation to remain available")
		assert.True(t, kinds[result.MutationKindControlFlow], "expected control flow mutation to be discovered")
		assert.True(t, kinds[result.MutationKindReturn], "expected return mutation to be discovered")
		assert.True(t, kinds[result.MutationKindNilCheck], "expected nil check mutation to be discovered")
		assert.True(t, kinds[result.MutationKindBooleanLiteral], "expected boolean literal mutation to be discovered")
		assert.True(t, kinds[result.MutationKindIntegerLiteral], "expected integer literal mutation to be discovered")
		assert.True(t, kinds[result.MutationKindFloatLiteral], "expected float literal mutation to be discovered")
		assert.True(t, kinds[result.MutationKindRuneLiteral], "expected rune literal mutation to be discovered")
		assert.True(t, kinds[result.MutationKindUnaryNot], "expected unary not mutation to be discovered")
		assert.True(t, kinds[result.MutationKindUnaryMinus], "expected unary minus mutation to be discovered")
		assert.True(t, kinds[result.MutationKindUnaryBitwiseNot], "expected unary bitwise not mutation to be discovered")
		assert.True(t, kinds[result.MutationKindSwitchCondition], "expected switch condition mutation to be discovered")
		assert.True(t, kinds[result.MutationKindStringLiteral], "expected string literal mutation to be discovered")

		require.NotNil(t, controlFlowCandidate)
		assert.Contains(t, controlFlowCandidate.File, filepath.ToSlash(filepath.Join("sample", "sample.go")))
		assert.Equal(t, result.MutationKindControlFlow, controlFlowCandidate.Kind)
		assert.Equal(t, "ready", controlFlowCandidate.Original)
		assert.Equal(t, "!ready", controlFlowCandidate.Replacement)
		assert.Positive(t, controlFlowCandidate.Line)
	})
}
