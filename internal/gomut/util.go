package gomut

import (
	"os"
	"path/filepath"
)

func goCommandEnv() []string {
	cacheDir := filepath.Join(os.TempDir(), "gomut-gocache")
	_ = os.MkdirAll(cacheDir, 0o755)

	return append(os.Environ(), "GOCACHE="+cacheDir)
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
