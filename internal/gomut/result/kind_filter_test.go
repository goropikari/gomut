package result_test

import (
	"gomut/internal/gomut/result"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMutationKindFilter(t *testing.T) {
	t.Run("given requested kinds, it allows only those kinds", func(t *testing.T) {
		// Arrange
		filter, err := result.ParseMutationKindFilter([]string{"comparison_operator", "return"})

		// Assert
		require.NoError(t, err)
		assert.True(t, filter.Matches(result.MutationKindComparisonOperator))
		assert.True(t, filter.Matches(result.MutationKindReturn))
		assert.False(t, filter.Matches(result.MutationKindLogicalOperator))
	})

	t.Run("given comma-separated and repeated kinds, it deduplicates them and trims spaces", func(t *testing.T) {
		// Arrange
		filter, err := result.ParseMutationKindFilter([]string{"comparison_operator, return", "return", "nil_check"})

		// Assert
		require.NoError(t, err)
		assert.True(t, filter.Matches(result.MutationKindComparisonOperator))
		assert.True(t, filter.Matches(result.MutationKindReturn))
		assert.True(t, filter.Matches(result.MutationKindNilCheck))
		assert.False(t, filter.Matches(result.MutationKindLogicalOperator))
	})

	t.Run("given an unknown kind, it returns a helpful error", func(t *testing.T) {
		// Act
		_, err := result.ParseMutationKindFilter([]string{"comparsion_operator"})

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "comparsion_operator")
		assert.Contains(t, err.Error(), "comparison_operator")
	})

	t.Run("given no kinds, it allows every kind", func(t *testing.T) {
		// Arrange
		filter, err := result.ParseMutationKindFilter(nil)

		// Assert
		require.NoError(t, err)
		assert.True(t, filter.Matches(result.MutationKindComparisonOperator))
		assert.True(t, filter.Matches(result.MutationKindStringLiteral))
	})

	t.Run("given all kinds, it exposes the supported kind list", func(t *testing.T) {
		// Arrange
		kinds := result.AllMutationKinds()

		// Assert
		require.NotEmpty(t, kinds)
		assert.Contains(t, kinds, result.MutationKindComparisonOperator)
		assert.Contains(t, kinds, result.MutationKindReturn)
		assert.Contains(t, kinds, result.MutationKindStringLiteral)
	})
}
