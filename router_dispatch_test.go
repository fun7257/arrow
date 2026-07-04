package arrow

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestRouterZeroMiddlewareUsesInlineClosure(t *testing.T) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "router.go"))
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}
	src := string(data)

	zeroMW := regexp.MustCompile(`if len\(r\.pipe\.middlewares\) == 0 \{[\s\S]*?return\n\t\}`)
	matches := zeroMW.FindAllString(src, -1)
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 zero-middleware branches in router.go, got %d", len(matches))
	}
	for i, block := range matches {
		if strings.Contains(block, "runNoMiddleware") {
			t.Fatalf("zero-mw branch %d must not call runNoMiddleware:\n%s", i+1, block)
		}
		if !strings.Contains(block, "defer recoverAndRelease(ctx)") {
			t.Fatalf("zero-mw branch %d must inline defer recoverAndRelease:\n%s", i+1, block)
		}
		if !strings.Contains(block, "ctx.afters") {
			t.Fatalf("zero-mw branch %d must run After callbacks:\n%s", i+1, block)
		}
	}
}