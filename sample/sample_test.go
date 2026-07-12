package sample_test

import (
	"gomut/sample"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClampScore(t *testing.T) {
	t.Run("given a score below zero, it clamps to zero", func(t *testing.T) {
		// Arrange
		got := sample.ClampScore(-5)

		// Assert
		assert.Equal(t, 0, got)
	})

	t.Run("given a score at the upper bound, it stays unchanged", func(t *testing.T) {
		// Arrange
		got := sample.ClampScore(100)

		// Assert
		assert.Equal(t, 100, got)
	})
}

func TestCanPublish(t *testing.T) {
	t.Run("given an adult with approval, it allows publishing", func(t *testing.T) {
		// Arrange
		got := sample.CanPublish(20, true)

		// Assert
		assert.True(t, got)
	})

	t.Run("given a minor with approval, it blocks publishing", func(t *testing.T) {
		// Arrange
		got := sample.CanPublish(17, true)

		// Assert
		assert.False(t, got)
	})

	t.Run("given an adult without approval, it blocks publishing", func(t *testing.T) {
		// Arrange
		got := sample.CanPublish(20, false)

		// Assert
		assert.False(t, got)
	})
}

func TestValidateQuantity(t *testing.T) {
	t.Run("given a negative quantity, it returns an error", func(t *testing.T) {
		// Arrange
		err := sample.ValidateQuantity(-1)

		// Assert
		require.Error(t, err)
		assert.EqualError(t, err, "quantity must be non-negative")
	})

	t.Run("given a positive quantity, it returns nil", func(t *testing.T) {
		// Arrange
		err := sample.ValidateQuantity(1)

		// Assert
		require.NoError(t, err)
	})
}
