package gomut

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func PrepareRunRoot(ctx context.Context, root string, stderr io.Writer) (string, func() error, error) {
	parent, err := os.MkdirTemp("", "gomut-run-")
	if err != nil {
		return "", nil, err
	}

	runRoot := filepath.Join(parent, "repo")

	fmt.Fprintln(stderr, "Creating isolated temporary copy...")

	if err := copyTree(ctx, root, runRoot); err != nil {
		_ = os.RemoveAll(parent)
		return "", nil, err
	}

	cleanup := func() error {
		return os.RemoveAll(parent)
	}

	fmt.Fprintf(stderr, "Using isolated copy: %s\n", runRoot)

	return runRoot, cleanup, nil
}

func prepareRunRoot(ctx context.Context, root string, stderr io.Writer) (string, func() error, error) {
	return PrepareRunRoot(ctx, root, stderr)
}

func prepareMutationRoot(ctx context.Context, root string) (string, func() error, error) {
	parent, err := os.MkdirTemp("", "gomut-mutation-")
	if err != nil {
		return "", nil, err
	}

	mutationRoot := filepath.Join(parent, "repo")

	if err := copyTree(ctx, root, mutationRoot); err != nil {
		_ = os.RemoveAll(parent)
		return "", nil, err
	}

	return mutationRoot, func() error {
		return os.RemoveAll(parent)
	}, nil
}

func copyTree(ctx context.Context, srcRoot, dstRoot string) error {
	return copyTreeDir(ctx, srcRoot, dstRoot, srcRoot)
}

func copyTreeDir(ctx context.Context, srcRoot, dstRoot, current string) error {
	entries, err := os.ReadDir(current)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}

		if entry.Name() == ".git" {
			continue
		}

		if err := copyTreeEntry(ctx, srcRoot, dstRoot, current, entry); err != nil {
			return err
		}
	}

	return nil
}

func copyTreeEntry(ctx context.Context, srcRoot, dstRoot, current string, entry fs.DirEntry) error {
	srcPath := filepath.Join(current, entry.Name())

	rel, err := filepath.Rel(srcRoot, srcPath)
	if err != nil {
		return err
	}

	dstPath := filepath.Join(dstRoot, rel)

	switch {
	case entry.Type()&os.ModeSymlink != 0:
		return copySymlink(srcPath, dstPath)
	case entry.IsDir():
		return copyDirectory(ctx, srcRoot, dstRoot, srcPath, dstPath, entry)
	default:
		return copyRegularFile(srcPath, dstPath, entry)
	}
}

func copySymlink(srcPath, dstPath string) error {
	target, err := os.Readlink(srcPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}

	return os.Symlink(target, dstPath)
}

func copyDirectory(ctx context.Context, srcRoot, dstRoot, srcPath, dstPath string, entry fs.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dstPath, info.Mode().Perm()); err != nil {
		return err
	}

	return copyTreeDir(ctx, srcRoot, dstRoot, srcPath)
}

func copyRegularFile(srcPath, dstPath string, entry fs.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}

	return copyFile(srcPath, dstPath, info.Mode().Perm())
}

func copyFile(srcPath, dstPath string, perm fs.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return dst.Chmod(perm)
}
