#!/usr/bin/env bash
# Capture benchmark evidence for Arrow perf goal verification.
#
# Writes exactly four files to SCRATCH (same command for all):
#   go test -bench=. -benchmem -count=1 -run='^$' ./...
#
#   bench_before.log — baseline commit (default 95a1c24) with HEAD bench overlay
#   bench_after.log  — optimized code at REPO_ROOT (HEAD working tree)
#   bench_run1.log   — repeatability on HEAD
#   bench_run2.log   — repeatability on HEAD
#
# The before run uses a detached worktree at BASELINE_COMMIT (full pre-opt tree)
# with only the benchmark harness and testdata overlaid from HEAD so the measured
# delta isolates hot-path optimizations, not bench fixture drift.
set -euo pipefail

SCRATCH="${SCRATCH:?set SCRATCH to the goal scratch directory}"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BASELINE_COMMIT="${BASELINE_COMMIT:-95a1c24}"
BENCH_CMD=(go test -bench=. -benchmem -count=1 -run='^$' ./...)
BENCH_OVERLAY_FILES=(bench_test.go bench_build_test.go bench_corpus_test.go router_hotpath_test.go)

cd "$REPO_ROOT"

overlay_bench_suite() {
	local dest=$1
	for f in "${BENCH_OVERLAY_FILES[@]}"; do
		cp "$REPO_ROOT/$f" "$dest/$f"
	done
	rm -rf "$dest/testdata/bench"
	cp -R "$REPO_ROOT/testdata/bench" "$dest/testdata/bench"
}

WT="${SCRATCH}/bench-baseline-wt"
rm -rf "$WT"
git worktree add -q "$WT" "$BASELINE_COMMIT"

echo "==> bench_before.log (baseline ${BASELINE_COMMIT} + HEAD bench overlay)"
overlay_bench_suite "$WT"
( cd "$WT" && "${BENCH_CMD[@]}" 2>&1 ) | tee "${SCRATCH}/bench_before.log"
git worktree remove -f "$WT"

echo "==> bench_after.log (HEAD optimized hot path)"
"${BENCH_CMD[@]}" 2>&1 | tee "${SCRATCH}/bench_after.log"

echo "==> bench_run1.log"
"${BENCH_CMD[@]}" 2>&1 | tee "${SCRATCH}/bench_run1.log"

echo "==> bench_run2.log"
"${BENCH_CMD[@]}" 2>&1 | tee "${SCRATCH}/bench_run2.log"

echo "Done. Evidence in ${SCRATCH}"