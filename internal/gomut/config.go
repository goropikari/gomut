package gomut

import (
	"bytes"
	"errors"
	"fmt"
	"gomut/internal/gomut/result"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const DefaultConfigFileName = ".gomut.yaml"

// Config represents gomut settings loaded from a YAML config file.
type Config struct {
	Target   *ConfigTarget   `yaml:"target,omitempty"`
	Timeout  *string         `yaml:"timeout,omitempty"`
	Progress *string         `yaml:"progress,omitempty"`
	JSONL    *string         `yaml:"jsonl,omitempty"`
	HTML     *string         `yaml:"html,omitempty"`
	Type     []string        `yaml:"type,omitempty"`
	Kind     yamlStringList  `yaml:"kind,omitempty"`
	Parallel *int            `yaml:"parallel,omitempty"`
	Exclude  []string        `yaml:"exclude,omitempty"`
	Baseline *BaselineConfig `yaml:"baseline,omitempty"`
}

type yamlStringList []string

func (s *yamlStringList) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		if strings.TrimSpace(value.Value) == "" {
			*s = nil
			return nil
		}

		*s = []string{value.Value}

		return nil
	case yaml.SequenceNode:
		values := make([]string, 0, len(value.Content))
		for _, node := range value.Content {
			values = append(values, node.Value)
		}

		*s = values

		return nil
	case 0:
		*s = nil
		return nil
	default:
		return fmt.Errorf("kind must be a string or sequence")
	}
}

// ConfigTarget represents the target selection block in the config file.
type ConfigTarget struct {
	Mode  *result.TargetMode `yaml:"mode,omitempty"`
	Value *string            `yaml:"value,omitempty"`
}

// BaselineConfig represents baseline comparison paths in the config file.
type BaselineConfig struct {
	Input  *string `yaml:"input,omitempty"`
	Output *string `yaml:"output,omitempty"`
}

// RunConfig captures the resolved runtime settings used by the mutation runner.
type RunConfig struct {
	Target       result.Target
	Timeout      time.Duration
	Parallel     int
	Exclude      []string
	KindFilter   result.MutationKindFilter
	OutputPath   string
	JSONLEnabled bool
	HTMLPath     string
	HTMLEnabled  bool
	ProgressMode ProgressMode
	ResultFilter result.MutationResultFilter
	Verbose      bool
}

type testRunInputs struct {
	pkgTarget       string
	allTarget       bool
	diffRange       string
	resultTypes     []string
	timeout         time.Duration
	parallel        int
	parallelChanged bool
	progressMode    string
	progressChanged bool
	verbose         bool
	kind            []string
	kindChanged     bool
	jsonlOutput     string
	jsonlEnabled    bool
	htmlOutput      string
	htmlEnabled     bool
	targetChanged   bool
	typeChanged     bool
	timeoutChanged  bool
	config          Config
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

func (c *Command) loadTestConfig(cmd *cobra.Command) (Config, error) {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath != "" {
		return LoadConfig(configPath)
	}

	root, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}

	defaultPath := DefaultConfigPath(root)
	if _, statErr := os.Stat(defaultPath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return Config{}, nil
		}

		return Config{}, fmt.Errorf("check config %s: %w", defaultPath, statErr)
	}

	return LoadConfig(defaultPath)
}

func (c *Command) buildTestRunConfig(cmd *cobra.Command) (RunConfig, error) {
	inputs, err := c.loadTestRunInputs(cmd)
	if err != nil {
		return RunConfig{}, err
	}

	if err := c.applyTestConfigDefaults(&inputs); err != nil {
		return RunConfig{}, err
	}

	resultFilter, err := result.ParseMutationResultFilter(inputs.resultTypes)
	if err != nil {
		return RunConfig{}, err
	}

	kindFilter, err := result.ParseMutationKindFilter(inputs.kind)
	if err != nil {
		return RunConfig{}, err
	}

	target, err := ResolveTarget(inputs.pkgTarget, inputs.allTarget, inputs.diffRange)
	if err != nil {
		return RunConfig{}, err
	}

	progressMode, err := parseProgressMode(inputs.progressMode)
	if err != nil {
		return RunConfig{}, err
	}

	return RunConfig{
		Target:       target,
		Timeout:      inputs.timeout,
		Parallel:     inputs.parallel,
		Exclude:      append([]string(nil), inputs.config.Exclude...),
		KindFilter:   kindFilter,
		OutputPath:   inputs.jsonlOutput,
		JSONLEnabled: inputs.jsonlEnabled,
		HTMLPath:     inputs.htmlOutput,
		HTMLEnabled:  inputs.htmlEnabled,
		ProgressMode: progressMode,
		ResultFilter: resultFilter,
		Verbose:      inputs.verbose,
	}, nil
}

func (c *Command) loadTestRunInputs(cmd *cobra.Command) (testRunInputs, error) {
	config, err := c.loadTestConfig(cmd)
	if err != nil {
		return testRunInputs{}, err
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")
	parallel, _ := cmd.Flags().GetInt("parallel")
	pkgTarget, _ := cmd.Flags().GetString("package")
	allTarget, _ := cmd.Flags().GetBool("all")
	diffRange, _ := cmd.Flags().GetString("diff")
	resultTypes, _ := cmd.Flags().GetStringSlice("type")
	progressMode, _ := cmd.Flags().GetString("progress")
	verbose, _ := cmd.Flags().GetBool("verbose")
	kind, _ := cmd.Flags().GetStringSlice("kind")

	return testRunInputs{
		pkgTarget:       pkgTarget,
		allTarget:       allTarget,
		diffRange:       diffRange,
		resultTypes:     append([]string(nil), resultTypes...),
		timeout:         timeout,
		parallel:        parallel,
		parallelChanged: cmd.Flags().Changed("parallel"),
		progressMode:    progressMode,
		progressChanged: cmd.Flags().Changed("progress"),
		verbose:         verbose,
		kind:            append([]string(nil), kind...),
		kindChanged:     cmd.Flags().Changed("kind"),
		jsonlOutput:     c.jsonlOutput,
		jsonlEnabled:    c.jsonlEnabled,
		htmlOutput:      c.htmlOutput,
		htmlEnabled:     c.htmlEnabled,
		targetChanged:   cmd.Flags().Changed("package") || cmd.Flags().Changed("all") || cmd.Flags().Changed("diff"),
		typeChanged:     cmd.Flags().Changed("type"),
		timeoutChanged:  cmd.Flags().Changed("timeout"),
		config:          config,
	}, nil
}

func (c *Command) applyTestConfigDefaults(inputs *testRunInputs) error {
	if err := c.applyTargetConfigDefaults(inputs); err != nil {
		return err
	}

	c.applyResultTypeConfigDefaults(inputs)
	c.applyKindConfigDefaults(inputs)

	if err := c.applyTimeoutConfigDefaults(inputs); err != nil {
		return err
	}

	c.applyParallelConfigDefaults(inputs)
	c.applyProgressConfigDefaults(inputs)
	c.applyOutputConfigDefaults(inputs)

	return nil
}

func (c *Command) applyTargetConfigDefaults(inputs *testRunInputs) error {
	if inputs.targetChanged || inputs.config.Target == nil || inputs.config.Target.Mode == nil {
		return nil
	}

	return applyConfigTargetMode(inputs, *inputs.config.Target.Mode, inputs.config.Target.Value)
}

func (c *Command) applyResultTypeConfigDefaults(inputs *testRunInputs) {
	if inputs.typeChanged || len(inputs.config.Type) == 0 {
		return
	}

	inputs.resultTypes = append([]string(nil), inputs.config.Type...)
}

func (c *Command) applyKindConfigDefaults(inputs *testRunInputs) {
	if inputs.kindChanged || len(inputs.config.Kind) == 0 {
		return
	}

	inputs.kind = append([]string(nil), inputs.config.Kind...)
}

func (c *Command) applyTimeoutConfigDefaults(inputs *testRunInputs) error {
	if inputs.timeoutChanged || inputs.config.Timeout == nil {
		return nil
	}

	timeout, err := time.ParseDuration(*inputs.config.Timeout)
	if err != nil {
		return fmt.Errorf("parse config timeout: %w", err)
	}

	inputs.timeout = timeout

	return nil
}

func (c *Command) applyParallelConfigDefaults(inputs *testRunInputs) {
	if !inputs.parallelChanged && inputs.config.Parallel != nil {
		inputs.parallel = *inputs.config.Parallel
	}

	if inputs.parallel <= 0 {
		inputs.parallel = runtime.NumCPU()
	}
}

func (c *Command) applyProgressConfigDefaults(inputs *testRunInputs) {
	if inputs.progressChanged || inputs.config.Progress == nil {
		return
	}

	inputs.progressMode = *inputs.config.Progress
}

func (c *Command) applyOutputConfigDefaults(inputs *testRunInputs) {
	if !inputs.jsonlEnabled && inputs.config.JSONL != nil {
		inputs.jsonlOutput = *inputs.config.JSONL
		inputs.jsonlEnabled = true
	}

	if !inputs.htmlEnabled && inputs.config.HTML != nil {
		inputs.htmlOutput = *inputs.config.HTML
		inputs.htmlEnabled = true
	}
}

func applyConfigTargetMode(inputs *testRunInputs, mode result.TargetMode, value *string) error {
	switch mode {
	case result.TargetModePackage:
		if value == nil || *value == "" {
			return errors.New("config target.value is required when target.mode is package")
		}

		inputs.pkgTarget = *value
	case result.TargetModeAll:
		inputs.allTarget = true
	case result.TargetModeDiff:
		inputs.diffRange = configDiffRange(value)
	default:
		return fmt.Errorf("unknown config target mode: %s", mode)
	}

	return nil
}

func configDiffRange(value *string) string {
	if value != nil && *value != "" {
		return *value
	}

	return "HEAD"
}

func parseProgressMode(value string) (ProgressMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(ProgressModeAuto):
		return ProgressModeAuto, nil
	case string(ProgressModeOn):
		return ProgressModeOn, nil
	case string(ProgressModeOff):
		return ProgressModeOff, nil
	default:
		return "", fmt.Errorf("unknown progress mode: %s", value)
	}
}

func ResolveTarget(pkg string, all bool, diffRange string) (result.Target, error) {
	selected := 0
	if pkg != "" {
		selected++
	}

	if all {
		selected++
	}

	if diffRange != "" {
		selected++
	}

	if selected == 0 {
		return result.Target{}, errors.New("select one target mode with --package, --all, or --diff")
	}

	if selected > 1 {
		return result.Target{}, errors.New("only one of --package, --all, or --diff can be set")
	}

	switch {
	case pkg != "":
		return result.Target{Mode: result.TargetModePackage, Value: pkg}, nil
	case all:
		return result.Target{Mode: result.TargetModeAll, Value: "./..."}, nil
	default:
		if diffRange == "" {
			diffRange = "HEAD"
		}

		return result.Target{Mode: result.TargetModeDiff, Value: diffRange}, nil
	}
}
