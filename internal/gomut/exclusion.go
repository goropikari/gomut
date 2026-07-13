package gomut

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"gomut/internal/gomut/result"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"
)

// ExclusionNotice describes a skipped file or candidate together with the
// reason it was excluded.
type ExclusionNotice struct {
	File   string
	Line   int
	Reason string
}

// ExclusionFilter evaluates configured exclusion rules against repository files
// and discovered mutation candidates.
type ExclusionFilter struct {
	root     string
	patterns []string

	mu    sync.Mutex
	cache map[string]*parsedExclusionFile
}

type parsedExclusionFile struct {
	fset       *token.FileSet
	file       *ast.File
	commentMap ast.CommentMap
}

// NewExclusionFilter prepares a filter for repository-relative exclusion rules.
func NewExclusionFilter(root string, patterns []string) (*ExclusionFilter, error) {
	return &ExclusionFilter{
		root:     root,
		patterns: append([]string(nil), patterns...),
		cache:    map[string]*parsedExclusionFile{},
	}, nil
}

// SkipFile reports whether a repository-relative file should be excluded and
// returns the human-readable reason when it is.
func (f *ExclusionFilter) SkipFile(file string) (bool, string) {
	if f == nil {
		return false, ""
	}

	if reason, ok := f.matchFileRule(file); ok {
		return true, reason
	}

	return false, ""
}

// SkipCandidate reports whether a discovered mutation candidate should be
// excluded and returns the human-readable reason when it is.
func (f *ExclusionFilter) SkipCandidate(candidate result.Candidate) (bool, string) {
	if f == nil {
		return false, ""
	}

	if skipped, reason := f.SkipFile(candidate.File); skipped {
		return true, reason
	}

	parsed, err := f.parsedSource(candidate.File)
	if err != nil {
		return false, ""
	}

	for node := range candidateNodesAtLine(parsed.fset, parsed.file, candidate.Line) {
		if node == nil {
			continue
		}

		if _, ok := node.(*ast.File); ok {
			continue
		}

		if commentGroupsContainIgnore(parsed.commentMap[node]) {
			return true, fmt.Sprintf("excluded by //gomut:ignore near %s:%d", candidate.File, parsed.fset.Position(node.Pos()).Line)
		}
	}

	return false, ""
}

func (f *ExclusionFilter) matchFileRule(file string) (string, bool) {
	normalized := filepath.ToSlash(strings.TrimSpace(file))
	if normalized == "" {
		return "", false
	}

	base := filepath.Base(normalized)

	for _, rawPattern := range f.patterns {
		pattern := filepath.ToSlash(strings.TrimSpace(rawPattern))
		if pattern == "" {
			continue
		}

		if patternMatches(normalized, pattern) || patternMatches(base, pattern) {
			return fmt.Sprintf("excluded by pattern %q", rawPattern), true
		}
	}

	return "", false
}

func patternMatches(value, pattern string) bool {
	matched, err := pathpkg.Match(pattern, value)
	if err == nil && matched {
		return true
	}

	if !containsGlob(pattern) {
		if value == pattern || strings.HasPrefix(value, pattern+"/") {
			return true
		}
	}

	return false
}

func containsGlob(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

func (f *ExclusionFilter) parsedSource(file string) (*parsedExclusionFile, error) {
	key := filepath.ToSlash(file)

	f.mu.Lock()
	if parsed, ok := f.cache[key]; ok {
		f.mu.Unlock()
		return parsed, nil
	}
	f.mu.Unlock()

	path := resolveSourcePath(f.root, file)

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()

	astFile, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	parsed := &parsedExclusionFile{
		fset:       fset,
		file:       astFile,
		commentMap: ast.NewCommentMap(fset, astFile, astFile.Comments),
	}

	f.mu.Lock()
	f.cache[key] = parsed
	f.mu.Unlock()

	return parsed, nil
}

func candidateNodesAtLine(fset *token.FileSet, file *ast.File, line int) map[ast.Node]struct{} {
	nodes := map[ast.Node]struct{}{}

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		pos := fset.Position(n.Pos())

		end := fset.Position(n.End())
		if pos.Line == 0 || end.Line == 0 {
			return true
		}

		if line < pos.Line || line > end.Line {
			return true
		}

		nodes[n] = struct{}{}

		return true
	})

	return nodes
}

func commentGroupsContainIgnore(groups []*ast.CommentGroup) bool {
	for _, group := range groups {
		if group == nil {
			continue
		}

		for _, comment := range group.List {
			if strings.Contains(comment.Text, "gomut:ignore") {
				return true
			}
		}
	}

	return false
}
