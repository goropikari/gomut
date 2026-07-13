package result_test

import (
	"gomut/internal/gomut/result"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMutationResultFilter(t *testing.T) {
	t.Run("given a lived filter, it allows lived results and rejects others", func(t *testing.T) {
		// Arrange
		filter, err := result.ParseMutationResultFilter([]string{"lived"})

		// Assert
		require.NoError(t, err)
		assert.True(t, filter.Matches(result.MutationResultLived))
		assert.False(t, filter.Matches(result.MutationResultKilled))
	})

	t.Run("given repeated filter values, it allows each requested result", func(t *testing.T) {
		// Arrange
		filter, err := result.ParseMutationResultFilter([]string{"killed", "timed-out"})

		// Assert
		require.NoError(t, err)
		assert.True(t, filter.Matches(result.MutationResultKilled))
		assert.True(t, filter.Matches(result.MutationResultTimedOut))
		assert.False(t, filter.Matches(result.MutationResultLived))
	})

	t.Run("given space or underscore separated values, it normalizes them", func(t *testing.T) {
		// Arrange
		filter, err := result.ParseMutationResultFilter([]string{"not covered", "not_viable"})

		// Assert
		require.NoError(t, err)
		assert.True(t, filter.Matches(result.MutationResultNotCovered))
		assert.True(t, filter.Matches(result.MutationResultNotViable))
		assert.False(t, filter.Matches(result.MutationResultKilled))
	})

	t.Run("given an unknown filter value, it returns an error", func(t *testing.T) {
		// Act
		_, err := result.ParseMutationResultFilter([]string{"unknown"})

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})
}
