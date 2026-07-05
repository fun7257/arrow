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

var forbiddenPatterns = []*regexp.Regexp{
	regexp.MustCompile(`Use\(` + `middleware\.Recover\(\),`),
	regexp.MustCompile(`\.Use\([^)]*,`),
	regexp.MustCompile(`Group\([^)]*\)\.Use\(`),
}

func TestClassicMiddlewareRegistration(t *testing.T) {
	root := repoRoot(t)
	var violations []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, ".md") {
			return nil
		}
		if filepath.Base(path) == "style_test.go" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		for _, re := range forbiddenPatterns {
			if loc := re.FindStringIndex(content); loc != nil {
				rel, _ := filepath.Rel(root, path)
				violations = append(violations, rel+": "+content[loc[0]:loc[1]])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("forbidden middleware registration patterns:\n%s", strings.Join(violations, "\n"))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}