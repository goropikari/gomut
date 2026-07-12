package gomut

import (
	"fmt"
	"strings"
)

// MutationResultFilter defines the allowed mutation execution results for output filtering.
type MutationResultFilter struct {
	allowed map[MutationResult]struct{}
}

// ParseMutationResultFilter converts raw CLI values into a mutation result filter.
func ParseMutationResultFilter(values []string) (MutationResultFilter, error) {
	allowed := map[MutationResult]struct{}{}

	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			result, ok := parseMutationResultToken(token)
			if !ok {
				trimmed := strings.TrimSpace(token)
				if trimmed == "" {
					continue
				}

				return MutationResultFilter{}, fmt.Errorf("unknown result type: %s", trimmed)
			}

			allowed[result] = struct{}{}
		}
	}

	return MutationResultFilter{allowed: allowed}, nil
}

func parseMutationResultToken(value string) (MutationResult, bool) {
	token := strings.TrimSpace(strings.ToLower(value))
	token = strings.ReplaceAll(token, "_", "-")
	token = strings.ReplaceAll(token, " ", "-")

	switch token {
	case "":
		return "", false
	case "killed":
		return MutationResultKilled, true
	case "lived":
		return MutationResultLived, true
	case "not-covered":
		return MutationResultNotCovered, true
	case "timed-out":
		return MutationResultTimedOut, true
	case "not-viable":
		return MutationResultNotViable, true
	default:
		return "", false
	}
}

// Matches reports whether the given result should be included in filtered output.
func (f MutationResultFilter) Matches(result MutationResult) bool {
	if len(f.allowed) == 0 {
		return true
	}

	_, ok := f.allowed[result]

	return ok
}
