#!/usr/bin/env bash
# Capture benchmark evidence for Arrow perf goal verification.
#
# Writes to SCRATCH:
#   bench_before.log + bench_before.frozen.log — once only (95a1c24 full tree)
#   bench_after.log, bench_run1.log, bench_run2.log — every run (HEAD)
#
# All use: go test -bench=. -benchmem -count=1 -run='^$' ./...
set -euo pipefail

SCRATCH="${SCRATCH:?set SCRATCH to the goal scratch directory}"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BASELINE_COMMIT="${BASELINE_COMMIT:-95a1c24}"
BENCH_CMD=(go test -bench=. -benchmem -count=1 -run='^$' ./...)

cd "$REPO_ROOT"

if [[ ! -f "${SCRATCH}/bench_before.frozen.log" ]]; then
	echo "==> bench_before.log (one-time baseline ${BASELINE_COMMIT}, full tree)"
	WT="${SCRATCH}/bench-baseline-wt"
	rm -rf "$WT"
	git worktree add -q "$WT" "$BASELINE_COMMIT"
	( cd "$WT" && "${BENCH_CMD[@]}" 2>&1 ) | tee "${SCRATCH}/bench_before.log"
	cp "${SCRATCH}/bench_before.log" "${SCRATCH}/bench_before.frozen.log"
	git worktree remove -f "$WT"
	echo "Frozen baseline saved to bench_before.frozen.log"
else
	echo "==> bench_before.frozen.log exists; skipping baseline re-capture"
	cp "${SCRATCH}/bench_before.frozen.log" "${SCRATCH}/bench_before.log"
fi

echo "==> bench_after.log (HEAD optimized)"
"${BENCH_CMD[@]}" 2>&1 | tee "${SCRATCH}/bench_after.log"

echo "==> bench_run1.log"
"${BENCH_CMD[@]}" 2>&1 | tee "${SCRATCH}/bench_run1.log"

echo "==> bench_run2.log"
"${BENCH_CMD[@]}" 2>&1 | tee "${SCRATCH}/bench_run2.log"

echo "Done. Evidence in ${SCRATCH}"