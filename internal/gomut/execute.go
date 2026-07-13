package gomut

import (
	"context"
	"gomut/internal/gomut/result"
	"os"
	"os/exec"
	"strings"
	"time"
)

func (r *Runner) executeMutation(ctx context.Context, root string, candidate result.Candidate, timeout time.Duration) (result.MutationResult, string, error) {
	if r.executeMutationFunc != nil {
		return r.executeMutationFunc(ctx, root, candidate, timeout)
	}

	mutated, err := ApplyMutation(root, candidate)
	if err != nil {
		return result.MutationResultNotViable, err.Error(), nil
	}

	pkg := candidate.PackagePath
	if pkg == "" {
		pkg, err = packageForFile(root, candidate.File)
		if err != nil {
			return result.MutationResultNotViable, err.Error(), nil
		}
	}

	path := resolveSourcePath(root, candidate.File)

	backup, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	if err := writeFilePreserveMode(path, mutated); err != nil {
		return "", "", err
	}
	defer func() {
		_ = writeFilePreserveMode(path, backup)
	}()

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "go", "test", pkg)
	cmd.Dir = root
	cmd.Env = goCommandEnv()
	out, err := cmd.CombinedOutput()

	if cmdCtx.Err() == context.DeadlineExceeded {
		return result.MutationResultTimedOut, "mutation test timed out", nil
	}

	if err != nil {
		if looksLikeNotViable(string(out)) {
			return result.MutationResultNotViable, trimOutput(out), nil
		}

		return result.MutationResultKilled, trimOutput(out), nil
	}

	return result.MutationResultLived, trimOutput(out), nil
}

func looksLikeNotViable(output string) bool {
	needles := []string{
		"build failed",
		"syntax error",
		"undefined:",
		"cannot use",
		"mismatched types",
		"too many errors",
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
