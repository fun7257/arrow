#!/usr/bin/env bash
# Verification for classic single-arg Use middleware registration.
# Usage (from repo root): bash scripts/verify_classic_use.sh [SCRATCH_DIR]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRATCH="${1:-$ROOT}"
cd "$ROOT"

GREP_LOG="$SCRATCH/grep-verification.log"
: >"$GREP_LOG"

echo "=== step 1: middleware.go Use signature ===" | tee -a "$GREP_LOG"
grep -n 'func (r \*Router) Use' middleware.go | tee -a "$GREP_LOG"

echo "=== step 2a: variadic Use grep (exclude testdata) ===" | tee -a "$GREP_LOG"
if grep -rn 'Use(middleware\.Recover(),' --include='*.go' --include='*.md' . --exclude-dir=testdata | tee -a "$GREP_LOG"; then
  echo "FAIL: variadic Use found" | tee -a "$GREP_LOG" >&2
  exit 1
else
  echo "OK: no variadic Use" | tee -a "$GREP_LOG"
fi

echo "=== step 2b: Group().Use grep (exclude testdata) ===" | tee -a "$GREP_LOG"
if grep -rnE 'Group\([^)]*\)\.Use\(' --include='*.go' --include='*.md' . --exclude-dir=testdata | tee -a "$GREP_LOG"; then
  echo "FAIL: Group().Use pattern found outside testdata" | tee -a "$GREP_LOG" >&2
  exit 1
else
  echo "OK: no Group().Use outside testdata" | tee -a "$GREP_LOG"
fi

echo "=== step 3: go test ===" | tee -a "$GREP_LOG"
go test ./... -count=1 -v 2>&1 | tee "$SCRATCH/test.log"

echo "=== step 4: examples/server + README samples ===" | tee -a "$GREP_LOG"
grep -n 'app.Use' examples/server/main.go README.md | tee -a "$GREP_LOG"

echo "=== verification complete ===" | tee -a "$GREP_LOG"
