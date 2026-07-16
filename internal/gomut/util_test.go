package gomut_test

import (
	"strings"
	"testing"

	"github.com/goropikari/gomut/internal/gomut"
	"github.com/stretchr/testify/assert"
)

func TestGoCommandEnv(t *testing.T) {
	t.Run("given no buildvcs flag, it disables VCS stamping", func(t *testing.T) {
		// Arrange
		t.Setenv("GOFLAGS", "-count=1")

		// Act
		env := gomut.GoCommandEnv()

		// Assert
		assert.Contains(t, envValue(env, "GOFLAGS"), "-count=1")
		assert.Contains(t, envValue(env, "GOFLAGS"), "-buildvcs=false")
	})

	t.Run("given buildvcs is already configured, it keeps the existing value", func(t *testing.T) {
		// Arrange
		t.Setenv("GOFLAGS", "-buildvcs=true")

		// Act
		env := gomut.GoCommandEnv()

		// Assert
		assert.Equal(t, "-buildvcs=true", envValue(env, "GOFLAGS"))
	})
}

func envValue(env []string, key string) string {
	prefix := key + "="

	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			return strings.TrimPrefix(item, prefix)
		}
	}

	return ""
}
