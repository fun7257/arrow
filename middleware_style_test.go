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

var forbiddenMiddlewarePatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.Use\([^)]*,`),
	regexp.MustCompile(`Group\([^)]*\)\.Use\(`),
}

func TestRepoUsesClassicMiddlewareRegistration(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(file)

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
		switch filepath.Base(path) {
		case "middleware_style_test.go", "plan_verification_test.go":
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		for _, re := range forbiddenMiddlewarePatterns {
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