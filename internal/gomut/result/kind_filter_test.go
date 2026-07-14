package result_test

import (
	"gomut/internal/gomut/result"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMutationKindFilter(t *testing.T) {
	t.Run("given the standard mode, it allows the standard set", func(t *testing.T) {
		// Arrange
		filter, err := result.ParseMutationKindFilter(string(result.MutationKindModeStandard), nil, nil)

		// Assert
		require.NoError(t, err)
		assert.True(t, filter.Matches(result.MutationKindComparisonOperator))
		assert.True(t, filter.Matches(result.MutationKindLogicalOperator))
		assert.True(t, filter.Matches(result.MutationKindArithmeticOperator))
		assert.True(t, filter.Matches(result.MutationKindGuardClause))
		assert.True(t, filter.Matches(result.MutationKindReturn))
		assert.True(t, filter.Matches(result.MutationKindNilCheck))
		assert.False(t, filter.Matches(result.MutationKindStringLiteral))
	})

	t.Run("given the all mode, it allows every supported kind", func(t *testing.T) {
		// Arrange
		filter, err := result.ParseMutationKindFilter(string(result.MutationKindModeAll), nil, nil)

		// Assert
		require.NoError(t, err)
		assert.True(t, filter.Matches(result.MutationKindComparisonOperator))
		assert.True(t, filter.Matches(result.MutationKindReturn))
		assert.True(t, filter.Matches(result.MutationKindNilCheck))
		assert.True(t, filter.Matches(result.MutationKindLogicalOperator))
		assert.True(t, filter.Matches(result.MutationKindStringLiteral))
	})

	t.Run("given enable and disable values, disable wins", func(t *testing.T) {
		// Act
		filter, err := result.ParseMutationKindFilter(
			string(result.MutationKindModeStandard),
			[]string{"bitwise_operator", "guard_clause"},
			[]string{"guard_clause"},
		)

		// Assert
		require.NoError(t, err)
		assert.True(t, filter.Matches(result.MutationKindComparisonOperator))
		assert.True(t, filter.Matches(result.MutationKindBitwiseOperator))
		assert.False(t, filter.Matches(result.MutationKindGuardClause))
	})

	t.Run("given all kinds disabled, it rejects every kind", func(t *testing.T) {
		// Arrange
		filter, err := result.ParseMutationKindFilter(string(result.MutationKindModeAll), nil, []string{"comparison_operator", "logical_operator", "guard_clause", "arithmetic_operator", "bitwise_operator", "shift_operator", "assignment_arithmetic", "assignment_shift", "control_flow", "loop_control", "assignment_bitwise", "inc_dec", "return", "nil_check", "boolean_literal", "integer_literal", "float_literal", "rune_literal", "unary_not", "unary_minus", "unary_bitwise_not", "switch_condition", "string_literal"})

		// Assert
		require.NoError(t, err)
		assert.False(t, filter.Matches(result.MutationKindComparisonOperator))
		assert.False(t, filter.Matches(result.MutationKindStringLiteral))
		assert.False(t, filter.Matches(result.MutationKindLoopControl))
	})

	t.Run("given an unknown kind, it returns a helpful error", func(t *testing.T) {
		// Arrange
		_, err := result.ParseMutationKindFilter(string(result.MutationKindModeStandard), []string{"comparsion_operator"}, nil)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "comparsion_operator")
		assert.Contains(t, err.Error(), "comparison_operator")
	})

	t.Run("given an unknown mode, it returns a helpful error", func(t *testing.T) {
		// Arrange
		_, err := result.ParseMutationKindFilter("experimental", nil, nil)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "experimental")
		assert.Contains(t, err.Error(), "standard")
		assert.Contains(t, err.Error(), "all")
	})

	t.Run("given the standard kinds list, it exposes the default set", func(t *testing.T) {
		// Arrange
		kinds := result.StandardMutationKinds()

		// Assert
		require.NotEmpty(t, kinds)
		assert.Contains(t, kinds, result.MutationKindComparisonOperator)
		assert.Contains(t, kinds, result.MutationKindLogicalOperator)
		assert.Contains(t, kinds, result.MutationKindArithmeticOperator)
		assert.Contains(t, kinds, result.MutationKindGuardClause)
		assert.Contains(t, kinds, result.MutationKindReturn)
		assert.Contains(t, kinds, result.MutationKindNilCheck)
		assert.NotContains(t, kinds, result.MutationKindStringLiteral)
	})
}
