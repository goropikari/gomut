package gomut

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultConfigFileName = ".gomut.yaml"

// Config represents gomut settings loaded from a YAML config file.
type Config struct {
	Target   *ConfigTarget   `yaml:"target,omitempty"`
	Timeout  *string         `yaml:"timeout,omitempty"`
	JSONL    *string         `yaml:"jsonl,omitempty"`
	HTML     *string         `yaml:"html,omitempty"`
	Type     []string        `yaml:"type,omitempty"`
	Parallel *int            `yaml:"parallel,omitempty"`
	Exclude  []string        `yaml:"exclude,omitempty"`
	Baseline *BaselineConfig `yaml:"baseline,omitempty"`
}

// ConfigTarget represents the target selection block in the config file.
type ConfigTarget struct {
	Mode  *TargetMode `yaml:"mode,omitempty"`
	Value *string     `yaml:"value,omitempty"`
}

// BaselineConfig represents baseline comparison paths in the config file.
type BaselineConfig struct {
	Input  *string `yaml:"input,omitempty"`
	Output *string `yaml:"output,omitempty"`
}

// DefaultConfigPath returns the default gomut config file path for a root directory.
func DefaultConfigPath(root string) string {
	return filepath.Join(root, DefaultConfigFileName)
}

// LoadConfig reads and parses a gomut config file from path.
func LoadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}, errors.New("config path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("load config %s: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("load config %s: %w", path, err)
	}

	return cfg, nil
}
