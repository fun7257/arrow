#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRATCH="${1:-$ROOT}"
cd "$ROOT"
GREP_LOG="$SCRATCH/grep-verification.log"
AUDIT_LOG="$SCRATCH/use-audit.log"
COMPILE_LOG="$SCRATCH/compile-evidence.log"
: >"$GREP_LOG"; : >"$AUDIT_LOG"; : >"$COMPILE_LOG"
log() { echo "$@" | tee -a "$GREP_LOG"; }
log "=== step 1: middleware.go Use signature ==="
grep -n 'func (r \*Router) Use' middleware.go | tee -a "$GREP_LOG"
log "=== step 2a: variadic Use grep ==="
set +e; grep -rn 'Use(middleware\.Recover(),' --include='*.go' --include='*.md' . --exclude-dir=testdata | tee -a "$GREP_LOG"; var_exit=$?; set -e
[ "$var_exit" -eq 0 ] && { log "FAIL: variadic Use found"; exit 1; } || log "OK: no variadic Use"
log "=== step 2b: Group().Use grep ==="
set +e; grep -rnE 'Group\([^)]*\)\.Use\(' --include='*.go' --include='*.md' . --exclude-dir=testdata | tee -a "$GREP_LOG"; grp_exit=$?; set -e
[ "$grp_exit" -eq 0 ] && { log "FAIL: Group().Use found"; exit 1; } || log "OK: no Group().Use outside testdata"
grep -rn '\.Use(' --include='*.go' . --exclude-dir=testdata | tee -a "$AUDIT_LOG"
log "=== step 3: go test ==="
export ARROW_VERIFY_SCRATCH="$SCRATCH"
go test -run TestGenerateVerificationTestLog -count=1 -v 2>&1 | tee -a "$GREP_LOG"
for required in TestCompileRejectsUseChaining TestCompileRejectsGroupUseAssignment TestCompileRejectsGroupUseStatement TestPlanVerificationNoVariadicUseOutsideTestdata TestPlanVerificationNoGroupUseOutsideTestdata TestRepoUsesClassicMiddlewareRegistration; do
  grep -qF -- "--- PASS: ${required} " "$SCRATCH/test.log" || { log "FAIL: missing ${required}"; exit 1; }
done
for pkg in github.com/fun7257/arrow github.com/fun7257/arrow/target; do
  grep -qF -- "ok  ${pkg}" "$SCRATCH/test.log" || { log "FAIL: missing ok ${pkg}"; exit 1; }
done
log "=== step 4: compile fixtures ==="
for dir in use_chain group_use_assign group_use_stmt; do
  set +e; (cd "testdata/compile/$dir" && go build .) 2>&1 | tee -a "$COMPILE_LOG"; set -e
done
log "=== verification complete ==="
