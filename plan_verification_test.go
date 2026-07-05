package arrow_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// These tests mirror plan.md verification step 2 grep commands with --exclude-dir=testdata.

func TestPlanVerificationNoVariadicUseOutsideTestdata(t *testing.T) {
	root := repoRoot(t)
	pattern := regexp.MustCompile(`Use\(` + `middleware\.Recover\(\),`)
	walkForbidden(t, root, pattern, "variadic multi-arg Recover Use")
}

func TestPlanVerificationNoGroupUseOutsideTestdata(t *testing.T) {
	root := repoRoot(t)
	pattern := regexp.MustCompile(`Group\([^)]*\)\.Use\(`)
	walkForbidden(t, root, pattern, "Group-then-Use chain")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func walkForbidden(t *testing.T, root string, pattern *regexp.Regexp, label string) {
	t.Helper()
	var violations []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".md") {
			return nil
		}
		if strings.HasSuffix(path, "plan_verification_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if loc := pattern.FindStringIndex(string(data)); loc != nil {
			rel, _ := filepath.Rel(root, path)
			violations = append(violations, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("plan grep would match %s outside testdata:\n%s", label, strings.Join(violations, "\n"))
	}
}
