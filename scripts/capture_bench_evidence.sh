#!/usr/bin/env bash
# Capture benchmark evidence for Arrow perf goal verification.
# Writes exactly four files to SCRATCH:
#   bench_before.log  — baseline commit (default 95a1c24)
#   bench_after.log   — current HEAD
#   bench_run1.log    — full suite snapshot
#   bench_run2.log    — full suite snapshot (repeatability)
set -euo pipefail

SCRATCH="${SCRATCH:?set SCRATCH to the goal scratch directory}"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BASELINE_COMMIT="${BASELINE_COMMIT:-95a1c24}"
GATE_BENCH=(-bench='BenchmarkArrow_(Minimal|Static)' -benchmem -count=5 -run='^$' ./...)
FULL_BENCH=(-bench=. -benchmem -count=1 -run='^$' ./...)

cd "$REPO_ROOT"

WT="${SCRATCH}/bench-baseline-wt"
rm -rf "$WT"
git worktree add -q "$WT" "$BASELINE_COMMIT"
echo "==> bench_before.log (baseline ${BASELINE_COMMIT})"
( cd "$WT" && go test "${GATE_BENCH[@]}" 2>&1 ) | tee "${SCRATCH}/bench_before.log"
git worktree remove -f "$WT"

echo "==> bench_after.log (HEAD)"
go test "${GATE_BENCH[@]}" 2>&1 | tee "${SCRATCH}/bench_after.log"

echo "==> bench_run1.log (full suite)"
go test "${FULL_BENCH[@]}" 2>&1 | tee "${SCRATCH}/bench_run1.log"

echo "==> bench_run2.log (full suite)"
go test "${FULL_BENCH[@]}" 2>&1 | tee "${SCRATCH}/bench_run2.log"

echo "Done. Evidence in ${SCRATCH}"