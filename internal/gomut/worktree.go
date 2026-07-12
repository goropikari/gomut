package gomut

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func PrepareWorktree(ctx context.Context, root string) (string, func() error, error) {
	parent, err := os.MkdirTemp("", "gomut-worktree-")
	if err != nil {
		return "", nil, err
	}

	worktreeRoot := filepath.Join(parent, "repo")
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", worktreeRoot, "HEAD")
	cmd.Dir = root

	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(parent)

		msg := string(out)
		if strings.Contains(msg, "Read-only file system") || strings.Contains(msg, "could not create leading directories of '.git/worktrees'") {
			return "", nil, fmt.Errorf("create worktree: %w\n%s\nworktree mode requires a writable git metadata directory (.git/worktrees)", err, msg)
		}

		return "", nil, fmt.Errorf("create worktree: %w\n%s", err, msg)
	}

	cleanup := func() error {
		remove := exec.Command("git", "worktree", "remove", "--force", worktreeRoot)
		remove.Dir = root

		remove.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		out, err := remove.CombinedOutput()
		if err != nil {
			_ = os.RemoveAll(parent)
			return fmt.Errorf("remove worktree: %w\n%s", err, string(out))
		}

		return os.RemoveAll(parent)
	}

	return worktreeRoot, cleanup, nil
}
