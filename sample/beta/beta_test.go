package beta

import "testing"

func TestClampScore(t *testing.T) {
	t.Run("given a score below zero, it clamps to zero", func(t *testing.T) {
		// Arrange
		got := ClampScore(-5)

		// Assert
		if got != 0 {
			t.Fatalf("expected zero, got %d", got)
		}
	})

	t.Run("given a score above the upper bound, it clamps to 100", func(t *testing.T) {
		// Arrange
		got := ClampScore(120)

		// Assert
		if got != 100 {
			t.Fatalf("expected 100, got %d", got)
		}
	})
}

func TestCanPublish(t *testing.T) {
	t.Run("given an adult with approval, it allows publishing", func(t *testing.T) {
		// Arrange
		got := CanPublish(20, true)

		// Assert
		if !got {
			t.Fatal("expected publishing to be allowed")
		}
	})
}
