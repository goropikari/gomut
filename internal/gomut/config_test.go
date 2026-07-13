package gomut_test

import (
	"gomut/internal/gomut/result"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gomut "gomut/internal/gomut"
)

func TestDefaultConfigPath(t *testing.T) {
	t.Run("given a repository root, it returns the default config file path", func(t *testing.T) {
		// Arrange
		root := filepath.Join("work", "repo")

		// Act
		path := gomut.DefaultConfigPath(root)

		// Assert
		assert.Equal(t, filepath.Join(root, gomut.DefaultConfigFileName), path)
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("given a valid config file, it parses the supported fields", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		path := filepath.Join(dir, ".gomut.yaml")
		require.NoError(t, os.WriteFile(path, []byte(`target:
  mode: package
  value: ./sample
timeout: 15s
progress: on
jsonl: mutations.jsonl
html: report.html
type:
  - lived
kind:
  - comparison_operator
  - return
parallel: 3
exclude:
  - internal/generated
baseline:
  input: baseline-in.jsonl
  output: baseline-out.jsonl
`), 0o600))

		// Act
		cfg, err := gomut.LoadConfig(path)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, cfg.Target)
		require.NotNil(t, cfg.Target.Mode)
		require.NotNil(t, cfg.Target.Value)
		assert.Equal(t, result.TargetModePackage, *cfg.Target.Mode)
		assert.Equal(t, "./sample", *cfg.Target.Value)
		require.NotNil(t, cfg.Timeout)
		assert.Equal(t, "15s", *cfg.Timeout)
		require.NotNil(t, cfg.Progress)
		assert.Equal(t, "on", *cfg.Progress)
		require.NotNil(t, cfg.JSONL)
		assert.Equal(t, "mutations.jsonl", *cfg.JSONL)
		require.NotNil(t, cfg.HTML)
		assert.Equal(t, "report.html", *cfg.HTML)
		require.Len(t, cfg.Type, 1)
		assert.Equal(t, "lived", cfg.Type[0])
		require.Len(t, cfg.Kind, 2)
		assert.Equal(t, []string{"comparison_operator", "return"}, []string(cfg.Kind))
		require.NotNil(t, cfg.Parallel)
		assert.Equal(t, 3, *cfg.Parallel)
		assert.Equal(t, []string{"internal/generated"}, cfg.Exclude)
		require.NotNil(t, cfg.Baseline)
		require.NotNil(t, cfg.Baseline.Input)
		require.NotNil(t, cfg.Baseline.Output)
		assert.Equal(t, "baseline-in.jsonl", *cfg.Baseline.Input)
		assert.Equal(t, "baseline-out.jsonl", *cfg.Baseline.Output)
	})

	t.Run("given a malformed config file, it returns an error", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		path := filepath.Join(dir, ".gomut.yaml")
		require.NoError(t, os.WriteFile(path, []byte("target:\n  mode: [\n"), 0o600))

		// Act
		_, err := gomut.LoadConfig(path)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), ".gomut.yaml")
	})

	t.Run("given a scalar kind value, it parses the value as a single-item list", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		path := filepath.Join(dir, ".gomut.yaml")
		require.NoError(t, os.WriteFile(path, []byte(`kind: comparison_operator
`), 0o600))

		// Act
		cfg, err := gomut.LoadConfig(path)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"comparison_operator"}, []string(cfg.Kind))
	})
}
