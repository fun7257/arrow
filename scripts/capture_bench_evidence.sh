#!/usr/bin/env bash
# Capture benchmark evidence for Arrow perf goal verification.
#
# Writes to SCRATCH:
#   bench_before.log + bench_before.frozen.log — once only (95a1c24 hot-path on HEAD bench)
#   bench_after.log, bench_run1.log, bench_run2.log — every run (HEAD)
#
# Baseline swaps only hot-path files from BASELINE_COMMIT onto HEAD so the
# comparison uses the same bench suite and fixtures.
#
# All use: go test -bench=. -benchmem -count=1 -run='^$' ./...
set -euo pipefail

SCRATCH="${SCRATCH:?set SCRATCH to the goal scratch directory}"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BASELINE_COMMIT="${BASELINE_COMMIT:-95a1c24}"
BENCH_CMD=(go test -bench=. -benchmem -count=1 -run='^$' ./...)
HOTPATH_FILES=(router.go pipeline.go context.go pool.go writer_wrap.go)
# HEAD-only tests that reference symbols absent from the baseline hot path.
BASELINE_STASH_TESTS=(pipeline_internal_test.go)

cd "$REPO_ROOT"

restore_hotpath() {
	git checkout HEAD -- "${HOTPATH_FILES[@]}"
	for f in "${BASELINE_STASH_TESTS[@]}"; do
		if [[ -f "${SCRATCH}/hotpath-stash/${f}.head-test" ]]; then
			mv "${SCRATCH}/hotpath-stash/${f}.head-test" "$f"
		fi
	done
}

swap_baseline_hotpath() {
	mkdir -p "${SCRATCH}/hotpath-stash"
	for f in "${HOTPATH_FILES[@]}"; do
		cp "$f" "${SCRATCH}/hotpath-stash/${f}.head"
	done
	for f in "${BASELINE_STASH_TESTS[@]}"; do
		if [[ -f $f ]]; then
			mv "$f" "${SCRATCH}/hotpath-stash/${f}.head-test"
		fi
	done
	git checkout "${BASELINE_COMMIT}" -- "${HOTPATH_FILES[@]}"
}

if [[ ! -f "${SCRATCH}/bench_before.frozen.log" ]]; then
	echo "==> bench_before.log (one-time baseline ${BASELINE_COMMIT} hot-path on HEAD bench)"
	trap restore_hotpath EXIT
	swap_baseline_hotpath
	( "${BENCH_CMD[@]}" 2>&1 ) | tee "${SCRATCH}/bench_before.log"
	trap - EXIT
	restore_hotpath
	cp "${SCRATCH}/bench_before.log" "${SCRATCH}/bench_before.frozen.log"
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