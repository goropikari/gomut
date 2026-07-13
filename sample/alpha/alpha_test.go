package alpha

import "testing"

func TestIsAdult(t *testing.T) {
	t.Run("given an adult age, it reports true", func(t *testing.T) {
		// Arrange
		got := IsAdult(18)

		// Assert
		if !got {
			t.Fatal("expected adult age to be accepted")
		}
	})

	t.Run("given a minor age, it reports false", func(t *testing.T) {
		// Arrange
		got := IsAdult(17)

		// Assert
		if got {
			t.Fatal("expected minor age to be rejected")
		}
	})
}

func TestDouble(t *testing.T) {
	t.Run("given a value, it doubles the input", func(t *testing.T) {
		// Arrange
		got := Double(3)

		// Assert
		if got != 6 {
			t.Fatalf("expected double to be 6, got %d", got)
		}
	})
}
