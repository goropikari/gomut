package gomut

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
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
		output    = fs.String("jsonl", "", "write JSON Lines output to this file; defaults to stdout")
	)

	if err := fs.Parse(args); err != nil {
		return err
	}

	target, err := ResolveTarget(*pkgTarget, *allTarget, *diffRange)
	if err != nil {
		return err
	}

	cfg := RunConfig{
		Target:     target,
		Timeout:    *timeout,
		OutputPath: *output,
	}

	runner := NewRunner(c.stdout, c.stderr)
	return runner.Run(ctx, cfg)
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

const usageText = `gomut test --package ./... [--timeout 2s] [--jsonl mutations.jsonl]
gomut test --all
gomut test --diff HEAD~1..HEAD
`
