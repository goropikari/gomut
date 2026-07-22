package gomut

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/goropikari/gomut/internal/gomut/result"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const DefaultConfigFileName = ".gomut.yaml"

const userConfigFileName = "config.yaml"

// Config represents gomut settings loaded from a YAML config file.
type Config struct {
	Target    *ConfigTarget    `yaml:"target,omitempty"`
	Timeout   *string          `yaml:"timeout,omitempty"`
	Progress  *string          `yaml:"progress,omitempty"`
	JSONL     *string          `yaml:"jsonl,omitempty"`
	HTML      *string          `yaml:"html,omitempty"`
	SARIF     *string          `yaml:"sarif,omitempty"`
	Type      []string         `yaml:"type,omitempty"`
	Kind      *KindConfig      `yaml:"kind,omitempty"`
	Parallel  *int             `yaml:"parallel,omitempty"`
	Exclude   []string         `yaml:"exclude,omitempty"`
	Isolation *IsolationConfig `yaml:"isolation,omitempty"`
	Baseline  *BaselineConfig  `yaml:"baseline,omitempty"`
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

// KindConfig represents the mutation kind selection block in the config file.
type KindConfig struct {
	Mode    *string        `yaml:"mode,omitempty"`
	Enable  yamlStringList `yaml:"enable,omitempty"`
	Disable yamlStringList `yaml:"disable,omitempty"`
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

// IsolationConfig represents temporary repository copy settings.
type IsolationConfig struct {
	CopyExclude yamlStringList `yaml:"copy_exclude,omitempty"`
}

// RunConfig captures the resolved runtime settings used by the mutation runner.
type RunConfig struct {
	Target               result.Target
	Timeout              time.Duration
	Parallel             int
	Exclude              []string
	IsolationCopyExclude []string
	KindFilter           result.MutationKindFilter
	OutputPath           string
	JSONLEnabled         bool
	HTMLPath             string
	HTMLEnabled          bool
	SARIFPath            string
	SARIFEnabled         bool
	ProgressMode         ProgressMode
	ResultFilter         result.MutationResultFilter
	Verbose              bool
}

type testRunInputs struct {
	targetArg          string
	diffRange          string
	resultTypes        []string
	timeout            time.Duration
	parallel           int
	parallelChanged    bool
	progressMode       string
	progressChanged    bool
	verbose            bool
	exclude            []string
	excludeChanged     bool
	kindMode           string
	kindModeChanged    bool
	kindEnable         []string
	kindEnableChanged  bool
	kindDisable        []string
	kindDisableChanged bool
	jsonlOutput        string
	jsonlEnabled       bool
	htmlOutput         string
	htmlEnabled        bool
	sarifOutput        string
	sarifEnabled       bool
	targetChanged      bool
	typeChanged        bool
	timeoutChanged     bool
	config             Config
}

// DefaultConfigPath returns the default gomut config file path for a root directory.
func DefaultConfigPath(root string) string {
	return filepath.Join(root, DefaultConfigFileName)
}

func userConfigPath(home string) string {
	return filepath.Join(home, ".gomut", userConfigFileName)
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

	configPaths := []string{DefaultConfigPath(root)}
	if home, homeErr := os.UserHomeDir(); homeErr == nil {
		configPaths = append(configPaths, userConfigPath(home))
	}

	for _, configPath := range configPaths {
		if _, statErr := os.Stat(configPath); statErr == nil {
			return LoadConfig(configPath)
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return Config{}, fmt.Errorf("check config %s: %w", configPath, statErr)
		}
	}

	return Config{}, nil
}

func (c *Command) buildTestRunConfig(cmd *cobra.Command, args ...string) (RunConfig, error) {
	inputs, err := c.loadTestRunInputs(cmd, args)
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

	kindFilter, err := result.ParseMutationKindFilter(inputs.kindMode, inputs.kindEnable, inputs.kindDisable)
	if err != nil {
		return RunConfig{}, err
	}

	target, err := ResolveTarget(inputs.targetArg, inputs.diffRange)
	if err != nil {
		return RunConfig{}, err
	}

	progressMode, err := parseProgressMode(inputs.progressMode)
	if err != nil {
		return RunConfig{}, err
	}

	return RunConfig{
		Target:               target,
		Timeout:              inputs.timeout,
		Parallel:             inputs.parallel,
		Exclude:              append([]string(nil), inputs.exclude...),
		IsolationCopyExclude: isolationCopyExclude(inputs.config),
		KindFilter:           kindFilter,
		OutputPath:           inputs.jsonlOutput,
		JSONLEnabled:         inputs.jsonlEnabled,
		HTMLPath:             inputs.htmlOutput,
		HTMLEnabled:          inputs.htmlEnabled,
		SARIFPath:            inputs.sarifOutput,
		SARIFEnabled:         inputs.sarifEnabled,
		ProgressMode:         progressMode,
		ResultFilter:         resultFilter,
		Verbose:              inputs.verbose,
	}, nil
}

func (c *Command) loadTestRunInputs(cmd *cobra.Command, args []string) (testRunInputs, error) {
	config, err := c.loadTestConfig(cmd)
	if err != nil {
		return testRunInputs{}, err
	}

	if len(args) > 1 {
		return testRunInputs{}, fmt.Errorf("unexpected arguments: %s", strings.Join(args[1:], " "))
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")
	parallel, _ := cmd.Flags().GetInt("parallel")
	diffRange, _ := cmd.Flags().GetString("diff")
	resultTypes, _ := cmd.Flags().GetStringSlice("type")
	exclude, _ := cmd.Flags().GetStringSlice("exclude")
	progressMode, _ := cmd.Flags().GetString("progress")
	verbose, _ := cmd.Flags().GetBool("verbose")
	kindMode, _ := cmd.Flags().GetString("mode")
	kindEnable, _ := cmd.Flags().GetStringSlice("enable")
	kindDisable, _ := cmd.Flags().GetStringSlice("disable")

	targetArg := ""
	if len(args) == 1 {
		targetArg = args[0]
	}

	return testRunInputs{
		targetArg:          targetArg,
		diffRange:          diffRange,
		resultTypes:        append([]string(nil), resultTypes...),
		timeout:            timeout,
		parallel:           parallel,
		parallelChanged:    cmd.Flags().Changed("parallel"),
		progressMode:       progressMode,
		progressChanged:    cmd.Flags().Changed("progress"),
		verbose:            verbose,
		exclude:            append([]string(nil), exclude...),
		excludeChanged:     cmd.Flags().Changed("exclude"),
		kindMode:           kindMode,
		kindModeChanged:    cmd.Flags().Changed("mode"),
		kindEnable:         append([]string(nil), kindEnable...),
		kindEnableChanged:  cmd.Flags().Changed("enable"),
		kindDisable:        append([]string(nil), kindDisable...),
		kindDisableChanged: cmd.Flags().Changed("disable"),
		jsonlOutput:        c.jsonlOutput,
		jsonlEnabled:       c.jsonlEnabled,
		htmlOutput:         c.htmlOutput,
		htmlEnabled:        c.htmlEnabled,
		sarifOutput:        c.sarifOutput,
		sarifEnabled:       c.sarifEnabled,
		targetChanged:      len(args) > 0 || cmd.Flags().Changed("diff"),
		typeChanged:        cmd.Flags().Changed("type"),
		timeoutChanged:     cmd.Flags().Changed("timeout"),
		config:             config,
	}, nil
}

func (c *Command) applyTestConfigDefaults(inputs *testRunInputs) error {
	if err := c.applyTargetConfigDefaults(inputs); err != nil {
		return err
	}

	c.applyResultTypeConfigDefaults(inputs)
	c.applyExcludeConfigDefaults(inputs)
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

func (c *Command) applyExcludeConfigDefaults(inputs *testRunInputs) {
	if inputs.excludeChanged {
		inputs.exclude = append(append([]string(nil), inputs.config.Exclude...), inputs.exclude...)
		return
	}

	if len(inputs.config.Exclude) == 0 {
		return
	}

	inputs.exclude = append([]string(nil), inputs.config.Exclude...)
}

func (c *Command) applyKindConfigDefaults(inputs *testRunInputs) {
	kindConfig := inputs.config.Kind
	baseMode := string(result.MutationKindModeStandard)
	configEnable := []string(nil)
	configDisable := []string(nil)

	if kindConfig != nil {
		if kindConfig.Mode != nil {
			baseMode = *kindConfig.Mode
		}

		configEnable = append([]string(nil), kindConfig.Enable...)
		configDisable = append([]string(nil), kindConfig.Disable...)
	}

	finalMode := baseMode
	if inputs.kindModeChanged {
		finalMode = inputs.kindMode
	}

	finalEnable := configEnable
	if inputs.kindEnableChanged {
		finalEnable = append([]string(nil), inputs.kindEnable...)

		finalDisable := configDisable
		if len(finalDisable) > 0 {
			finalDisable = removeStringValues(finalDisable, finalEnable)
		}

		configDisable = finalDisable
	}

	finalDisable := configDisable
	if inputs.kindDisableChanged {
		finalDisable = append([]string(nil), inputs.kindDisable...)
		if len(finalEnable) > 0 {
			finalEnable = removeStringValues(finalEnable, finalDisable)
		}
	}

	inputs.kindMode = finalMode
	inputs.kindEnable = finalEnable
	inputs.kindDisable = finalDisable
}

func removeStringValues(values, excluded []string) []string {
	if len(values) == 0 || len(excluded) == 0 {
		return append([]string(nil), values...)
	}

	excludedSet := make(map[string]struct{}, len(excluded))
	for _, value := range excluded {
		excludedSet[value] = struct{}{}
	}

	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := excludedSet[value]; ok {
			continue
		}

		filtered = append(filtered, value)
	}

	return filtered
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

	if !inputs.sarifEnabled && inputs.config.SARIF != nil {
		inputs.sarifOutput = *inputs.config.SARIF
		inputs.sarifEnabled = true
	}
}

func isolationCopyExclude(cfg Config) []string {
	if cfg.Isolation == nil {
		return nil
	}

	return append([]string(nil), cfg.Isolation.CopyExclude...)
}

func applyConfigTargetMode(inputs *testRunInputs, mode result.TargetMode, value *string) error {
	switch mode {
	case result.TargetModePackage:
		if value == nil || *value == "" {
			return errors.New("config target.value is required when target.mode is package")
		}

		inputs.targetArg = *value
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

func ResolveTarget(targetArg string, diffRange string) (result.Target, error) {
	if targetArg != "" && diffRange != "" {
		return result.Target{}, errors.New("use either a positional target or --diff, not both")
	}

	if diffRange != "" {
		return result.Target{Mode: result.TargetModeDiff, Value: diffRange}, nil
	}

	if targetArg == "" {
		return result.Target{}, errors.New("target is required. use `gomut test ./...` or `gomut test --diff <range>`")
	}

	if targetArg == "./..." {
		return result.Target{Mode: result.TargetModePackage, Value: "./..."}, nil
	}

	return result.Target{Mode: result.TargetModePackage, Value: targetArg}, nil
}
