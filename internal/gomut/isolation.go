package gomut

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var defaultCopyExcludes = []string{".git", ".agents", ".cache", ".codex", ".worktree"}

func PrepareRunRoot(ctx context.Context, root string, stderr io.Writer) (string, func() error, error) {
	return prepareRunRoot(ctx, root, stderr, nil)
}

func prepareRunRoot(ctx context.Context, root string, stderr io.Writer, exclude []string) (string, func() error, error) {
	parent, err := os.MkdirTemp("", "gomut-run-")
	if err != nil {
		return "", nil, err
	}

	runRoot := filepath.Join(parent, "repo")
	matcher := newCopyExcludeMatcher(exclude)

	fmt.Fprintln(stderr, "Creating isolated temporary copy...")

	if err := copyTree(ctx, root, runRoot, matcher); err != nil {
		_ = os.RemoveAll(parent)
		return "", nil, err
	}

	cleanup := func() error {
		return os.RemoveAll(parent)
	}

	fmt.Fprintf(stderr, "Using isolated copy: %s\n", runRoot)

	return runRoot, cleanup, nil
}

func prepareMutationRoot(ctx context.Context, root string) (string, func() error, error) {
	parent, err := os.MkdirTemp("", "gomut-mutation-")
	if err != nil {
		return "", nil, err
	}

	mutationRoot := filepath.Join(parent, "repo")

	if err := copyTree(ctx, root, mutationRoot, newCopyExcludeMatcher(nil)); err != nil {
		_ = os.RemoveAll(parent)
		return "", nil, err
	}

	return mutationRoot, func() error {
		return os.RemoveAll(parent)
	}, nil
}

func copyTree(ctx context.Context, srcRoot, dstRoot string, matcher copyExcludeMatcher) error {
	return copyTreeDir(ctx, srcRoot, dstRoot, srcRoot, matcher)
}

func copyTreeDir(ctx context.Context, srcRoot, dstRoot, current string, matcher copyExcludeMatcher) error {
	entries, err := os.ReadDir(current)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}

		_, rel, err := copyTreeEntryPaths(srcRoot, current, entry.Name())
		if err != nil {
			return err
		}

		if matcher.matches(entry.Name(), rel) {
			continue
		}

		if err := copyTreeEntry(ctx, srcRoot, dstRoot, current, entry, matcher); err != nil {
			return err
		}
	}

	return nil
}

type copyExcludeMatcher struct {
	patterns []string
}

func newCopyExcludeMatcher(extra []string) copyExcludeMatcher {
	patterns := make([]string, 0, len(defaultCopyExcludes)+len(extra))
	patterns = append(patterns, defaultCopyExcludes...)

	for _, pattern := range extra {
		normalized := normalizeCopyExcludePattern(pattern)
		if normalized == "" {
			continue
		}

		patterns = append(patterns, normalized)
	}

	return copyExcludeMatcher{patterns: patterns}
}

func normalizeCopyExcludePattern(pattern string) string {
	normalized := strings.Trim(filepath.ToSlash(strings.TrimSpace(pattern)), "/")
	if normalized == "." {
		return ""
	}

	return normalized
}

func (m copyExcludeMatcher) matches(name, rel string) bool {
	for _, pattern := range m.patterns {
		if copyExcludePatternMatches(pattern, name, rel) {
			return true
		}
	}

	return false
}

func copyExcludePatternMatches(pattern, name, rel string) bool {
	if pattern == "" {
		return false
	}

	if copyExcludeTreePatternMatches(pattern, rel) {
		return true
	}

	if !strings.Contains(pattern, "/") {
		return copyExcludeNamePatternMatches(pattern, name)
	}

	return copyExcludePathPatternMatches(pattern, rel)
}

func copyExcludeTreePatternMatches(pattern, rel string) bool {
	if !strings.HasSuffix(pattern, "/**") {
		return false
	}

	prefix := strings.TrimSuffix(pattern, "/**")

	return rel == prefix || strings.HasPrefix(rel, prefix+"/")
}

func copyExcludeNamePatternMatches(pattern, name string) bool {
	if !strings.ContainsAny(pattern, "*?[") {
		return name == pattern
	}

	matched, err := path.Match(pattern, name)

	return err == nil && matched
}

func copyExcludePathPatternMatches(pattern, rel string) bool {
	if !strings.ContainsAny(pattern, "*?[") {
		return rel == pattern || strings.HasPrefix(rel, pattern+"/")
	}

	matched, err := path.Match(pattern, rel)

	return err == nil && matched
}

func copyTreeRel(srcRoot, srcPath string) (string, error) {
	rel, err := filepath.Rel(srcRoot, srcPath)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(rel), nil
}

func copyTreeEntryPaths(srcRoot, current, name string) (string, string, error) {
	srcPath := filepath.Join(current, name)

	rel, err := copyTreeRel(srcRoot, srcPath)
	if err != nil {
		return "", "", err
	}

	return srcPath, rel, nil
}

func copyTreeEntry(ctx context.Context, srcRoot, dstRoot, current string, entry fs.DirEntry, matcher copyExcludeMatcher) error {
	srcPath := filepath.Join(current, entry.Name())

	rel, err := copyTreeRel(srcRoot, srcPath)
	if err != nil {
		return err
	}

	dstPath := filepath.Join(dstRoot, filepath.FromSlash(rel))

	switch {
	case entry.Type()&os.ModeSymlink != 0:
		return copySymlink(srcPath, dstPath)
	case entry.IsDir():
		return copyDirectory(ctx, srcRoot, dstRoot, srcPath, dstPath, entry, matcher)
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

func copyDirectory(ctx context.Context, srcRoot, dstRoot, srcPath, dstPath string, entry fs.DirEntry, matcher copyExcludeMatcher) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dstPath, info.Mode().Perm()); err != nil {
		return err
	}

	return copyTreeDir(ctx, srcRoot, dstRoot, srcPath, matcher)
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
