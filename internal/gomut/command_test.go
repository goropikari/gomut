package gomut

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTarget(t *testing.T) {
	t.Run("package", func(t *testing.T) {
		target, err := resolveTarget("./internal/foo", false, "")
		if err != nil {
			t.Fatal(err)
		}
		if target.Mode != TargetModePackage || target.Value != "./internal/foo" {
			t.Fatalf("unexpected target: %+v", target)
		}
	})

	t.Run("all", func(t *testing.T) {
		target, err := resolveTarget("", true, "")
		if err != nil {
			t.Fatal(err)
		}
		if target.Mode != TargetModeAll || target.Value != "./..." {
			t.Fatalf("unexpected target: %+v", target)
		}
	})

	t.Run("diff", func(t *testing.T) {
		target, err := resolveTarget("", false, "HEAD~1..HEAD")
		if err != nil {
			t.Fatal(err)
		}
		if target.Mode != TargetModeDiff || target.Value != "HEAD~1..HEAD" {
			t.Fatalf("unexpected target: %+v", target)
		}
	})
}

func TestParseDiffPatch(t *testing.T) {
	patch := `diff --git a/foo.go b/foo.go
index 0000000..1111111 100644
--- a/foo.go
+++ b/foo.go
@@ -10,0 +11,2 @@
+x
+y
`
	files, err := parseDiffPatch(patch)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "foo.go" {
		t.Fatalf("unexpected files: %#v", files)
	}
	if !diffLineAllowed("foo.go", 11) || !diffLineAllowed("foo.go", 12) {
		t.Fatalf("expected diff lines to be allowed")
	}
	if diffLineAllowed("foo.go", 9) {
		t.Fatalf("unexpected line allowance")
	}
}

func TestApplyMutation(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	if err := os.WriteFile(file, []byte("package sample\n\nfunc add() int { return 1 + 2 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	candidate := Candidate{
		File:        "sample.go",
		Start:       42,
		End:         43,
		Replacement: "-",
	}
	out, err := applyMutation(dir, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out); got != "package sample\n\nfunc add() int { return 1 - 2 }\n" {
		t.Fatalf("unexpected output: %q", got)
	}
}

