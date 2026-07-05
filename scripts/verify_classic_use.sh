#!/usr/bin/env bash
# Verification for classic single-arg Use middleware registration.
# Usage (from repo root): bash scripts/verify_classic_use.sh [SCRATCH_DIR]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRATCH="${1:-$ROOT}"
cd "$ROOT"

GREP_LOG="$SCRATCH/grep-verification.log"
AUDIT_LOG="$SCRATCH/use-audit.log"
: >"$GREP_LOG"
: >"$AUDIT_LOG"

log() { echo "$@" | tee -a "$GREP_LOG"; }

log "=== step 1: middleware.go Use signature ==="
log '$ grep -n '\''func (r \*Router) Use'\'' middleware.go'
grep -n 'func (r \*Router) Use' middleware.go | tee -a "$GREP_LOG"

log ""
log "=== step 2a: variadic Use grep ==="
log '$ grep -rn '\''Use(middleware\.Recover(),'\'' --include='\''*.go'\'' --include='\''*.md'\'' .'
set +e
grep -rn 'Use(middleware\.Recover(),' --include='*.go' --include='*.md' . | tee -a "$GREP_LOG"
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
log "=== step 2b: Group().Use grep ==="
log '$ grep -rnE '\''Group\([^)]*\)\.Use\('\'' --include='\''*.go'\'' --include='\''*.md'\'' .'
set +e
grep -rnE 'Group\([^)]*\)\.Use\(' --include='*.go' --include='*.md' . | tee -a "$GREP_LOG"
grp_exit=$?
set -e
log "exit_code=$grp_exit"
if [ "$grp_exit" -eq 0 ]; then
  log "FAIL: Group().Use pattern found" >&2
  exit 1
else
  log "OK: no Group().Use (grep exit $grp_exit)"
fi

log ""
log "=== step 2c: use-site audit ==="
log '$ grep -rn '\''\.Use('\'' --include='\''*.go'\'' .'
grep -rn '\.Use(' --include='*.go' . | tee -a "$AUDIT_LOG"
log "audit lines: $(wc -l < "$AUDIT_LOG" | tr -d ' ')"

log ""
log "=== step 3: go test (verbose) ==="
log '$ ARROW_VERIFY_SCRATCH='"$SCRATCH"' go test -run TestGenerateVerificationTestLog -count=1 -v'
export ARROW_VERIFY_SCRATCH="$SCRATCH"
go test -run TestGenerateVerificationTestLog -count=1 -v 2>&1 | tee -a "$GREP_LOG"
test_lines=$(wc -l < "$SCRATCH/test.log" | tr -d ' ')
log "test.log lines=$test_lines"
grep -E '^(=== RUN|--- PASS|ok  )' "$SCRATCH/test.log" | grep -E 'TestPlanVerification|TestRepoUsesClassic|TestGroup|TestPipeline' | tee -a "$GREP_LOG" || true
for required in \
  TestPlanVerificationNoVariadicUse \
  TestPlanVerificationNoGroupUseChaining \
  TestRepoUsesClassicMiddlewareRegistration; do
  if ! grep -qF -- "--- PASS: ${required} " "$SCRATCH/test.log"; then
    log "FAIL: missing PASS for ${required} in test.log" >&2
    exit 1
  fi
done
for pkg in 'github.com/fun7257/arrow' 'github.com/fun7257/arrow/target'; do
  if ! grep -Eq "^ok[[:space:]]+${pkg}([[:space:]]|$)" "$SCRATCH/test.log"; then
    log "FAIL: missing ok for ${pkg} in test.log" >&2
    exit 1
  fi
done

log ""
log "=== step 4: examples/server + README samples ==="
grep -n 'app.Use' examples/server/main.go README.md | tee -a "$GREP_LOG"

log ""
log "=== verification complete ==="