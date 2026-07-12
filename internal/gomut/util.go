package gomut

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func listPackages(ctx context.Context, root, pattern string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "go", "list", pattern)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Fields(strings.TrimSpace(string(out)))
	sort.Strings(lines)
	return lines, nil
}

func packageGoFiles(root, pkg string) ([]string, error) {
	cmd := exec.Command("go", "list", "-f", "{{.Dir}} {{join .GoFiles \" \"}}", pkg)
	cmd.Dir = root
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
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func packageForFile(root, file string) (string, error) {
	return packageForDir(root, filepath.Dir(filepath.Join(root, file)))
}

func readCoverage(path string) (map[string]FileCoverage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	coverage := map[string]FileCoverage{}
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
		coverage[repoRel(file)] = appendCoverageBlock(coverage[repoRel(file)], block, count > 0)
	}
	return coverage, nil
}

func parseCoverageBlock(spec string) (string, CoverageRange) {
	parts := strings.Split(spec, ":")
	if len(parts) != 2 {
		return "", CoverageRange{}
	}
	file := filepath.ToSlash(parts[0])
	ranges := strings.Split(parts[1], ",")
	if len(ranges) != 2 {
		return "", CoverageRange{}
	}
	start := strings.Split(ranges[0], ".")
	end := strings.Split(ranges[1], ".")
	if len(start) != 2 || len(end) != 2 {
		return "", CoverageRange{}
	}
	startLine, err1 := strconv.Atoi(start[0])
	endLine, err2 := strconv.Atoi(end[0])
	if err1 != nil || err2 != nil {
		return "", CoverageRange{}
	}
	return file, CoverageRange{StartLine: startLine, EndLine: endLine}
}

func appendCoverageBlock(fc FileCoverage, block CoverageRange, covered bool) FileCoverage {
	block.Covered = covered
	fc.Ranges = append(fc.Ranges, block)
	return fc
}

func jsonString(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}
