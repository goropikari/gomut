package gomut

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type Command struct {
	stdout       io.Writer
	stderr       io.Writer
	jsonlOutput  string
	jsonlEnabled bool
	htmlOutput   string
	htmlEnabled  bool
}

func NewCommand(stdout, stderr io.Writer) *Command {
	return &Command{stdout: stdout, stderr: stderr}
}

func (c *Command) Run(ctx context.Context, args []string) error {
	normalizedArgs, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, err := NormalizeTestArgs(args)
	if err != nil {
		return err
	}

	c.jsonlOutput = jsonlOutput
	c.jsonlEnabled = jsonlEnabled
	c.htmlOutput = htmlOutput
	c.htmlEnabled = htmlEnabled

	root := c.newRootCommand()
	root.SetOut(c.stdout)
	root.SetErr(c.stderr)
	root.SetArgs(normalizedArgs)

	return root.ExecuteContext(ctx)
}

func (c *Command) newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "gomut",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(c.newTestCommand())

	return root
}

func (c *Command) newTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "test",
		Short:         "Run mutation testing",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runTest(cmd, args)
		},
	}

	flags := cmd.Flags()
	flags.String("package", "", "package import path or pattern")
	flags.Bool("all", false, "test all packages")
	flags.String("diff", "", "git diff range or branch name, for example HEAD~1..HEAD or main")
	flags.StringSlice("type", nil, "mutation result types to output")
	flags.Duration("timeout", 10*time.Second, "timeout per mutation")
	flags.Int("parallel", 0, "number of concurrent mutation workers")
	flags.String("config", "", "config file path")
	flags.String("progress", string(ProgressModeAuto), "progress display mode: auto, on, or off")
	flags.String("jsonl", "", "jsonl output file path")
	flags.Lookup("jsonl").NoOptDefVal = ""
	flags.String("html", "", "html output file path")
	flags.Lookup("html").NoOptDefVal = ""

	return cmd
}

func (c *Command) runTest(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(args, " "))
	}

	cfg, err := c.buildTestRunConfig(cmd)
	if err != nil {
		return err
	}

	runner := NewRunner(c.stdout, c.stderr)

	return runner.Run(cmd.Context(), cfg)
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
	jsonlOutput     string
	jsonlEnabled    bool
	htmlOutput      string
	htmlEnabled     bool
	targetChanged   bool
	typeChanged     bool
	timeoutChanged  bool
	config          Config
}

func (c *Command) buildTestRunConfig(cmd *cobra.Command) (RunConfig, error) {
	inputs, err := c.loadTestRunInputs(cmd)
	if err != nil {
		return RunConfig{}, err
	}

	if err := c.applyTestConfigDefaults(&inputs); err != nil {
		return RunConfig{}, err
	}

	resultFilter, err := ParseMutationResultFilter(inputs.resultTypes)
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
		OutputPath:   inputs.jsonlOutput,
		JSONLEnabled: inputs.jsonlEnabled,
		HTMLPath:     inputs.htmlOutput,
		HTMLEnabled:  inputs.htmlEnabled,
		ProgressMode: progressMode,
		ResultFilter: resultFilter,
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

func applyConfigTargetMode(inputs *testRunInputs, mode TargetMode, value *string) error {
	switch mode {
	case TargetModePackage:
		if value == nil || *value == "" {
			return errors.New("config target.value is required when target.mode is package")
		}

		inputs.pkgTarget = *value
	case TargetModeAll:
		inputs.allTarget = true
	case TargetModeDiff:
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

func NormalizeTestArgs(args []string) ([]string, string, bool, string, bool, error) {
	normalized := make([]string, 0, len(args))
	state := normalizedTestArgs{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		consumed, handled := state.consumeOutputFlag(args, i, arg)
		if handled {
			i += consumed
			continue
		}

		normalized = append(normalized, arg)
	}

	return normalized, state.jsonlOutput, state.jsonlEnabled, state.htmlOutput, state.htmlEnabled, nil
}

type normalizedTestArgs struct {
	jsonlOutput  string
	jsonlEnabled bool
	htmlOutput   string
	htmlEnabled  bool
}

func (n *normalizedTestArgs) consumeOutputFlag(args []string, i int, arg string) (int, bool) {
	switch {
	case arg == "--jsonl" || arg == "-jsonl":
		n.jsonlEnabled = true
		output, consumed := consumeFlagValue(args, i)
		n.jsonlOutput = output

		return consumed, true
	case strings.HasPrefix(arg, "--jsonl="):
		n.jsonlEnabled = true
		n.jsonlOutput = strings.TrimPrefix(arg, "--jsonl=")

		return 0, true
	case strings.HasPrefix(arg, "-jsonl="):
		n.jsonlEnabled = true
		n.jsonlOutput = strings.TrimPrefix(arg, "-jsonl=")

		return 0, true
	case arg == "--html" || arg == "-html":
		n.htmlEnabled = true
		output, consumed := consumeFlagValue(args, i)
		n.htmlOutput = output

		return consumed, true
	case strings.HasPrefix(arg, "--html="):
		n.htmlEnabled = true
		n.htmlOutput = strings.TrimPrefix(arg, "--html=")

		return 0, true
	case strings.HasPrefix(arg, "-html="):
		n.htmlEnabled = true
		n.htmlOutput = strings.TrimPrefix(arg, "-html=")

		return 0, true
	default:
		return 0, false
	}
}

func consumeFlagValue(args []string, i int) (string, int) {
	if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
		return args[i+1], 1
	}

	return "", 0
}

func ResolveTarget(pkg string, all bool, diffRange string) (Target, error) {
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
		return Target{}, errors.New("select one target mode with --package, --all, or --diff")
	}

	if selected > 1 {
		return Target{}, errors.New("only one of --package, --all, or --diff can be set")
	}

	switch {
	case pkg != "":
		return Target{Mode: TargetModePackage, Value: pkg}, nil
	case all:
		return Target{Mode: TargetModeAll, Value: "./..."}, nil
	default:
		if diffRange == "" {
			diffRange = "HEAD"
		}

		return Target{Mode: TargetModeDiff, Value: diffRange}, nil
	}
}

func writeJSONL(w io.Writer, record Record) error {
	enc := json.NewEncoder(w)
	return enc.Encode(record)
}

func repoRel(root, path string) string {
	wd := root
	if wd == "" {
		wd, _ = os.Getwd()
	}

	wd, err := filepath.Abs(wd)
	if err != nil {
		return filepath.ToSlash(path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return filepath.ToSlash(path)
	}

	rel, err := filepath.Rel(wd, absPath)
	if err != nil {
		return filepath.ToSlash(path)
	}

	return filepath.ToSlash(rel)
}
