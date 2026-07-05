package arrow_test

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompileRejectsUseChaining(t *testing.T) {
	out := testCompileFails(t, "use_chain")
	requireCompileDiagnostic(t, out,
		"(no value) used as value",
		"must reject void-return Use chaining, not variadic signature",
		"too many arguments",
	)
}

func TestCompileRejectsGroupUseAssignment(t *testing.T) {
	out := testCompileFails(t, "group_use_assign")
	requireCompileDiagnostic(t, out,
		"(no value) used as value",
		"must reject assigning void-return group Use",
		"too many arguments",
	)
}

func TestCompileRejectsGroupUseStatement(t *testing.T) {
	out := testCompileFails(t, "group_use_stmt")
	requireCompileDiagnostic(t, out,
		"cannot call pointer method",
		"must reject non-addressable Group().Use statement",
		"used as value",
	)
	if !strings.Contains(out, "groupRoutes") {
		t.Fatalf("compile output = %q, want diagnostic mentioning groupRoutes", out)
	}
}

func requireCompileDiagnostic(t *testing.T, out string, want string, forbidMsg string, forbid ...string) {
	t.Helper()
	if !strings.Contains(out, want) {
		t.Fatalf("compile output = %q, want diagnostic containing %q", out, want)
	}
	for _, bad := range forbid {
		if strings.Contains(out, bad) {
			t.Fatalf("compile output = %q, %s (found %q)", out, forbidMsg, bad)
		}
	}
}

func testCompileFails(t *testing.T, dir string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(file)
	target := filepath.Join(root, "testdata", "compile", dir)

	var buf bytes.Buffer
	cmd := exec.Command("go", "build", "-buildvcs=false", ".")
	cmd.Dir = target
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err == nil {
		t.Fatalf("expected compile failure in %s", target)
	} else if _, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("expected exit error from go build in %s, got %T: %v\noutput: %s", target, err, err, buf.String())
	}
	return buf.String()
}
