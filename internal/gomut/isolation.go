package gomut

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func prepareRunRoot(ctx context.Context, root string, stderr io.Writer) (string, func() error, error) {
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

func copyTree(ctx context.Context, srcRoot, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		if entry.Name() == ".git" {
			if entry.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		dstPath := filepath.Join(dstRoot, rel)
		if entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				return err
			}

			return os.Symlink(target, dstPath)
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		switch {
		case entry.IsDir():
			return os.MkdirAll(dstPath, info.Mode().Perm())
		default:
			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				return err
			}

			return copyFile(path, dstPath, info.Mode().Perm())
		}
	})
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
