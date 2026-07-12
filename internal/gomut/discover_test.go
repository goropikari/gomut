package gomut_test

import (
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

func AddBonus(score, bonus int) int {
	score += bonus
	return score
}

func AdvanceCount(count int) int {
	count++
	return count
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
		candidates, err := gomut.DiscoverCandidates(root, []string{"./sample"}, gomut.Target{Mode: gomut.TargetModePackage, Value: "./sample"}, map[string]gomut.FileCoverage{})

		// Assert
		require.NoError(t, err)
		require.NotEmpty(t, candidates)

		kinds := map[gomut.MutationKind]bool{}

		var controlFlowCandidate *gomut.Candidate

		for i := range candidates {
			candidate := candidates[i]

			kinds[candidate.Kind] = true
			if candidate.Kind == gomut.MutationKindControlFlow && candidate.Original == "ready" {
				controlFlowCandidate = &candidates[i]
			}
		}

		assert.True(t, kinds[gomut.MutationKindComparisonOperator], "expected comparison operator mutation to remain available")
		assert.True(t, kinds[gomut.MutationKindLogicalOperator], "expected logical operator mutation to remain available")
		assert.True(t, kinds[gomut.MutationKindArithmeticOperator], "expected arithmetic operator mutation to remain available")
		assert.True(t, kinds[gomut.MutationKindAssignmentArithmetic], "expected assignment arithmetic mutation to be discovered")
		assert.True(t, kinds[gomut.MutationKindAssignmentBitwise], "expected assignment bitwise mutation to be discovered")
		assert.True(t, kinds[gomut.MutationKindIncDec], "expected inc/dec mutation to be discovered")
		assert.True(t, kinds[gomut.MutationKindGuardClause], "expected guard clause mutation to remain available")
		assert.True(t, kinds[gomut.MutationKindControlFlow], "expected control flow mutation to be discovered")
		assert.True(t, kinds[gomut.MutationKindReturn], "expected return mutation to be discovered")
		assert.True(t, kinds[gomut.MutationKindNilCheck], "expected nil check mutation to be discovered")
		assert.True(t, kinds[gomut.MutationKindBooleanLiteral], "expected boolean literal mutation to be discovered")

		require.NotNil(t, controlFlowCandidate)
		assert.Contains(t, controlFlowCandidate.File, filepath.ToSlash(filepath.Join("sample", "sample.go")))
		assert.Equal(t, gomut.MutationKindControlFlow, controlFlowCandidate.Kind)
		assert.Equal(t, "ready", controlFlowCandidate.Original)
		assert.Equal(t, "!ready", controlFlowCandidate.Replacement)
		assert.Positive(t, controlFlowCandidate.Line)
	})
}
