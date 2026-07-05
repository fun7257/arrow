#!/usr/bin/env bash
# Verification for classic single-arg Use middleware registration.
# Usage (from repo root): bash scripts/verify_classic_use.sh [SCRATCH_DIR]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRATCH="${1:-$ROOT}"
cd "$ROOT"

GREP_LOG="$SCRATCH/grep-verification.log"
AUDIT_LOG="$SCRATCH/use-audit.log"
COMPILE_LOG="$SCRATCH/compile-evidence.log"
: >"$GREP_LOG"
: >"$AUDIT_LOG"
: >"$COMPILE_LOG"

log() { echo "$@" | tee -a "$GREP_LOG"; }

log "=== step 1: middleware.go Use signature ==="
log '$ grep -n '\''func (r \*Router) Use'\'' middleware.go'
grep -n 'func (r \*Router) Use' middleware.go | tee -a "$GREP_LOG"

log ""
log "=== step 2a: variadic Use grep (exclude testdata) ==="
log '$ grep -rn '\''Use(middleware\.Recover(),'\'' --include='\''*.go'\'' --include='\''*.md'\'' . --exclude-dir=testdata'
set +e
grep -rn 'Use(middleware\.Recover(),' --include='*.go' --include='*.md' . --exclude-dir=testdata | tee -a "$GREP_LOG"
var_exit=$?
set -e
log "exit_code=$var_exit"
if [ "$var_exit" -eq 0 ]; then
  log "FAIL: variadic Use found" >&2
  exit 1
else
  log "OK: no variadic Use (grep exit $var_exit)"
fi

log ""
log "=== step 2b: Group().Use grep (exclude testdata) ==="
log '$ grep -rnE '\''Group\([^)]*\)\.Use\('\'' --include='\''*.go'\'' --include='\''*.md'\'' . --exclude-dir=testdata'
set +e
grep -rnE 'Group\([^)]*\)\.Use\(' --include='*.go' --include='*.md' . --exclude-dir=testdata | tee -a "$GREP_LOG"
grp_exit=$?
set -e
log "exit_code=$grp_exit"
if [ "$grp_exit" -eq 0 ]; then
  log "FAIL: Group().Use pattern found outside testdata" >&2
  exit 1
else
  log "OK: no Group().Use outside testdata (grep exit $grp_exit)"
fi

log ""
log "=== step 2c: use-site audit (exclude testdata) ==="
log '$ grep -rn '\''\.Use('\'' --include='\''*.go'\'' . --exclude-dir=testdata'
grep -rn '\.Use(' --include='*.go' . --exclude-dir=testdata | tee -a "$AUDIT_LOG"
log "audit lines: $(wc -l < "$AUDIT_LOG" | tr -d ' ')"

log ""
log "=== step 3: go test (verbose) ==="
log '$ go test ./... -count=1 -v 2>&1 | tee '"$SCRATCH/test.log"
go test ./... -count=1 -v 2>&1 | tee "$SCRATCH/test.log"
test_lines=$(wc -l < "$SCRATCH/test.log" | tr -d ' ')
log "test.log lines=$test_lines"
grep -E '^(=== RUN|--- PASS|ok  )' "$SCRATCH/test.log" | grep -E 'TestCompile|TestGroup|TestPipeline' | tee -a "$GREP_LOG" || true

log ""
log "=== step 4: compile-fail fixtures (group_use_stmt must fail) ==="
for dir in use_chain group_use_assign group_use_stmt; do
  log "--- testdata/compile/$dir ---"
  log '$ cd testdata/compile/'"$dir"' && go build .'
  set +e
  (cd "testdata/compile/$dir" && go build .) 2>&1 | tee -a "$COMPILE_LOG"
  build_exit=$?
  set -e
  log "exit_code=$build_exit"
done

log ""
log "=== step 5: examples/server + README samples ==="
grep -n 'app.Use' examples/server/main.go README.md | tee -a "$GREP_LOG"

log ""
log "=== verification complete ==="
