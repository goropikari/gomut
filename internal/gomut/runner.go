package gomut

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Runner struct {
	stdout io.Writer
	stderr io.Writer
}

func NewRunner(stdout, stderr io.Writer) *Runner {
	return &Runner{stdout: stdout, stderr: stderr}
}

func (r *Runner) Run(ctx context.Context, cfg RunConfig) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	packages, err := r.resolvePackages(ctx, root, cfg.Target)
	if err != nil {
		return err
	}
	if len(packages) == 0 {
		return errors.New("no packages matched target")
	}

	coverage, err := r.runBaseline(ctx, root, packages)
	if err != nil {
		return err
	}

	candidates, err := DiscoverCandidates(root, packages, cfg.Target, coverage)
	if err != nil {
		return err
	}

	var output io.WriteCloser
	if cfg.OutputPath != "" {
		output, err = os.Create(cfg.OutputPath)
		if err != nil {
			return err
		}
		defer output.Close()
	}

	summary := Summary{}
	startedAt := time.Now().Format(time.RFC3339)
	command := strings.Join(os.Args, " ")

	var records []Record
	for _, candidate := range candidates {
		summary.Total++
		if !candidate.Covered {
			summary.NotCovered++
			record := Record{
				Target:    cfg.Target,
				StartedAt: startedAt,
				Command:   command,
				Summary:   summary,
				Mutation: MutationMetadata{
					File:    candidate.File,
					Line:    candidate.Line,
					Kind:    candidate.Kind,
					Result:  MutationResultNotCovered,
					Message: "line not covered by baseline tests",
				},
			}
			records = append(records, record)
			if output != nil {
				if err := writeJSONL(output, record); err != nil {
					return err
				}
			}
			continue
		}
		result, message, err := r.executeMutation(ctx, root, candidate, cfg.Timeout)
		if err != nil {
			return err
		}

		switch result {
		case MutationResultKilled:
			summary.Killed++
		case MutationResultLived:
			summary.Lived++
		case MutationResultNotCovered:
			summary.NotCovered++
		case MutationResultTimedOut:
			summary.TimedOut++
		case MutationResultNotViable:
			summary.NotViable++
		}

		record := Record{
			Target:    cfg.Target,
			StartedAt: startedAt,
			Command:   command,
			Summary:   summary,
			Mutation: MutationMetadata{
				File:    candidate.File,
				Line:    candidate.Line,
				Kind:    candidate.Kind,
				Result:  result,
				Message: message,
			},
		}
		records = append(records, record)
		if output != nil {
			if err := writeJSONL(output, record); err != nil {
				return err
			}
		}
	}

	r.printSummary(summary, len(candidates))
	if len(records) == 0 {
		fmt.Fprintln(r.stdout, "No mutation candidates found.")
	}
	return nil
}

func (r *Runner) printSummary(summary Summary, total int) {
	fmt.Fprintln(r.stdout, "Mutation summary")
	fmt.Fprintf(r.stdout, "  total: %d\n", total)
	fmt.Fprintf(r.stdout, "  killed: %d\n", summary.Killed)
	fmt.Fprintf(r.stdout, "  lived: %d\n", summary.Lived)
	fmt.Fprintf(r.stdout, "  not covered: %d\n", summary.NotCovered)
	fmt.Fprintf(r.stdout, "  timed out: %d\n", summary.TimedOut)
	fmt.Fprintf(r.stdout, "  not viable: %d\n", summary.NotViable)
}

func (r *Runner) resolvePackages(ctx context.Context, root string, target Target) ([]string, error) {
	switch target.Mode {
	case TargetModePackage:
		return []string{target.Value}, nil
	case TargetModeAll:
		return listPackages(ctx, root, ".")
	case TargetModeDiff:
		files, err := diffFiles(ctx, root, target.Value)
		if err != nil {
			return nil, err
		}
		return packageDirsFromFiles(root, files)
	default:
		return nil, fmt.Errorf("unsupported target mode %q", target.Mode)
	}
}

func (r *Runner) runBaseline(ctx context.Context, root string, packages []string) (map[string]FileCoverage, error) {
	merged := map[string]FileCoverage{}
	for _, pkg := range packages {
		coverProfile := filepath.Join(os.TempDir(), strings.ReplaceAll("gomut-"+strings.ReplaceAll(pkg, "/", "-"), "...", "all")+".cover")
		cmd := exec.CommandContext(ctx, "go", "test", "-coverprofile", coverProfile, pkg)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("baseline go test failed for %s: %w\n%s", pkg, err, string(out))
		}
		coverage, err := readCoverage(coverProfile)
		if err != nil {
			return nil, err
		}
		for file, cov := range coverage {
			merged[file] = mergeCoverage(merged[file], cov)
		}
	}
	return merged, nil
}

func mergeCoverage(dst, src FileCoverage) FileCoverage {
	dst.Ranges = append(dst.Ranges, src.Ranges...)
	return dst
}

func (r *Runner) executeMutation(ctx context.Context, root string, candidate Candidate, timeout time.Duration) (MutationResult, string, error) {
	mutated, err := applyMutation(root, candidate)
	if err != nil {
		return MutationResultNotViable, err.Error(), nil
	}

	pkg := candidate.PackagePath
	if pkg == "" {
		pkg, err = packageForFile(root, candidate.File)
		if err != nil {
			return MutationResultNotViable, err.Error(), nil
		}
	}

	path := filepath.Join(root, candidate.File)
	backup, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(path, mutated, 0o644); err != nil {
		return "", "", err
	}
	defer func() {
		_ = os.WriteFile(path, backup, 0o644)
	}()

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "go", "test", pkg)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if cmdCtx.Err() == context.DeadlineExceeded {
		return MutationResultTimedOut, "mutation test timed out", nil
	}
	if err != nil {
		if looksLikeNotViable(string(out)) {
			return MutationResultNotViable, trimOutput(out), nil
		}
		return MutationResultKilled, trimOutput(out), nil
	}
	return MutationResultLived, trimOutput(out), nil
}

func looksLikeNotViable(output string) bool {
	needles := []string{
		"build failed",
		"syntax error",
		"undefined:",
		"cannot use",
		"expected",
		"mismatched types",
	}
	for _, needle := range needles {
		if strings.Contains(output, needle) {
			return true
		}
	}
	return false
}

func trimOutput(out []byte) string {
	text := strings.TrimSpace(string(out))
	if text == "" {
		return "tests passed"
	}
	return text
}
