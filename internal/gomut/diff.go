package gomut

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type diffHunk struct {
	Start int
	End   int
}

var diffState = map[string][]diffHunk{}

func diffFiles(ctx context.Context, root, diffRange string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--unified=0", diffRange, "--", "*.go")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return ParseDiffPatch(string(out))
}

func ParseDiffPatch(patch string) ([]string, error) {
	var files []string
	var current string
	var hunks []diffHunk
	for _, line := range strings.Split(patch, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			if current != "" {
				diffState[current] = append([]diffHunk(nil), hunks...)
				files = append(files, current)
			}
			hunks = nil
			current = ""
		case strings.HasPrefix(line, "+++ b/"):
			current = strings.TrimPrefix(line, "+++ b/")
		case strings.HasPrefix(line, "@@ "):
			hunk, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			hunks = append(hunks, hunk)
		}
	}
	if current != "" {
		diffState[current] = append([]diffHunk(nil), hunks...)
		files = append(files, current)
	}
	return files, nil
}

func parseHunkHeader(line string) (diffHunk, error) {
	parts := strings.Split(line, " ")
	if len(parts) < 3 {
		return diffHunk{}, fmt.Errorf("invalid hunk header: %s", line)
	}
	plus := parts[2]
	plus = strings.TrimPrefix(plus, "+")
	rangeParts := strings.SplitN(plus, ",", 2)
	start, err := strconv.Atoi(rangeParts[0])
	if err != nil {
		return diffHunk{}, err
	}
	end := start
	if len(rangeParts) == 2 {
		span, err := strconv.Atoi(rangeParts[1])
		if err != nil {
			return diffHunk{}, err
		}
		end = start + span - 1
	}
	return diffHunk{Start: start, End: end}, nil
}

func DiffLineAllowed(file string, line int) bool {
	hunks := diffState[filepath.ToSlash(file)]
	for _, h := range hunks {
		if line >= h.Start && line <= h.End {
			return true
		}
	}
	return false
}
