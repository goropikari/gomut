package gomut

import (
	"os"
	"path/filepath"
	"strings"
)

func goCommandEnv() []string {
	cacheDir := filepath.Join(os.TempDir(), "gomut-gocache")
	_ = os.MkdirAll(cacheDir, 0o755)

	env := upsertEnv(os.Environ(), "GOCACHE", cacheDir)

	goFlags := os.Getenv("GOFLAGS")
	if !hasBuildVCSFlag(goFlags) {
		goFlags = strings.TrimSpace(goFlags + " -buildvcs=false")
	}

	return upsertEnv(env, "GOFLAGS", goFlags)
}

func upsertEnv(env []string, key, value string) []string {
	prefix := key + "="
	entry := prefix + value

	for i, item := range env {
		if strings.HasPrefix(item, prefix) {
			env[i] = entry
			return env
		}
	}

	return append(env, entry)
}

func hasBuildVCSFlag(value string) bool {
	for _, field := range strings.Fields(value) {
		if field == "-buildvcs" || strings.HasPrefix(field, "-buildvcs=") {
			return true
		}
	}

	return false
}

func packageForFile(root, file string) (string, error) {
	return packageForDir(root, filepath.Dir(resolveSourcePath(root, file)))
}

func writeFilePreserveMode(path string, data []byte) error {
	mode := os.FileMode(0o600)

	info, err := os.Stat(path)
	if err == nil {
		mode = info.Mode().Perm()
	}

	return os.WriteFile(path, data, mode) // #nosec G703 -- path is derived from repository-local source paths only
}

func repoRel(root, path string) string {
	wd := root
	if wd == "" {
		wd, _ = os.Getwd()
	}

	wd, err := filepath.Abs(wd)
	if err != nil {
		return filepath.ToSlash(path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return filepath.ToSlash(path)
	}

	rel, err := filepath.Rel(wd, absPath)
	if err != nil {
		return filepath.ToSlash(path)
	}

	return filepath.ToSlash(rel)
}
