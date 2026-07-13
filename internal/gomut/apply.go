package gomut

import (
	"fmt"
	"gomut/internal/gomut/result"
	"os"
	"path/filepath"
)

func ApplyMutation(root string, candidate result.Candidate) ([]byte, error) {
	path := resolveSourcePath(root, candidate.File)

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if candidate.Start < 0 || candidate.End > len(src) || candidate.Start > candidate.End {
		return nil, fmt.Errorf("invalid mutation range for %s:%d", candidate.File, candidate.Line)
	}

	out := make([]byte, 0, len(src)-(candidate.End-candidate.Start)+len(candidate.Replacement))
	out = append(out, src[:candidate.Start]...)
	out = append(out, candidate.Replacement...)
	out = append(out, src[candidate.End:]...)

	return out, nil
}

func resolveSourcePath(root, file string) string {
	if filepath.IsAbs(file) {
		return file
	}

	return filepath.Join(root, file)
}
