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

// TestWriteScratchCompileEvidence records compile-fail fixture output to
// ARROW_VERIFY_SCRATCH/compile-evidence.log when ARROW_VERIFY_SCRATCH is set.
// verify_classic_use.sh step 4 also captures this; this test drives the same
// go build entry points as compile_api_test.go.
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
