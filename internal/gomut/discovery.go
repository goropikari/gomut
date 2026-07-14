package gomut

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"gomut/internal/gomut/result"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func DiscoverCandidates(root string, packages []string, target result.Target, coverage map[string]result.FileCoverage) ([]result.Candidate, error) {
	candidates, _, err := discoverCandidates(root, packages, target, coverage, nil, result.MutationKindFilter{})
	if err != nil {
		return nil, err
	}

	return candidates, nil
}

// DiscoverCandidatesWithExclusions collects mutation candidates and exclusion
// notices using the provided exclusion filter.
func DiscoverCandidatesWithExclusions(root string, packages []string, target result.Target, coverage map[string]result.FileCoverage, filter *ExclusionFilter, kindFilter result.MutationKindFilter) ([]result.Candidate, []ExclusionNotice, error) {
	return discoverCandidates(root, packages, target, coverage, filter, kindFilter)
}

func discoverCandidates(root string, packages []string, target result.Target, coverage map[string]result.FileCoverage, filter *ExclusionFilter, kindFilter result.MutationKindFilter) ([]result.Candidate, []ExclusionNotice, error) {
	var (
		candidates []result.Candidate
		notices    []ExclusionNotice
	)

	for _, pkg := range packages {
		files, err := packageGoFiles(root, pkg)
		if err != nil {
			return nil, nil, fmt.Errorf("list go files for %s: %w", pkg, err)
		}

		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}

			if filter != nil {
				if skipped, reason := filter.SkipFile(repoRel(root, file)); skipped {
					notices = append(notices, ExclusionNotice{File: repoRel(root, file), Reason: reason})
					continue
				}
			}

			fileCandidates, fileNotices, err := discoverFileCandidates(root, pkg, file, target, coverage, filter, kindFilter)
			if err != nil {
				return nil, nil, fmt.Errorf("discover candidates for %s: %w", file, err)
			}

			notices = append(notices, fileNotices...)
			candidates = append(candidates, fileCandidates...)
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].File != candidates[j].File {
			return candidates[i].File < candidates[j].File
		}

		if candidates[i].Line != candidates[j].Line {
			return candidates[i].Line < candidates[j].Line
		}

		return candidates[i].Kind < candidates[j].Kind
	})

	return candidates, notices, nil
}

func discoverFileCandidates(root, pkg, file string, target result.Target, coverage map[string]result.FileCoverage, filter *ExclusionFilter, kindFilter result.MutationKindFilter) ([]result.Candidate, []ExclusionNotice, error) {
	src, err := os.ReadFile(file)
	if err != nil {
		return nil, nil, err
	}

	fset := token.NewFileSet()

	astFile, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	covered := coverage[repoRel(root, file)]

	var (
		candidates []result.Candidate
		notices    []ExclusionNotice
	)

	var ancestors []ast.Node

	ast.Inspect(astFile, func(n ast.Node) bool {
		if n == nil {
			if len(ancestors) > 0 {
				ancestors = ancestors[:len(ancestors)-1]
			}

			return false
		}

		ancestors = append(ancestors, n)

		if candidate, ok := mutationCandidateFromNode(root, fset, src, astFile, file, pkg, n, ancestors[:len(ancestors)-1], target, covered); ok {
			if !kindFilter.Matches(candidate.Kind) {
				return true
			}

			if filter != nil {
				if skipped, reason := filter.SkipCandidate(candidate); skipped {
					notices = append(notices, ExclusionNotice{File: candidate.File, Line: candidate.Line, Reason: reason})
					return true
				}
			}

			candidates = append(candidates, candidate)
		}

		return true
	})

	return candidates, notices, nil
}

func resolvePackages(ctx context.Context, originalRoot, root string, target result.Target) ([]string, error) {
	switch target.Mode {
	case result.TargetModePackage:
		packages, err := listPackages(ctx, root, target.Value)
		if err != nil {
			return nil, fmt.Errorf("list packages for target %s: %w", target.Value, err)
		}

		return packages, nil
	case result.TargetModeAll:
		packages, err := listPackages(ctx, root, "./...")
		if err != nil {
			return nil, fmt.Errorf("list packages for all target: %w", err)
		}

		return packages, nil
	case result.TargetModeDiff:
		files, err := DiffFiles(ctx, originalRoot, target.Value)
		if err != nil {
			return nil, fmt.Errorf("collect diff files: %w", err)
		}

		packages, err := packageDirsFromFiles(root, files)
		if err != nil {
			return nil, fmt.Errorf("resolve packages from diff files: %w", err)
		}

		return packages, nil
	default:
		return nil, fmt.Errorf("unsupported target mode %q", target.Mode)
	}
}

func runBaseline(ctx context.Context, root string, packages []string) (map[string]result.FileCoverage, error) {
	merged := map[string]result.FileCoverage{}

	modulePath, err := modulePath(root)
	if err != nil {
		return nil, err
	}

	for _, pkg := range packages {
		coverProfile := filepath.Join(os.TempDir(), strings.ReplaceAll("gomut-"+strings.ReplaceAll(pkg, "/", "-"), "...", "all")+".cover")
		cmd := exec.CommandContext(ctx, "go", "test", "-coverprofile", coverProfile, pkg)
		cmd.Dir = root
		cmd.Env = goCommandEnv()

		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("baseline go test failed for %s: %w\n%s", pkg, err, string(out))
		}

		coverage, err := readCoverage(root, coverProfile, modulePath)
		if err != nil {
			return nil, err
		}

		for file, cov := range coverage {
			merged[file] = mergeCoverage(merged[file], cov)
		}
	}

	return merged, nil
}

func mergeCoverage(dst, src result.FileCoverage) result.FileCoverage {
	dst.Ranges = append(dst.Ranges, src.Ranges...)
	return dst
}

func listPackages(ctx context.Context, root, pattern string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "go", "list", pattern)
	cmd.Dir = root
	cmd.Env = goCommandEnv()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go list %s: %w: %s", pattern, err, strings.TrimSpace(string(out)))
	}

	lines := strings.Fields(strings.TrimSpace(string(out)))
	sort.Strings(lines)

	return lines, nil
}

func packageGoFiles(root, pkg string) ([]string, error) {
	cmd := exec.Command("go", "list", "-f", "{{.Dir}} {{join .GoFiles \" \"}}", pkg)
	cmd.Dir = root
	cmd.Env = goCommandEnv()

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return nil, fmt.Errorf("package not found: %s", pkg)
	}

	dir := fields[0]

	var files []string
	for _, name := range fields[1:] {
		files = append(files, filepath.Join(dir, name))
	}

	return files, nil
}

func packageDirsFromFiles(root string, files []string) ([]string, error) {
	dirs := map[string]struct{}{}

	for _, file := range files {
		dir := filepath.Dir(filepath.Join(root, file))

		pkg, err := packageForDir(root, dir)
		if err != nil {
			return nil, err
		}

		dirs[pkg] = struct{}{}
	}

	var packages []string
	for pkg := range dirs {
		packages = append(packages, pkg)
	}

	sort.Strings(packages)

	return packages, nil
}

func packageForDir(root, dir string) (string, error) {
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}", dir)
	cmd.Dir = root
	cmd.Env = goCommandEnv()

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func modulePath(root string) (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Path}}")
	cmd.Dir = root
	cmd.Env = goCommandEnv()

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func readCoverage(root, path, modulePath string) (map[string]result.FileCoverage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	coverage := map[string]result.FileCoverage{}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}

		file, block := parseCoverageBlock(fields[0])
		if file == "" {
			continue
		}

		count, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}

		normalized := coveragePath(root, file, modulePath)
		coverage[normalized] = appendCoverageBlock(coverage[normalized], block, count > 0)
	}

	return coverage, nil
}

func coveragePath(root, path, modulePath string) string {
	if modulePath != "" {
		prefix := modulePath + "/"
		if strings.HasPrefix(path, prefix) {
			return filepath.ToSlash(strings.TrimPrefix(path, prefix))
		}
	}

	return repoRel(root, path)
}

func parseCoverageBlock(spec string) (string, result.CoverageRange) {
	parts := strings.Split(spec, ":")
	if len(parts) != 2 {
		return "", result.CoverageRange{}
	}

	file := filepath.ToSlash(parts[0])

	ranges := strings.Split(parts[1], ",")
	if len(ranges) != 2 {
		return "", result.CoverageRange{}
	}

	start := strings.Split(ranges[0], ".")

	end := strings.Split(ranges[1], ".")
	if len(start) != 2 || len(end) != 2 {
		return "", result.CoverageRange{}
	}

	startLine, err1 := strconv.Atoi(start[0])

	endLine, err2 := strconv.Atoi(end[0])
	if err1 != nil || err2 != nil {
		return "", result.CoverageRange{}
	}

	return file, result.CoverageRange{StartLine: startLine, EndLine: endLine}
}

func appendCoverageBlock(fc result.FileCoverage, block result.CoverageRange, covered bool) result.FileCoverage {
	block.Covered = covered
	fc.Ranges = append(fc.Ranges, block)

	return fc
}
