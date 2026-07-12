package gomut

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type Command struct {
	stdout      io.Writer
	stderr      io.Writer
	jsonlOutput string
}

func NewCommand(stdout, stderr io.Writer) *Command {
	return &Command{stdout: stdout, stderr: stderr}
}

func (c *Command) Run(ctx context.Context, args []string) error {
	normalizedArgs, jsonlOutput, err := NormalizeTestArgs(args)
	if err != nil {
		return err
	}

	c.jsonlOutput = jsonlOutput

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
	flags.String("diff", "", "git diff range, for example HEAD~1..HEAD")
	flags.Duration("timeout", 10*time.Second, "timeout per mutation")
	flags.String("jsonl", "", "jsonl output file path")
	flags.Lookup("jsonl").NoOptDefVal = ""

	return cmd
}

func (c *Command) runTest(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(args, " "))
	}

	var (
		pkgTarget, _   = cmd.Flags().GetString("package")
		allTarget, _   = cmd.Flags().GetBool("all")
		diffRange, _   = cmd.Flags().GetString("diff")
		timeout, _     = cmd.Flags().GetDuration("timeout")
		jsonlOutput, _ = cmd.Flags().GetString("jsonl")
	)
	if jsonlOutput == "" {
		jsonlOutput = c.jsonlOutput
	}

	target, err := ResolveTarget(pkgTarget, allTarget, diffRange)
	if err != nil {
		return err
	}

	cfg := RunConfig{
		Target:     target,
		Timeout:    timeout,
		OutputPath: jsonlOutput,
	}

	runner := NewRunner(c.stdout, c.stderr)

	return runner.Run(cmd.Context(), cfg)
}

func NormalizeTestArgs(args []string) ([]string, string, error) {
	normalized := make([]string, 0, len(args))
	output := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--jsonl" || arg == "-jsonl":
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				output = args[i+1]
				i++
			} else {
				output = ""
			}
		case strings.HasPrefix(arg, "--jsonl="):
			output = strings.TrimPrefix(arg, "--jsonl=")
		case strings.HasPrefix(arg, "-jsonl="):
			output = strings.TrimPrefix(arg, "-jsonl=")
		default:
			normalized = append(normalized, arg)
		}
	}

	return normalized, output, nil
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
