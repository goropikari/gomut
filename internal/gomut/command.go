package gomut

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/goropikari/gomut/internal/gomut/result"

	"github.com/spf13/cobra"
)

type Command struct {
	stdout       io.Writer
	stderr       io.Writer
	jsonlOutput  string
	jsonlEnabled bool
	htmlOutput   string
	htmlEnabled  bool
	sarifOutput  string
	sarifEnabled bool
}

func NewCommand(stdout, stderr io.Writer) *Command {
	return &Command{stdout: stdout, stderr: stderr}
}

func (c *Command) Run(ctx context.Context, args []string) error {
	normalizedArgs, jsonlOutput, jsonlEnabled, htmlOutput, htmlEnabled, sarifOutput, sarifEnabled, err := NormalizeTestArgs(args)
	if err != nil {
		return err
	}

	c.jsonlOutput = jsonlOutput
	c.jsonlEnabled = jsonlEnabled
	c.htmlOutput = htmlOutput
	c.htmlEnabled = htmlEnabled
	c.sarifOutput = sarifOutput
	c.sarifEnabled = sarifEnabled

	root := c.newRootCommand()
	if !isDiffInvocation(normalizedArgs) && !containsHelpFlag(normalizedArgs) {
		root.RemoveCommand(root.Commands()...)
	}

	root.SetOut(c.stdout)
	root.SetErr(c.stderr)
	root.SetArgs(normalizedArgs)

	return root.ExecuteContext(ctx)
}

func isDiffInvocation(args []string) bool {
	return len(args) > 0 && args[0] == "diff"
}

func containsHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}

	return false
}

func (c *Command) newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "gomut <target>",
		Short:         "Run mutation testing",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runTest(cmd, args)
		},
	}

	c.addRunFlags(root)

	diff := &cobra.Command{
		Use:           "diff [range]",
		Short:         "Run mutation testing for changed files",
		Example:       "  gomut diff HEAD~1..HEAD\n  gomut diff main...HEAD\n  gomut diff main",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runTest(cmd, args)
		},
	}
	c.addRunFlags(diff)
	root.AddCommand(diff)

	return root
}

func (c *Command) addRunFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringSlice("type", nil, "mutation result types to output")
	flags.String("mode", string(result.MutationKindModeStandard), "mutation kind mode: standard or all")
	flags.StringSlice("enable", nil, "additional mutation kinds to enable")
	flags.StringSlice("disable", nil, "mutation kinds to disable")
	flags.StringSlice("exclude", nil, "mutation candidate file patterns to exclude")
	flags.Duration("timeout", 10*time.Second, "timeout per mutation")
	flags.Int("parallel", 0, "number of concurrent mutation workers")
	flags.String("config", "", "config file path")
	flags.String("progress", string(ProgressModeAuto), "progress display mode: auto, on, or off")
	flags.Bool("verbose", false, "show exclusion notices on stderr")
	flags.String("jsonl", "", "jsonl output file path")
	flags.Lookup("jsonl").NoOptDefVal = ""
	flags.String("html", "", "html output file path")
	flags.Lookup("html").NoOptDefVal = ""
	flags.String("sarif", "", "sarif output file path")
	flags.Lookup("sarif").NoOptDefVal = ""
}

func (c *Command) runTest(cmd *cobra.Command, args []string) error {
	cfg, err := c.buildTestRunConfig(cmd, args...)
	if err != nil {
		return err
	}

	runner := NewRunner(c.stdout, c.stderr)

	return runner.Run(cmd.Context(), cfg)
}

func NormalizeTestArgs(args []string) ([]string, string, bool, string, bool, string, bool, error) {
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

	return normalized, state.jsonlOutput, state.jsonlEnabled, state.htmlOutput, state.htmlEnabled, state.sarifOutput, state.sarifEnabled, nil
}

type normalizedTestArgs struct {
	jsonlOutput  string
	jsonlEnabled bool
	htmlOutput   string
	htmlEnabled  bool
	sarifOutput  string
	sarifEnabled bool
}

func (n *normalizedTestArgs) consumeOutputFlag(args []string, i int, arg string) (int, bool) {
	if output, consumed, ok := consumeOptionalStringFlag(args, i, arg, "jsonl"); ok {
		n.jsonlEnabled = true
		n.jsonlOutput = output

		return consumed, true
	}

	if output, consumed, ok := consumeOptionalStringFlag(args, i, arg, "html"); ok {
		n.htmlEnabled = true
		n.htmlOutput = output

		return consumed, true
	}

	if output, consumed, ok := consumeOptionalStringFlag(args, i, arg, "sarif"); ok {
		n.sarifEnabled = true
		n.sarifOutput = output

		return consumed, true
	}

	return 0, false
}

func consumeOptionalStringFlag(args []string, i int, arg, name string) (string, int, bool) {
	switch {
	case arg == "--"+name || arg == "-"+name:
		output, consumed := consumeFlagValue(args, i)
		return output, consumed, true
	case strings.HasPrefix(arg, "--"+name+"="):
		return strings.TrimPrefix(arg, "--"+name+"="), 0, true
	case strings.HasPrefix(arg, "-"+name+"="):
		return strings.TrimPrefix(arg, "-"+name+"="), 0, true
	default:
		return "", 0, false
	}
}

func consumeFlagValue(args []string, i int) (string, int) {
	if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
		return args[i+1], 1
	}

	return "", 0
}
