package sample_test

import (
	"testing"

	"gomut/sample"

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

func TestIsAdult(t *testing.T) {
	t.Run("given an adult age, it reports true", func(t *testing.T) {
		// Arrange
		got := sample.IsAdult(18)

		// Assert
		assert.True(t, got)
	})

	t.Run("given a minor age, it reports false", func(t *testing.T) {
		// Arrange
		got := sample.IsAdult(17)

		// Assert
		assert.False(t, got)
	})
}

func TestIsAllowed(t *testing.T) {
	t.Run("given approval, it reports true", func(t *testing.T) {
		// Arrange
		got := sample.IsAllowed(true)

		// Assert
		assert.True(t, got)
	})

	t.Run("given no approval, it reports false", func(t *testing.T) {
		// Arrange
		got := sample.IsAllowed(false)

		// Assert
		assert.False(t, got)
	})
}

func TestHasNickname(t *testing.T) {
	t.Run("given a nickname pointer, it reports true", func(t *testing.T) {
		// Arrange
		nickname := "gopher"

		got := sample.HasNickname(&nickname)

		// Assert
		assert.True(t, got)
	})

	t.Run("given a nil nickname pointer, it reports false", func(t *testing.T) {
		// Arrange
		got := sample.HasNickname(nil)

		// Assert
		assert.False(t, got)
	})
}

func TestAlwaysEnabled(t *testing.T) {
	t.Run("given no input, it returns true", func(t *testing.T) {
		// Arrange
		got := sample.AlwaysEnabled()

		// Assert
		assert.True(t, got)
	})
}

func TestAlwaysDisabled(t *testing.T) {
	t.Run("given no input, it returns false", func(t *testing.T) {
		// Arrange
		got := sample.AlwaysDisabled()

		// Assert
		assert.False(t, got)
	})
}

func TestEnableFlag(t *testing.T) {
	t.Run("given a flag to enable, it returns the combined mask", func(t *testing.T) {
		// Arrange
		got := sample.EnableFlag(0b0001, 0b0100)

		// Assert
		assert.Equal(t, uint8(0b0101), got)
	})
}

func TestKeepCommonBits(t *testing.T) {
	t.Run("given two masks, it keeps only the shared bits", func(t *testing.T) {
		// Arrange
		got := sample.KeepCommonBits(0b1101, 0b1011)

		// Assert
		assert.Equal(t, uint8(0b1001), got)
	})
}

func TestMergeFlags(t *testing.T) {
	t.Run("given two masks, it combines all bits", func(t *testing.T) {
		// Arrange
		got := sample.MergeFlags(0b0101, 0b0011)

		// Assert
		assert.Equal(t, uint8(0b0111), got)
	})
}

func TestClearFlagBits(t *testing.T) {
	t.Run("given a mask and bits to clear, it removes the requested bits", func(t *testing.T) {
		// Arrange
		got := sample.ClearFlagBits(0b1111, 0b0101)

		// Assert
		assert.Equal(t, uint8(0b1010), got)
	})
}

func TestShiftLeft(t *testing.T) {
	t.Run("given a value, it shifts the value left", func(t *testing.T) {
		// Arrange
		got := sample.ShiftLeft(0b0001)

		// Assert
		assert.Equal(t, uint8(0b0010), got)
	})
}

func TestShiftRight(t *testing.T) {
	t.Run("given a value, it shifts the value right", func(t *testing.T) {
		// Arrange
		got := sample.ShiftRight(0b1000)

		// Assert
		assert.Equal(t, uint8(0b0100), got)
	})
}

func TestShiftCounter(t *testing.T) {
	t.Run("given a value, it shifts the value in place", func(t *testing.T) {
		// Arrange
		got := sample.ShiftCounter(0b0011)

		// Assert
		assert.Equal(t, uint8(0b0110), got)
	})
}

func TestNegateScore(t *testing.T) {
	t.Run("given a positive score, it returns the negated score", func(t *testing.T) {
		// Arrange
		got := sample.NegateScore(7)

		// Assert
		assert.Equal(t, -7, got)
	})
}

func TestInvertBits(t *testing.T) {
	t.Run("given a bit mask, it returns the bitwise complement", func(t *testing.T) {
		// Arrange
		got := sample.InvertBits(0b00001111)

		// Assert
		assert.Equal(t, uint8(0b11110000), got)
	})
}

func TestAddBonus(t *testing.T) {
	t.Run("given a score and bonus, it adds the bonus to the score", func(t *testing.T) {
		// Arrange
		got := sample.AddBonus(10, 5)

		// Assert
		assert.Equal(t, 15, got)
	})
}

func TestAdvanceCount(t *testing.T) {
	t.Run("given a counter, it increments the counter", func(t *testing.T) {
		// Arrange
		got := sample.AdvanceCount(3)

		// Assert
		assert.Equal(t, 4, got)
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

func TestIgnoredThreshold(t *testing.T) {
	t.Run("given a low score, it reports true", func(t *testing.T) {
		// Arrange
		got := sample.IgnoredThreshold(49)

		// Assert
		assert.True(t, got)
	})

	t.Run("given a high score, it reports false", func(t *testing.T) {
		// Arrange
		got := sample.IgnoredThreshold(50)

		// Assert
		assert.False(t, got)
	})
}

func TestDefaultRetryLimit(t *testing.T) {
	t.Run("given no input, it returns the configured retry limit", func(t *testing.T) {
		// Arrange
		got := sample.DefaultRetryLimit()

		// Assert
		assert.Equal(t, 3, got)
	})
}

func TestGreeting(t *testing.T) {
	t.Run("given no input, it returns the greeting text", func(t *testing.T) {
		// Arrange
		got := sample.Greeting()

		// Assert
		assert.Equal(t, "hello", got)
	})
}

func TestDefaultTaxRate(t *testing.T) {
	t.Run("given no input, it returns the default tax rate", func(t *testing.T) {
		// Arrange
		got := sample.DefaultTaxRate()

		// Assert
		assert.Equal(t, 0.1, got)
	})
}

func TestDefaultGrade(t *testing.T) {
	t.Run("given no input, it returns the default grade rune", func(t *testing.T) {
		// Arrange
		got := sample.DefaultGrade()

		// Assert
		assert.Equal(t, rune('A'), got)
	})
}

func TestIsBlocked(t *testing.T) {
	t.Run("given approval, it reports false", func(t *testing.T) {
		// Arrange
		got := sample.IsBlocked(true)

		// Assert
		assert.False(t, got)
	})

	t.Run("given no approval, it reports true", func(t *testing.T) {
		// Arrange
		got := sample.IsBlocked(false)

		// Assert
		assert.True(t, got)
	})
}

func TestApprovalLabel(t *testing.T) {
	t.Run("given approval, it returns approved", func(t *testing.T) {
		// Arrange
		got := sample.ApprovalLabel(true)

		// Assert
		assert.Equal(t, "approved", got)
	})

	t.Run("given no approval, it returns blocked", func(t *testing.T) {
		// Arrange
		got := sample.ApprovalLabel(false)

		// Assert
		assert.Equal(t, "blocked", got)
	})
}

func TestConstVariable(t *testing.T) {
	t.Run("sample", func(t *testing.T) {
		assert.True(t, sample.ConstVariable())
	})
}
