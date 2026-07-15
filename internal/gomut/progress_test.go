package gomut_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/goropikari/gomut/internal/gomut"
)

func TestNewProgressReporter(t *testing.T) {
	t.Run("given progress mode on, it emits mutation progress updates", func(t *testing.T) {
		// Arrange
		var stderr bytes.Buffer

		reporter := gomut.NewProgressReporter(gomut.ProgressConfig{
			Mode:        gomut.ProgressModeOn,
			Writer:      &stderr,
			Interactive: true,
			Total:       3,
		})

		// Act
		require.True(t, reporter.Enabled())
		reporter.Start(3)
		reporter.Update(1)
		reporter.Update(2)
		reporter.Update(3)
		reporter.Finish()

		// Assert
		output := stderr.String()
		assert.Contains(t, output, "Progress")
		assert.Contains(t, output, "1/3")
		assert.Contains(t, output, "3/3")
	})

	t.Run("given auto mode in a non-interactive run, it stays quiet", func(t *testing.T) {
		// Arrange
		var stderr bytes.Buffer

		reporter := gomut.NewProgressReporter(gomut.ProgressConfig{
			Mode:        gomut.ProgressModeAuto,
			Writer:      &stderr,
			Interactive: false,
			CI:          false,
			Total:       3,
		})

		// Act
		require.False(t, reporter.Enabled())
		reporter.Start(3)
		reporter.Update(1)
		reporter.Update(2)
		reporter.Finish()

		// Assert
		assert.Empty(t, stderr.String())
	})

	t.Run("given auto mode in CI, it stays quiet", func(t *testing.T) {
		// Arrange
		var stderr bytes.Buffer

		reporter := gomut.NewProgressReporter(gomut.ProgressConfig{
			Mode:        gomut.ProgressModeAuto,
			Writer:      &stderr,
			Interactive: true,
			CI:          true,
			Total:       3,
		})

		// Act
		require.False(t, reporter.Enabled())
		reporter.Start(3)
		reporter.Update(1)
		reporter.Update(2)
		reporter.Finish()

		// Assert
		assert.Empty(t, stderr.String())
	})

	t.Run("given progress mode off, it stays quiet", func(t *testing.T) {
		// Arrange
		var stderr bytes.Buffer

		reporter := gomut.NewProgressReporter(gomut.ProgressConfig{
			Mode:        gomut.ProgressModeOff,
			Writer:      &stderr,
			Interactive: true,
			Total:       3,
		})

		// Act
		require.False(t, reporter.Enabled())
		reporter.Start(3)
		reporter.Update(1)
		reporter.Update(2)
		reporter.Finish()

		// Assert
		assert.Empty(t, stderr.String())
	})
}
