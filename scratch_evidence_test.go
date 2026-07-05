package arrow_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var requiredVerificationTests = []string{
	"TestCompileRejectsUseChaining",
	"TestCompileRejectsGroupUseAssignment",
	"TestCompileRejectsGroupUseStatement",
	"TestPlanVerificationNoVariadicUseOutsideTestdata",
	"TestPlanVerificationNoGroupUseOutsideTestdata",
	"TestRepoUsesClassicMiddlewareRegistration",
}

var requiredVerificationPackages = []string{
	"github.com/fun7257/arrow",
	"github.com/fun7257/arrow/target",
}

func TestGenerateVerificationTestLog(t *testing.T) {
	scratch := os.Getenv("ARROW_VERIFY_SCRATCH")
	if scratch == "" {
		t.Skip("ARROW_VERIFY_SCRATCH not set")
	}
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		t.Fatal(err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(file)
	logPath := filepath.Join(scratch, "test.log")

	cmd := exec.Command("go", "test", "./...", "-count=1", "-v", "-skip", "TestGenerateVerificationTestLog")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if writeErr := os.WriteFile(logPath, out, 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	if err != nil {
		t.Fatalf("go test ./...: %v\n%s", err, out)
	}

	log := string(out)
	for _, name := range requiredVerificationTests {
		if !strings.Contains(log, "--- PASS: "+name+" ") {
			t.Fatalf("test.log missing PASS for %s", name)
		}
	}
	for _, pkg := range requiredVerificationPackages {
		found := false
		for _, line := range strings.Split(log, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "ok") && strings.Contains(line, pkg) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("test.log missing ok for %s", pkg)
		}
	}
}

func TestWriteScratchCompileEvidence(t *testing.T) {
	scratch := os.Getenv("ARROW_VERIFY_SCRATCH")
	if scratch == "" {
		t.Skip("ARROW_VERIFY_SCRATCH not set")
	}
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		t.Fatal(err)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(file)
	logPath := filepath.Join(scratch, "compile-evidence.log")

	var log strings.Builder
	for _, dir := range []string{"use_chain", "group_use_assign", "group_use_stmt"} {
		target := filepath.Join(root, "testdata", "compile", dir)
		fmt.Fprintf(&log, "--- testdata/compile/%s ---\n", dir)
		fmt.Fprintf(&log, "$ cd testdata/compile/%s && go build .\n", dir)

		var buf bytes.Buffer
		cmd := exec.Command("go", "build", ".")
		cmd.Dir = target
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		_, isExit := err.(*exec.ExitError)
		if err == nil || !isExit {
			t.Fatalf("compile %s: want exit error, got %v\n%s", dir, err, buf.String())
		}
		log.WriteString(buf.String())
		fmt.Fprintf(&log, "exit_code=1\n")
	}

	if err := os.WriteFile(logPath, []byte(log.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}
