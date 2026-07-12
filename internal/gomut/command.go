package gomut

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)

type Command struct {
	stdout io.Writer
	stderr io.Writer
}

func NewCommand(stdout, stderr io.Writer) *Command {
	return &Command{stdout: stdout, stderr: stderr}
}

func (c *Command) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return c.printUsage()
	}

	switch args[0] {
	case "test":
		return c.runTest(ctx, args[1:])
	case "help", "-h", "--help":
		return c.printUsage()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (c *Command) printUsage() error {
	_, err := fmt.Fprint(c.stdout, usageText)
	return err
}

func (c *Command) runTest(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		pkgTarget = fs.String("package", "", "package import path or pattern")
		allTarget = fs.Bool("all", false, "test all packages")
		diffRange = fs.String("diff", "", "git diff range, for example HEAD~1..HEAD")
		timeout   = fs.Duration("timeout", 10*time.Second, "timeout per mutation")
	)

	parsedArgs, jsonlOutput, err := NormalizeTestArgs(args)
	if err != nil {
		return err
	}

	if err := fs.Parse(parsedArgs); err != nil {
		return err
	}

	target, err := ResolveTarget(*pkgTarget, *allTarget, *diffRange)
	if err != nil {
		return err
	}

	cfg := RunConfig{
		Target:     target,
		Timeout:    *timeout,
		OutputPath: jsonlOutput,
	}

	runner := NewRunner(c.stdout, c.stderr)

	return runner.Run(ctx, cfg)
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

func repoRel(path string) string {
	rel, err := filepath.Rel(".", path)
	if err != nil {
		return path
	}

	return filepath.ToSlash(rel)
}

const usageText = `gomut test --package ./... [--timeout 2s] [--jsonl [mutations.jsonl]]
gomut test --all
gomut test --diff HEAD~1..HEAD
`
