package gomut

import (
	"context"
	"fmt"
	"io"
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
